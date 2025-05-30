package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/lib/logger/sl"
	"premium_caste/internal/transport/http/dto"
	"premium_caste/internal/transport/http/dto/request"
	"premium_caste/internal/transport/http/dto/response"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	_ "premium_caste/docs"
)

type UserService interface {
	Login(ctx context.Context, c echo.Context, email, password string) (*models.TokenPair, error)
	RegisterNewUser(ctx context.Context, input dto.UserRegisterInput) (uuid.UUID, error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	GetUserById(ctx context.Context, userID uuid.UUID) (models.User, error)
}

type MediaService interface {
	UploadMedia(ctx context.Context, input dto.MediaUploadInput) (*models.Media, error)
	AttachMediaToGroup(ctx context.Context, groupID uuid.UUID, mediaID uuid.UUID) error
	AttachMedia(ctx context.Context, ownerID uuid.UUID, description string) error
	ListGroupMedia(ctx context.Context, groupID uuid.UUID) ([]models.Media, error)
}

type AuthService interface {
	GenerateTokens(user models.User) (*models.TokenPair, error)
	RefreshTokens(refreshToken string) (*models.TokenPair, error)
}

type BlogService interface {
	CreatePost(ctx context.Context, req dto.CreateBlogPostRequest) (*dto.BlogPostResponse, error)
	UpdatePost(ctx context.Context, postID uuid.UUID, req dto.UpdateBlogPostRequest) (*dto.BlogPostResponse, error)
	GetPostByID(ctx context.Context, id uuid.UUID) (*dto.BlogPostResponse, error)
	PublishPost(ctx context.Context, postID uuid.UUID) (*dto.BlogPostResponse, error)
	ArchivePost(ctx context.Context, postID uuid.UUID) (*dto.BlogPostResponse, error)
	DeletePost(ctx context.Context, postID uuid.UUID) error
	AddMediaGroup(ctx context.Context, postID uuid.UUID, req dto.AddMediaGroupRequest) (*dto.PostMediaGroupsResponse, error)
	ListPosts(ctx context.Context, statusFilter string, page, perPage int) (*dto.BlogPostListResponse, error)
	GetPostMediaGroups(ctx context.Context, postID uuid.UUID, relationType string) (*dto.PostMediaGroupsResponse, error)
}

type Routers struct {
	log          *slog.Logger
	UserService  UserService
	MediaService MediaService
	AuthService  AuthService
	BlogService  BlogService
}

func NewRouter(log *slog.Logger, userService UserService, mediaService MediaService, authService AuthService, blogService BlogService) *Routers {
	return &Routers{
		log:          log,
		UserService:  userService,
		MediaService: mediaService,
		AuthService:  authService,
		BlogService:  blogService,
	}
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExist          = errors.New("user already exist")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidUUID        = errors.New("not valid UUID")
)

// Login godoc
// @Summary Аутентификация пользователя
// @Description Вход в систему по email и паролю. Возвращает JWT-токен.
// @Tags users
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "Данные для входа"
// @Success 200 {object} response.Response{data=map[string]string} "Успешный вход (токен)"
// @Failure 400 {object} response.ErrorResponse "Неверный формат запроса"
// @Failure 401 {object} response.ErrorResponse "Ошибка аутентификации"
// @Router /api/v1/login [post]
func (r *Routers) Login(c echo.Context) error {
	const op = "http.routers.Login"

	log := r.log.With(
		slog.String("op", op),
	)

	var req request.LoginRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.ErrInvalidRequestFormat)
	}

	if err := c.Validate(req); err != nil {
		response.ErrInvalidRequestFormat.Details = err.Error()
		log.Warn("invalid format request", slog.String("identifier", req.Identifier))
		return c.JSON(http.StatusBadRequest, response.ErrInvalidRequestFormat)
	}

	token, err := r.UserService.Login(c.Request().Context(), c, req.Identifier, req.Password)
	if err != nil {
		response.ErrAuthenticationFailed.Details = err.Error()
		return c.JSON(http.StatusUnauthorized, response.ErrAuthenticationFailed)
	}

	return c.JSON(http.StatusOK, response.Response{
		Status: "success",
		Data:   map[string]string{"user_id": token.UserID.String(), "access_token": token.AccessToken, "refresh_token": token.RefreshToken},
	})
}

// Register godoc
// @Summary Регистрация нового пользователя
// @Description Создание аккаунта. Возвращает ID пользователя.
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.UserRegisterInput true "Данные для регистрации"
// @Success 201 {object} response.Response{data=object{user_id=string}} "Успешная регистрация"
// @Failure 400 {object} response.ErrorResponse "Неверный формат запроса"
// @Failure 409 {object} response.ErrorResponse "Пользователь уже существует"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/register [post]
func (r *Routers) Register(c echo.Context) error {
	const op = "http.routers.Register"

	log := r.log.With(
		slog.String("op", op),
	)

	var req dto.UserRegisterInput

	if err := c.Bind(&req); err != nil {
		r.log.Error("failed to bind request", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrInvalidRegisterRequest)
	}

	if err := c.Validate(req); err != nil {
		response.ErrInvalidRegisterRequest.Details = err.Error()
		r.log.Error("validation failed", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrInvalidRegisterRequest)
	}

	userID, err := r.UserService.RegisterNewUser(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrUserExist) {
			log.Warn("user already exists", slog.String("email", req.Email))
			return c.JSON(http.StatusConflict, response.ErrUserAlreadyExists)
		}

		log.Error("registration failed", sl.Err(err))
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Status:  "error",
			Error:   "internal_error",
			Details: "Internal server error",
		})
	}

	log.Info("user registered successfully", slog.String("user_id", userID.String()))

	return c.JSON(http.StatusCreated, response.Response{
		Status: "success",
		Data: map[string]uuid.UUID{
			"user_id": userID,
		},
	})
}

func (r *Routers) Refresh(c echo.Context) error {
	const op = "http.routers.Refresh"

	log := r.log.With(
		slog.String("op", op),
	)

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("validation bind", sl.Err(err))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	newTokens, err := r.AuthService.RefreshTokens(req.RefreshToken)
	if err != nil {
		log.Error("error refresh tokens", sl.Err(err))
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
	}

	return c.JSON(http.StatusOK, newTokens)
}

// IsAdminPermission
// @Summary Проверка административного статуса пользователя
// @Description Проверяет, является ли указанный пользователь администратором
// @Tags Users
// @Accept  json
// @Produce  json
// @Param user_id path string true "UUID пользователя" format(uuid)
// @Success 200 {object} map[string]bool "Результат проверки" example({"is_admin": true})
// @Failure 400 {object} map[string]string "Невалидный UUID" example({"error": "invalid user ID format"})
// @Failure 500 {object} map[string]string "Ошибка сервера" example({"error": "failed to check admin status"})
// @Security ApiKeyAuth
// @Router /api/v1/users/{user_id}/is-admin [get]
func (r *Routers) IsAdminPermission(c echo.Context) error {
	const op = "http.routers.IsAdminPermission"

	log := r.log.With(
		slog.String("op", op),
	)

	userIDStr := c.Param("user_id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Error("error parse uuid", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: "invalid user ID format",
		})
	}

	isAdmin, err := r.UserService.IsAdmin(c.Request().Context(), userID)
	if err != nil {
		log.Error("failed to check admin status",
			slog.String("error", err.Error()),
			slog.Any("user_id", userID),
		)
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: "failed to check admin status",
		})
	}

	sess, _ := session.Get("session", c)
	sess.Values["user_id"] = userID.String()
	sess.Save(c.Request(), c.Response())

	return c.JSON(http.StatusOK, map[string]bool{
		"is_admin": isAdmin,
	})
}

// GetUserById godoc
// @Summary Получение информации о пользователе
// @Description Возвращает полную информацию о пользователе по его UUID
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param user_id body string true "UUID пользователя" format(uuid) example("a8a8a8a8-a8a8-a8a8-a8a8-a8a8a8a8a8a8")
// @Success 200 {object} models.User "Успешно полученные данные пользователя"
// @Failure 400 {object} response.ErrorResponse "Некорректный UUID пользователя"
// @Failure 404 {object} response.ErrorResponse "Пользователь не найден"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /api/v1/users/get [post]
func (r *Routers) GetUserById(c echo.Context) error {
	const op = "http.routers.GetUserById"

	log := r.log.With(
		slog.String("op", op),
	)

	var req struct {
		UserID uuid.UUID `json:"user_id"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("validation bind", sl.Err(err))
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: "validation bind",
		})
	}

	user, err := r.UserService.GetUserById(c.Request().Context(), req.UserID)
	if err != nil {
		log.Error("error get user", sl.Err(err))
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: "failed get user",
		})

	}

	return c.JSON(http.StatusOK, user)
}

// UploadMedia godoc
// @Summary Загрузка медиафайла
// @Description Загружает файл на сервер с возможностью указания метаданных
// @Tags Медиа
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл для загрузки (макс. 10MB)"
// @Param uploader_id formData string true "UUID пользователя-загрузчика" format(uuid)
// @Param media_type formData string true "Тип контента" Enums(photo, video, audio, document)
// @Param is_public formData boolean false "Публичный доступ (по умолчанию false)"
// @Param metadata formData string false "Дополнительные метаданные в JSON-формате"
// @Param width formData integer false "Ширина в пикселях (для изображений/видео)"
// @Param height formData integer false "Высота в пикселях (для изображений/видео)"
// @Param duration formData integer false "Длительность в секундах (для видео/аудио)"
// @Success 201 {object} models.Media "Успешно загруженный медиаобъект"
// @Failure 400 {object} response.ErrorResponse "Некорректные входные данные"
// @Failure 413 {object} response.ErrorResponse "Превышен максимальный размер файла"
// @Failure 415 {object} response.ErrorResponse "Неподдерживаемый тип файла"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Security ApiKeyAuth
// @Router /api/v1/media/upload [post]
func (r *Routers) UploadMedia(c echo.Context) error {
	const op = "http.routers.UploadMedia"

	log := r.log.With(
		slog.String("op", op),
	)

	startTime := time.Now()
	defer func() {
		log.Info("Request completed",
			"duration", time.Since(startTime))
	}()

	log.Info("Start uploading media",
		"method", c.Request().Method,
		"path", c.Path(),
		"client_ip", c.RealIP())

	file, err := c.FormFile("file")
	if err != nil {
		log.Warn("Empty file in request",
			"error", err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "File is required"})
	}

	log.Debug("Got file for upload",
		"filename", file.Filename,
		"size", file.Size,
		"mime_type", file.Header.Get("Content-Type"))

	input, err := r.parseMediaUploadInput(c)
	if err != nil {
		log.Warn("Error parsing data",
			"error", err.Error(),
			"uploader_id", c.FormValue("uploader_id"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	input.File = file

	log.Debug("Options for upload",
		"uploader_id", input.UploaderID,
		"media_type", input.MediaType,
		"is_public", input.IsPublic,
		"metadata", input.CustomMetadata)

	media, err := r.MediaService.UploadMedia(c.Request().Context(), *input)
	if err != nil {
		log.Error("Error upload media",
			"error", err.Error(),
			"uploader_id", input.UploaderID,
			"filename", file.Filename)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Info("Upload successfull",
		"media_id", media.ID,
		"uploader_id", media.UploaderID,
		"file_size", media.FileSize,
		"duration", time.Since(startTime))

	return c.JSON(http.StatusCreated, media)
}

// AttachMediaToGroup godoc
// @Summary Прикрепить медиа к группе
// @Description Связывает медиафайл с существующей медиагруппой
// @Tags Медиа-группы
// @Accept multipart/form-data
// @Produce json
// @Param group_id path string true "UUID группы" format(uuid)
// @Param media_id formData string true "UUID медиафайла" format(uuid)
// @Success 200 "Успешное прикрепление (no content)"
// @Failure 400 {object} response.ErrorResponse "Невалидные UUID группы или медиа"
// @Failure 500 {object} response.ErrorResponse "Ошибка привязки медиа"
// @Security ApiKeyAuth
// @Router /api/v1/media/groups/{group_id}/attach [post]
func (r *Routers) AttachMediaToGroup(c echo.Context) error {
	const op = "http.routers.AttachMediaToGroup"

	log := r.log.With(
		slog.String("op", op),
	)

	req := new(dto.AttachMediaRequest)
	if err := c.Bind(req); err != nil {
		log.Error("error invalid request data format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: "Invalid request data format",
		})
	}

	if err := c.Validate(req); err != nil {
		log.Error("error validate", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	groupID, _ := uuid.Parse(req.GroupID)
	mediaID, _ := uuid.Parse(req.MediaID)

	if err := r.MediaService.AttachMediaToGroup(
		c.Request().Context(),
		groupID,
		mediaID,
	); err != nil {
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.NoContent(http.StatusOK)
}

// CreateMediaGroup godoc
// @Summary Создать медиагруппу
// @Description Создает новую группу для организации медиафайлов
// @Tags Медиа-группы
// @Accept multipart/form-data
// @Produce json
// @Param owner_id formData string true "UUID владельца группы" format(uuid)
// @Param description formData string false "Описание группы"
// @Success 201 "Группа создана (no content)"
// @Failure 400 {object} response.ErrorResponse "Невалидный UUID владельца"
// @Failure 500 {object} response.ErrorResponse "Ошибка создания группы"
// @Security ApiKeyAuth
// @Router /api/v1/media/groups [post]
func (r *Routers) CreateMediaGroup(c echo.Context) error {
	const op = "http.routers.CreateMediaGroup"

	log := r.log.With(
		slog.String("op", op),
	)

	req := new(dto.CreateMediaGroupRequest)

	if err := c.Bind(req); err != nil {
		log.Error("error invalid request data", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: "Invalid request data",
		})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	ownerID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	if err := r.MediaService.AttachMedia(
		c.Request().Context(),
		ownerID,
		req.Description,
	); err != nil {
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.NoContent(http.StatusCreated)
}

// ListGroupMedia godoc
// @Summary Получить медиа группы
// @Description Возвращает список всех медиафайлов в группе
// @Tags Медиа-группы
// @Produce json
// @Param group_id path string true "UUID группы" format(uuid)
// @Success 200 {array} models.Media "Список медиафайлов"
// @Failure 400 {object} response.ErrorResponse "Невалидный UUID группы"
// @Failure 500 {object} response.ErrorResponse "Ошибка получения списка"
// @Security ApiKeyAuth
// @Router /api/v1/media/groups/{group_id} [get]
func (r *Routers) ListGroupMedia(c echo.Context) error {
	const op = "http.routers.ListGroupMedia"

	log := r.log.With(
		slog.String("op", op),
	)

	req := new(dto.ListGroupMediaRequest)
	if err := c.Bind(req); err != nil {
		log.Error("invalid query parameters", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: "Invalid query parameters",
		})
	}

	if err := c.Validate(req); err != nil {
		log.Error("validataion failed", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error:   "Validation failed",
			Details: err.Error(),
		})
	}

	groupID, err := uuid.Parse(req.GroupID)
	if err != nil {
		log.Error("failed parse uuid", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error:   "Faield parse",
			Details: err.Error(),
		})
	}

	media, err := r.MediaService.ListGroupMedia(c.Request().Context(), groupID)
	if err != nil {
		log.Error("failed list group", sl.Err(err))
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	response := map[string]interface{}{
		"data": media,
		"meta": map[string]interface{}{
			"count":    len(media),
			"group_id": groupID,
		},
	}

	return c.JSON(http.StatusOK, response)
}

func (r *Routers) parseMediaUploadInput(c echo.Context) (*dto.MediaUploadInput, error) {
	uploaderID, err := uuid.Parse(c.FormValue("uploader_id"))
	if err != nil {
		return nil, err
	}

	var metadata map[string]any
	if metaStr := c.FormValue("metadata"); metaStr != "" {
		if err := json.Unmarshal([]byte(metaStr), &metadata); err != nil {
			return nil, err
		}
	}

	input := &dto.MediaUploadInput{
		UploaderID:     uploaderID,
		MediaType:      c.FormValue("media_type"),
		IsPublic:       c.FormValue("is_public") == "true",
		CustomMetadata: metadata,
	}

	if widthStr := c.FormValue("width"); widthStr != "" {
		if width, err := strconv.Atoi(widthStr); err == nil {
			input.Width = &width
		}
	}
	if heightStr := c.FormValue("height"); heightStr != "" {
		if height, err := strconv.Atoi(heightStr); err == nil {
			input.Height = &height
		}
	}
	if durationStr := c.FormValue("duration"); durationStr != "" {
		if duration, err := strconv.Atoi(durationStr); err == nil {
			input.Duration = &duration
		}
	}

	return input, nil
}

// CreatePost godoc
// @Summary Создать новый пост
// @Description Создает новый пост блога. Добавлять только authod_id -> существующий пользователь. Добавлять FeaturedImageID -> только существующую медиа
// @Tags Посты
// @Accept json
// @Produce json
// @Param request body dto.CreateBlogPostRequest true "Данные поста"
// @Success 201 {object} dto.BlogPostResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts [post]
// Добавлять только authod_id -> существующий пользователь
// Добавлять FeaturedImageID -> только существующую медиа
func (r *Routers) CreatePost(c echo.Context) error {
	const op = "http.routers.CreatePost"

	log := r.log.With(
		slog.String("op", op),
	)

	var req dto.CreateBlogPostRequest

	if err := c.Bind(&req); err != nil {
		log.Error("invalid request data", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request data"})
	}

	// if err := c.Validate(req); err != nil {
	// 	log.Error("validation failed", sl.Err(err))
	// 	return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
	// }

	post, err := r.BlogService.CreatePost(c.Request().Context(), req)
	if err != nil {
		log.Error("error create post", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusCreated, post)
}

// GetPost godoc
// @Summary Получить пост
// @Description Возвращает пост по его ID
// @Tags Посты
// @Produce json
// @Param id path string true "UUID поста" format(uuid)
// @Success 200 {object} dto.BlogPostResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts/{id} [get]
func (r *Routers) GetPost(c echo.Context) error {
	const op = "http.routers.GetPost"

	log := r.log.With(
		slog.String("op", op),
	)

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error("invalid post id format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid post ID format"})
	}

	post, err := r.BlogService.GetPostByID(c.Request().Context(), postID)
	if err != nil {
		log.Error("error get post by id", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "error get post by id"})
	}

	return c.JSON(http.StatusOK, post)
}

// UpdatePost godoc
// @Summary Обновить пост
// @Description Обновляет данные поста
// @Tags Посты
// @Accept json
// @Produce json
// @Param id path string true "UUID поста" format(uuid)
// @Param request body dto.UpdateBlogPostRequest true "Данные для обновления"
// @Success 200 {object} dto.BlogPostResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts/{id} [put]
func (r *Routers) UpdatePost(c echo.Context) error {
	const op = "http.routers.UpdatePost"

	log := r.log.With(
		slog.String("op", op),
	)

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error("invalid post ID format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid post ID format"})
	}

	req := new(dto.UpdateBlogPostRequest)
	if err := c.Bind(req); err != nil {
		log.Error("invalid request data", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request data"})
	}

	// if err := c.Validate(req); err != nil {
	// 	log.Error("failed validate data", sl.Err(err))
	// 	return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
	// }

	post, err := r.BlogService.UpdatePost(c.Request().Context(), postID, *req)
	if err != nil {
		log.Error("failed update post", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "Error update post"})
	}

	return c.JSON(http.StatusOK, post)
}

// DeletePost godoc
// @Summary Удалить пост
// @Description Удаляет пост (физическое удаление)
// @Tags Посты
// @Param id path string true "UUID поста" format(uuid)
// @Success 204
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts/{id} [delete]
func (r *Routers) DeletePost(c echo.Context) error {
	const op = "http.routers.DeletePost"

	log := r.log.With(
		slog.String("op", op),
	)

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error("invalid post ID format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid post ID format"})
	}

	if err := r.BlogService.DeletePost(c.Request().Context(), postID); err != nil {
		log.Error("failed delete post", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "failed delete post"})
	}

	return c.NoContent(http.StatusNoContent)
}

// PublishPost godoc
// @Summary Опубликовать пост
// @Description Устанавливает статус поста "published"
// @Tags Посты
// @Param id path string true "UUID поста" format(uuid)
// @Success 200 {object} dto.BlogPostResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts/{id}/publish [patch]
func (r *Routers) PublishPost(c echo.Context) error {
	const op = "http.routers.PublishPost"

	log := r.log.With(
		slog.String("op", op),
	)

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error("invalid post ID format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid post ID format"})
	}

	post, err := r.BlogService.PublishPost(c.Request().Context(), postID)
	if err != nil {
		log.Error("failed publish post", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "failed publish post"})
	}

	return c.JSON(http.StatusOK, post)
}

// ArchivePost godoc
// @Summary Архивировать пост
// @Description Архивирует пост (soft delete)
// @Tags Посты
// @Param id path string true "UUID поста" format(uuid)
// @Success 200 {object} dto.BlogPostResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts/{id}/archive [patch]
func (r *Routers) ArchivePost(c echo.Context) error {
	const op = "http.routers.ArchivePost"

	log := r.log.With(
		slog.String("op", op),
	)

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error("invalid post ID format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid post ID format"})
	}

	post, err := r.BlogService.ArchivePost(c.Request().Context(), postID)
	if err != nil {
		log.Error("failed archive post", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "failed archive post"})
	}

	return c.JSON(http.StatusOK, post)
}

// ListPosts godoc
// @Summary Список постов
// @Description Возвращает список постов с пагинацией и фильтрацией по статусу. http://localhost:8080/api/v1/posts?status=archived&page=1&per_page=1
// @Tags Посты
// @Produce json
// @Param status query string false "Фильтр по статусу (draft, published, archived)"
// @Param page query int false "Номер страницы" default(1)
// @Param per_page query int false "Количество элементов на странице" default(10)
// @Success 200 {object} dto.BlogPostListResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts [get]
func (r *Routers) ListPosts(c echo.Context) error {
	const op = "http.routers.ListPosts"

	log := r.log.With(
		slog.String("op", op),
	)

	status := c.QueryParam("status")

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(c.QueryParam("per_page"))
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 10
	}

	posts, err := r.BlogService.ListPosts(c.Request().Context(), status, page, perPage)
	if err != nil {
		log.Error("failed list post", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "failed list post"})
	}

	return c.JSON(http.StatusOK, posts)
}

// AddMediaGroup godoc
// @Summary Добавить медиа-группу к посту
// @Description Привязывает медиа-группу к посту с указанием типа связи
// @Tags Посты
// @Accept json
// @Produce json
// @Param id path string true "UUID поста" format(uuid)
// @Param request body dto.AddMediaGroupRequest true "Данные медиа-группы"
// @Success 200 {object} dto.PostMediaGroupsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts/{id}/media-groups [post]
func (r *Routers) AddMediaGroup(c echo.Context) error {
	const op = "http.routers.AddMediaGroup"

	log := r.log.With(
		slog.String("op", op),
	)

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error("invalid post format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid post ID format"})
	}

	req := new(dto.AddMediaGroupRequest)
	if err := c.Bind(req); err != nil {
		log.Error("invalid request data", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request data"})
	}

	if err := c.Validate(req); err != nil {
		log.Error("invalid validate data", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
	}

	_, err = r.BlogService.AddMediaGroup(c.Request().Context(), postID, *req)
	if err != nil {
		log.Error("failed add media group", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request data"})
	}

	return c.JSON(http.StatusOK, response.Response{Data: "succesfull"})
}

// GetPostMediaGroups godoc
// @Summary Получить медиа-группы поста
// @Description Возвращает список медиа-групп, привязанных к посту
// @Tags Посты
// @Produce json
// @Param id path string true "UUID поста" format(uuid)
// @Param relation_type query string false "Тип связи (content, gallery, attachment)"
// @Success 200 {object} dto.PostMediaGroupsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/posts/{id}/media-groups [get]
func (r *Routers) GetPostMediaGroups(c echo.Context) error {
	const op = "http.routers.GetPostMediaGroups"

	log := r.log.With(
		slog.String("op", op),
	)

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Error("failed parse uuid", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid uuid format"})
	}

	relationType := c.QueryParam("relation_type")

	resp, err := r.BlogService.GetPostMediaGroups(c.Request().Context(), postID, relationType)
	if err != nil {
		log.Error("failed get post media group", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request data"})
	}

	return c.JSON(http.StatusOK, resp)
}
