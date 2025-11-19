package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Login(ctx context.Context, email, password string) (*models.TokenPair, error)
	RegisterNewUser(ctx context.Context, input dto.UserRegisterInput) (uuid.UUID, error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	GetUserById(ctx context.Context, userID uuid.UUID) (models.User, error)
}

type MediaService interface {
	UploadMedia(ctx context.Context, input dto.MediaUploadInput) (*models.Media, error)
	UploadMultipleMedia(ctx context.Context, inputs []dto.MediaUploadInput) ([]*models.Media, error)
	AttachMediaToGroup(ctx context.Context, groupID uuid.UUID, mediaIDs []uuid.UUID) error
	AttachMedia(ctx context.Context, ownerID uuid.UUID, description string) (uuid.UUID, error)
	ListGroupMedia(ctx context.Context, groupID uuid.UUID) ([]models.Media, error)
	GetAllImages(ctx context.Context, limitInt int) ([]models.Media, error)
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

type GalleryService interface {
	CreateGallery(ctx context.Context, req dto.CreateGalleryRequest) (uuid.UUID, error)
	UpdateGallery(ctx context.Context, req dto.UpdateGalleryRequest) error
	UpdateGalleryStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteGallery(ctx context.Context, id uuid.UUID) error
	GetGalleryByID(ctx context.Context, id uuid.UUID) (*dto.GalleryResponse, error)
	GetGalleries(ctx context.Context, statusFilter string, page int, perPage int) ([]dto.GalleryResponse, int, error)
	GetGalleriesByTags(ctx context.Context, tags []string, matchAll bool) ([]dto.GalleryResponse, error)
	AddTags(ctx context.Context, galleryID string, tags []string) error
	RemoveTags(ctx context.Context, galleryID string, tagsToRemove []string) error
	ReplaceTags(ctx context.Context, galleryID string, newTags []string) error
	GetTags(ctx context.Context, galleryID string) ([]string, error)
	HasTags(ctx context.Context, galleryID string, tags []string) (bool, error)
}

type Routers struct {
	log            *slog.Logger
	UserService    UserService
	MediaService   MediaService
	AuthService    AuthService
	BlogService    BlogService
	GalleryService GalleryService
}

func NewRouter(log *slog.Logger, userService UserService, mediaService MediaService, authService AuthService, blogService BlogService, galleryService GalleryService) *Routers {
	return &Routers{
		log:            log,
		UserService:    userService,
		MediaService:   mediaService,
		AuthService:    authService,
		BlogService:    blogService,
		GalleryService: galleryService,
	}
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExist          = errors.New("user already exist")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidUUID        = errors.New("not valid UUID")
	ErrGalleryNotFound    = errors.New("gallery not founc")
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

	token, err := r.UserService.Login(c.Request().Context(), req.Identifier, req.Password)
	if err != nil {
		response.ErrAuthenticationFailed.Details = err.Error()
		return c.JSON(http.StatusUnauthorized, response.ErrAuthenticationFailed)
	}

	sess, _ := session.Get("session", c)
	sess.Values["user_id"] = token.UserID
	sess.Save(c.Request(), c.Response())

	http.SetCookie(c.Response().Writer, &http.Cookie{
		Name:     "access_token",
		Value:    token.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	http.SetCookie(c.Response().Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    token.RefreshToken,
		Path:     "/api/v1/refresh",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	return c.JSON(http.StatusOK, response.Response{
		Status: "success",
		Data: map[string]interface{}{
			"user_id":       token.UserID.String(),
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"session": map[string]interface{}{
				"expires_in": 86400 * 7,
				"expires_at": time.Now().Add(86400 * 7 * time.Second).Format(time.RFC3339),
			},
		},
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

	http.SetCookie(c.Response().Writer, &http.Cookie{
		Name:     "access_token",
		Value:    newTokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	http.SetCookie(c.Response().Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    newTokens.RefreshToken,
		Path:     "/api/v1/refresh",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

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
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.JSON(http.StatusInternalServerError, "failed to save session")
	}

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
// @Router /api/v1/users/users_id [post]
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

// UploadMultipleMedia обрабатывает запрос на загрузку множества медиафайлов
// UploadMultipleMedia godoc
// @Summary Загрузка множества медиафайлов
// @Description Загружает несколько медиафайлов на сервер. Поддерживает передачу файлов и общих параметров загрузки.
// @Tags Медиа
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Файлы для загрузки (поддерживается множественная загрузка)"
// @Param uploader_id formData string true "ID пользователя, загружающего файлы"
// @Param media_type formData string true "Тип медиа (например, image, video)"
// @Param is_public formData boolean true "Флаг публичности файла"
// @Param metadata formData string false "Дополнительные метаданные (опционально)"
// @Success 201 {array} models.Media "Успешная загрузка, возвращает массив созданных медиа-объектов"
// @Failure 400 {object} map[string]string "Ошибка валидации входных данных"
// @Failure 500 {object} map[string]string "Ошибка сервера при загрузке файлов"
// @Router /api/v1/media/multiple [post]
func (r *Routers) UploadMultipleMedia(c echo.Context) error {
	const op = "http.routers.UploadMultipleMedia"

	log := r.log.With(
		slog.String("op", op),
	)

	startTime := time.Now()
	defer func() {
		log.Info("Request completed",
			"duration", time.Since(startTime))
	}()

	log.Info("Start uploading multiple media",
		"method", c.Request().Method,
		"path", c.Path(),
		"client_ip", c.RealIP())

	// Получаем файлы из запроса
	form, err := c.MultipartForm()
	if err != nil {
		log.Warn("Failed to parse multipart form",
			"error", err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form data"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		log.Warn("No files in request")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "At least one file is required"})
	}

	log.Debug("Got files for upload",
		"count", len(files))

	// Парсим общие параметры загрузки
	baseInput, err := r.parseMediaUploadInput(c)
	if err != nil {
		log.Warn("Error parsing base data",
			"error", err.Error(),
			"uploader_id", c.FormValue("uploader_id"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Подготавливаем входные данные для каждого файла
	inputs := make([]dto.MediaUploadInput, 0, len(files))
	for _, file := range files {
		input := *baseInput
		input.File = file
		inputs = append(inputs, input)
	}

	// Выполняем загрузку через сервисный слой
	medias, err := r.MediaService.UploadMultipleMedia(c.Request().Context(), inputs)
	if err != nil {
		log.Error("Error uploading multiple media",
			"error", err.Error(),
			"uploader_id", baseInput.UploaderID,
			"file_count", len(files))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Info("Upload successful",
		"media_count", len(medias),
		"uploader_id", baseInput.UploaderID,
		"duration", time.Since(startTime))

	return c.JSON(http.StatusCreated, medias)
}

// AttachMediaToGroup godoc
// @Summary Прикрепить медиа к группе
// @Description Связывает один или несколько медиафайлов с существующей медиагруппой
// @Tags Медиа-группы
// @Accept json
// @Produce json
// @Param group_id path string true "UUID группы" format(uuid)
// @Param request body dto.AttachMediaRequest true "Данные для прикрепления"
//
//	{
//	  "mediaIDs": [
//	    "3fa85f64-5717-4562-b3fc-2c963f66afa6",
//	    "4fb85f64-5717-4562-b3fc-2c963f66afa7"
//	  ]
//	}
//
// @Success 200 "Успешное прикрепление (no content)"
// @Failure 400 {object} response.ErrorResponse "Невалидные данные: пустой массив, неверный формат UUID"
// @Failure 404 {object} response.ErrorResponse "Группа не найдена"
// @Failure 413 {object} response.ErrorResponse "Превышен лимит количества медиафайлов"
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
		log.Error("invalid request data format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: "Invalid request data format",
		})
	}

	// 2. Валидация запроса
	if err := c.Validate(req); err != nil {
		log.Error("validation failed", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	// 3. Парсинг groupID
	groupID, err := uuid.Parse(req.GroupID)
	if err != nil {
		log.Error("invalid groupID format", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: "Invalid groupID format",
		})
	}

	// 4. Парсинг массива mediaIDs
	mediaIDs := make([]uuid.UUID, 0, len(req.MediaIDs))
	for _, idStr := range req.MediaIDs {
		mediaID, err := uuid.Parse(idStr)
		if err != nil {
			log.Error("invalid mediaID format", slog.String("mediaID", idStr), sl.Err(err))
			return c.JSON(http.StatusBadRequest, response.ErrorResponse{
				Error: fmt.Sprintf("Invalid mediaID format: %s", idStr),
			})
		}
		mediaIDs = append(mediaIDs, mediaID)
	}

	// 5. Вызов сервиса с массивом mediaIDs
	if err := r.MediaService.AttachMediaToGroup(
		c.Request().Context(),
		groupID,
		mediaIDs,
	); err != nil {
		log.Error("failed to attach media", sl.Err(err))
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, response.Response{
		Status: "success",
		Data: map[string]interface{}{
			"attachedCount": len(mediaIDs),
			"groupID":       groupID.String(),
		},
	})
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

	groupID, err := r.MediaService.AttachMedia(c.Request().Context(), ownerID, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	response := map[string]interface{}{
		"data": groupID,
	}

	return c.JSON(http.StatusOK, response)
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

// GetAllImages godoc
// @Summary Получить все изображения
// @Description Возвращает список всех загруженных изображений с метаданными
// @Tags Медиа
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{} "Успешный ответ"
// @SuccessExample {json} Успешный ответ:
//
//	{
//	    "data": [
//	        {
//	            "id": "550e8400-e29b-41d4-a716-446655440000",
//	            "uploader_id": "550e8400-e29b-41d4-a716-446655440000",
//	            "created_at": "2025-06-06T11:08:12Z",
//	            "original_filename": "nature.jpg",
//	            "storage_path": "images/2025/06/550e8400-e29b-41d4-a716-446655440000.jpg",
//	            "file_size": 102400,
//	            "mime_type": "image/jpeg",
//	            "width": 1920,
//	            "height": 1080,
//	            "is_public": true,
//	            "metadata": {}
//	        }
//	    ],
//	    "meta": {
//	        "count": 1
//	    }
//	}
//
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/media/images [get]
func (r *Routers) GetAllImages(c echo.Context) error {
	const op = "http.routers.GetAllImages"

	log := r.log.With(
		slog.String("op", op),
	)

	// Получаем параметр limit из query (по умолчанию 0 — без лимита)
	limit := c.QueryParam("limit")
	var limitInt int
	if limit != "" {
		var err error
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			log.Error("invalid limit parameter", sl.Err(err))
			return c.JSON(http.StatusBadRequest, response.ErrorResponse{
				Error: "limit must be a valid integer",
			})
		}
	}

	media, err := r.MediaService.GetAllImages(c.Request().Context(), limitInt)
	if err != nil {
		log.Error("failed get images", sl.Err(err))
		return c.JSON(http.StatusInternalServerError, response.ErrorResponse{
			Error: err.Error(),
		})
	}

	response := map[string]interface{}{
		"data": media,
		"meta": map[string]interface{}{
			"count": len(media),
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

// CreateGalleryHandler создает новую галерею.
// @Summary Создание новой галереи
// @Description Создает новую галерею на основе переданных данных.
// @Tags Галереи
// @Accept json
// @Produce json
// @Param request body dto.CreateGalleryRequest true "Данные для создания галереи"
//
//	{
//	    "title": "test title gallery",
//	    "slug": "test slug gallery",
//	    "description": "test Description gallery",
//	    "images": [
//	        "uploads/1221067c-cc35-4dae-b5f5-feee4bbb3e22/test.png",
//	        "uploads/1221067c-cc35-4dae-b5f5-feee4bbb3e22/test3.png"
//	    ],
//	    "cover_image_index": 1,
//	    "author_id": "1221067c-cc35-4dae-b5f5-feee4bbb3e22",
//	    "status": "draft",
//	    "tags": ["test tags", "test tags"],
//	    "metadata": {}
//	}
//
// @Success 201 {object} map[string]string "Галерея успешно создана, возвращает ID созданной галереи"
// @Failure 400 {object} map[string]string "Некорректные данные запроса"
// @Failure 500 {object} map[string]string "Ошибка при создании галереи"
// @Router /galleries [post]
func (r *Routers) CreateGalleryHandler(c echo.Context) error {
	const op = "http.routers.CreateGalleryHandler"

	log := r.log.With(
		slog.String("op", op),
	)

	var req dto.CreateGalleryRequest
	if err := c.Bind(&req); err != nil {
		log.Error("invalid request data", sl.Err(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Вызываем сервис
	id, err := r.GalleryService.CreateGallery(c.Request().Context(), req)
	if err != nil {
		log.Error("error create gallery", sl.Err(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"id": id.String()})
}

// UpdateGalleryHandler обрабатывает запрос на обновление галереи
// UpdateGalleryHandler обновляет существующую галерею.
// @Summary Обновление галереи
// @Description Обновляет данные существующей галереи на основе переданных данных.
// @Tags Галереи
// @Accept json
// @Produce json
// @Param request body dto.UpdateGalleryRequest true "Данные для обновления галереи"
//
//	{
//	    "id": "1221067c-cc35-4dae-b5f5-feee4bbb3e22",
//	    "title": "updated test title gallery",
//	    "slug": "updated-test-slug-gallery",
//	    "description": "updated test Description gallery",
//	    "images": [
//	        "uploads/1221067c-cc35-4dae-b5f5-feee4bbb3e22/updated-test.png",
//	        "uploads/1221067c-cc35-4dae-b5f5-feee4bbb3e22/updated-test3.png"
//	    ],
//	    "cover_image_index": 1,
//	    "author_id": "1221067c-cc35-4dae-b5f5-feee4bbb3e22",
//	    "status": "published",
//	    "tags": ["updated test tags", "updated test tags"],
//	    "metadata": {}
//	}
//
// @Success 200 {object} map[string]string "Галерея успешно обновлена"
// @Failure 400 {object} map[string]string "Некорректные данные запроса"
// @Failure 500 {object} map[string]string "Ошибка при обновлении галереи"
// @Router /galleries/{id} [put]
func (r *Routers) UpdateGalleryHandler(c echo.Context) error {
	var req dto.UpdateGalleryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Вызываем сервис
	err := r.GalleryService.UpdateGallery(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "gallery updated successfully"})
}

// UpdateGalleryStatusHandler обновляет статус галереи.
// @Summary Обновление статуса галереи
// @Description Обновляет статус существующей галереи на основе переданных данных.
// @Tags Галереи
// @Accept json
// @Produce json
// @Param id path string true "ID галереи"
// @Param request body dto.UpdateGalleryStatusRequest true "Данные для обновления статуса галереи"
//
//	{
//	    "status": "published"
//	}
//
// @Success 200 {object} map[string]string "Статус галереи успешно обновлен"
// @Failure 400 {object} map[string]string "Некорректные данные запроса"
// @Failure 500 {object} map[string]string "Ошибка при обновлении статуса галереи"
// @Router /galleries/{id}/status [put]
// UpdateGalleryStatusHandler обрабатывает запрос на обновление статуса галереи
func (r *Routers) UpdateGalleryStatusHandler(c echo.Context) error {
	galleryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid gallery ID"})
	}

	var req dto.UpdateGalleryStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Вызываем сервис
	err = r.GalleryService.UpdateGalleryStatus(c.Request().Context(), galleryID, req.Status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "gallery status updated successfully"})
}

// DeleteGalleryHandler удаляет галерею.
// @Summary Удаление галереи
// @Description Удаляет существующую галерею по её ID.
// @Tags Галереи
// @Accept json
// @Produce json
// @Param id path string true "ID галереи"
// @Success 200 {object} map[string]string "Галерея успешно удалена"
// @Failure 400 {object} map[string]string "Некорректный ID галереи"
// @Failure 500 {object} map[string]string "Ошибка при удалении галереи"
// @Router /galleries/{id} [delete]
// DeleteGalleryHandler обрабатывает запрос на удаление галереи
func (r *Routers) DeleteGalleryHandler(c echo.Context) error {
	galleryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid gallery ID"})
	}

	// Вызываем сервис
	err = r.GalleryService.DeleteGallery(c.Request().Context(), galleryID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "gallery deleted successfully"})
}

// GetGalleryByIDHandler обрабатывает запрос на получение галереи по ID
// GetGalleryByIDHandler возвращает галерею по её ID.
// @Summary Получение галереи по ID
// @Description Возвращает полную информацию о галерее по её уникальному идентификатору.
// @Tags Галереи
// @Accept json
// @Produce json
// @Param id path string true "UUID галереи" example("1221067c-cc35-4dae-b5f5-feee4bbb3e22")
// @Success 200 {object} map[string]string "Успешный ответ с данными галереи"
// @Failure 400 {object} map[string]string "Некорректный формат UUID"
// @Failure 404 {object} map[string]string "Галерея не найдена"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries/{id} [get]
func (r *Routers) GetGalleryByIDHandler(c echo.Context) error {
	galleryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid gallery ID"})
	}

	// Вызываем сервис
	gallery, err := r.GalleryService.GetGalleryByID(c.Request().Context(), galleryID)
	if err != nil {
		if errors.Is(err, ErrGalleryNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "gallery not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, gallery)
}

// GetGalleriesHandler обрабатывает запрос на получение списка галерей
// GetGalleriesHandler возвращает список галерей с фильтрацией и пагинацией.
// @Summary Получение списка галерей
// @Description Возвращает список галерей с возможностью фильтрации по статусу и пагинацией.
// @Tags Галереи
// @Accept json
// @Produce json
// @Param status query string false "Фильтр по статусу галереи" example("published")
// @Param page query int false "Номер страницы (по умолчанию: 1)" example(1)
// @Param per_page query int false "Количество элементов на странице (по умолчанию: 10, максимум: 100)" example(10)
// @Success 200 {object} map[string]interface{} "Успешный ответ с данными галерей и общим количеством"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries [get]
func (r *Routers) GetGalleriesHandler(c echo.Context) error {
	statusFilter := c.QueryParam("status")

	// Устанавливаем значения по умолчанию
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(c.QueryParam("per_page"))
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// Вызываем сервис
	galleries, total, err := r.GalleryService.GetGalleries(c.Request().Context(), statusFilter, page, perPage)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"galleries": galleries,
		"total":     total,
	})
}

// GetGalleriesByTagsHandler обрабатывает запрос на получение списка галерей по тегам
// GetGalleriesByTagsHandler возвращает список галерей, отфильтрованных по тегам.
// @Summary Получение списка галерей по тегам
// @Description Возвращает список галерей, отфильтрованных по указанным тегам, с возможностью выбора логики фильтрации (AND/OR).
// @Tags Галереи
// @Accept json
// @Produce json
// @Param tags query []string true "Список тегов для фильтрации" example(["nature", "art"])
//
//	GET /galleries/by-tags?tags=nature&tags=art&match_all=true
//
// @Param match_all query bool false "Режим фильтрации: true — AND, false — OR (по умолчанию: false)" example(false)
// @Success 200 {object} map[string]interface{} "Успешный ответ с данными галерей"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries/by-tags [get]
func (r *Routers) GetGalleriesByTagsHandler(c echo.Context) error {
	// Получаем список тегов из запроса
	tags := c.QueryParams()["tags"]
	if len(tags) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tags parameter is required"})
	}

	// Получаем режим фильтрации (AND/OR)
	matchAll, err := strconv.ParseBool(c.QueryParam("match_all"))
	if err != nil {
		matchAll = false
	}

	// Вызываем сервис
	galleries, err := r.GalleryService.GetGalleriesByTags(c.Request().Context(), tags, matchAll)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"galleries": galleries,
	})
}

// AddTagsHandler обрабатывает запрос на добавление тегов к галерее
// @Summary Добавление тегов к галерее
// @Description Добавляет указанные теги к галерее
// @Tags Галереи
// @Accept json
// @Produce json
// @Param gallery_id path string true "Идентификатор галереи" example("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
// @Param tags body []string true "Список тегов для добавления" example(["art", "design"])
// @Success 204 "Теги успешно добавлены"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries/{gallery_id}/tags [post]
func (r *Routers) AddTagsHandler(c echo.Context) error {
	galleryID := c.Param("gallery_id")

	var tags dto.GalleryTagsRequest
	if err := c.Bind(&tags); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := r.GalleryService.AddTags(c.Request().Context(), galleryID, tags.Tags); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveTagsHandler обрабатывает запрос на удаление тегов из галереи
// @Summary Удаление тегов из галереи
// @Description Удаляет указанные теги из галереи
// @Tags Галереи
// @Accept json
// @Produce json
// @Param gallery_id path string true "Идентификатор галереи" example("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
// @Param tags body []string true "Список тегов для удаления" example(["old", "tag"])
// @Success 204 "Теги успешно удалены"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries/{gallery_id}/tags [delete]
func (r *Routers) RemoveTagsHandler(c echo.Context) error {
	galleryID := c.Param("gallery_id")

	var tags dto.GalleryTagsRequest
	if err := c.Bind(&tags); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := r.GalleryService.RemoveTags(c.Request().Context(), galleryID, tags.Tags); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// ReplaceTagsHandler обрабатывает запрос на замену тегов галереи
// @Summary Замена тегов галереи
// @Description Полностью заменяет теги галереи на указанные
// @Tags Галереи
// @Accept json
// @Produce json
// @Param gallery_id path string true "Идентификатор галереи" example("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
// @Param tags body []string true "Новый список тегов" example(["new", "tags"])
// @Success 204 "Теги успешно заменены"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries/{gallery_id}/tags [put]
func (r *Routers) ReplaceTagsHandler(c echo.Context) error {
	galleryID := c.Param("gallery_id")

	var tags dto.GalleryTagsRequest
	if err := c.Bind(&tags); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := r.GalleryService.ReplaceTags(c.Request().Context(), galleryID, tags.Tags); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// GetTagsHandler обрабатывает запрос на получение тегов галереи
// @Summary Получение тегов галереи
// @Description Возвращает список тегов для указанной галереи
// @Tags Галереи
// @Accept json
// @Produce json
// @Param gallery_id path string true "Идентификатор галереи" example("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
// @Success 200 {object} []string "Список тегов галереи"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries/{gallery_id}/tags [get]
func (r *Routers) GetTagsHandler(c echo.Context) error {
	galleryID := c.Param("gallery_id")

	tags, err := r.GalleryService.GetTags(c.Request().Context(), galleryID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, tags)
}

// HasTagsHandler обрабатывает запрос на проверку наличия тегов у галереи
// @Summary Проверка наличия тегов у галереи
// @Description Проверяет, содержит ли галерея все указанные теги
// @Tags Галереи
// @Accept json
// @Produce json
// @Param gallery_id path string true "Идентификатор галереи" example("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
// @Param tags query []string true "Список тегов для проверки" example(["art", "design"])
// @Success 200 {object} bool "Результат проверки"
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /galleries/{gallery_id}/has-tags [get]
func (r *Routers) HasTagsHandler(c echo.Context) error {
	galleryID := c.Param("gallery_id")
	tags := c.QueryParams()["tags"]

	if len(tags) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tags parameter is required"})
	}

	hasTags, err := r.GalleryService.HasTags(c.Request().Context(), galleryID, tags)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, hasTags)
}
