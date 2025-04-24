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
	"github.com/labstack/echo/v4"

	_ "premium_caste/docs"
)

type UserService interface {
	Login(ctx context.Context, email, password string) (string, error)
	RegisterNewUser(ctx context.Context, input dto.UserRegisterInput) (uuid.UUID, error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

type MediaService interface {
	UploadMedia(ctx context.Context, input dto.MediaUploadInput) (*models.Media, error)
	AttachMediaToGroup(ctx context.Context, groupID uuid.UUID, mediaID uuid.UUID) error
	AttachMedia(ctx context.Context, ownerID uuid.UUID, description string) error
	ListGroupMedia(ctx context.Context, groupID uuid.UUID) ([]models.Media, error)
}

type Routers struct {
	log          *slog.Logger
	UserService  UserService
	MediaService MediaService
}

func NewRouter(log *slog.Logger, userService UserService, mediaService MediaService) *Routers {
	return &Routers{
		log:          log,
		UserService:  userService,
		MediaService: mediaService,
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
	const op = "http.routers.auth.Login"

	log := r.log.With(
		slog.String("op", op),
	)

	var req request.LoginRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.ErrInvalidRequestFormat)
	}

	if err := c.Validate(req); err != nil {
		response.ErrInvalidRequestFormat.Details = err.Error()
		log.Warn("invalid format request", slog.String("email", req.Email))
		return c.JSON(http.StatusBadRequest, response.ErrInvalidRequestFormat)
	}

	token, err := r.UserService.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		response.ErrAuthenticationFailed.Details = err.Error()
		return c.JSON(http.StatusUnauthorized, response.ErrAuthenticationFailed)
	}

	return c.JSON(http.StatusOK, response.Response{
		Status: "success",
		Data:   map[string]string{"token": token},
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
	const op = "http.routers.auth.Register"

	log := r.log.With(
		slog.String("op", op),
	)

	var req dto.UserRegisterInput

	if err := c.Bind(&req); err != nil {
		log.Error("failed to bind request", sl.Err(err))
		return c.JSON(http.StatusBadRequest, response.ErrInvalidRegisterRequest)
	}

	if err := c.Validate(req); err != nil {
		response.ErrInvalidRegisterRequest.Details = err.Error()
		log.Error("validation failed", sl.Err(err))
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
	userIDStr := c.Param("user_id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user ID format",
		})
	}

	isAdmin, err := r.UserService.IsAdmin(c.Request().Context(), userID)
	if err != nil {
		r.log.Error("failed to check admin status",
			slog.String("error", err.Error()),
			slog.Any("user_id", userID),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to check admin status",
		})
	}

	return c.JSON(http.StatusOK, map[string]bool{
		"is_admin": isAdmin,
	})
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
	startTime := time.Now()
	defer func() {
		r.log.Info("Request completed",
			"duration", time.Since(startTime))
	}()

	r.log.Info("Start uploading media",
		"method", c.Request().Method,
		"path", c.Path(),
		"client_ip", c.RealIP())

	file, err := c.FormFile("file")
	if err != nil {
		r.log.Warn("Empty file in request",
			"error", err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "File is required"})
	}

	r.log.Debug("Got file for upload",
		"filename", file.Filename,
		"size", file.Size,
		"mime_type", file.Header.Get("Content-Type"))

	input, err := r.parseMediaUploadInput(c)
	if err != nil {
		r.log.Warn("Error parsing data",
			"error", err.Error(),
			"uploader_id", c.FormValue("uploader_id"))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	input.File = file

	r.log.Debug("Options for upload",
		"uploader_id", input.UploaderID,
		"media_type", input.MediaType,
		"is_public", input.IsPublic,
		"metadata", input.CustomMetadata)

	media, err := r.MediaService.UploadMedia(c.Request().Context(), *input)
	if err != nil {
		r.log.Error("Error upload media",
			"error", err.Error(),
			"uploader_id", input.UploaderID,
			"filename", file.Filename)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	r.log.Info("Upload successfull",
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
	req := new(dto.AttachMediaRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request data format",
		})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	groupID, _ := uuid.Parse(req.GroupID)
	mediaID, _ := uuid.Parse(req.MediaID)

	if err := r.MediaService.AttachMediaToGroup(
		c.Request().Context(),
		groupID,
		mediaID,
	); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
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
	req := new(dto.CreateMediaGroupRequest)

	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request data",
		})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	ownerID, _ := uuid.Parse(req.OwnerID)

	if err := r.MediaService.AttachMedia(
		c.Request().Context(),
		ownerID,
		req.Description,
	); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
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
	req := new(dto.ListGroupMediaRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid query parameters",
		})
	}

	if err := c.Validate(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	groupID, _ := uuid.Parse(req.GroupID)

	media, err := r.MediaService.ListGroupMedia(c.Request().Context(), groupID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
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

// func (h *AuthHandler) Refresh(c echo.Context) error {
//     var req struct {
//         RefreshToken string `json:"refresh_token"`
//     }

//     if err := c.Bind(&req); err != nil {
//         return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
//     }

//     newTokens, err := h.tokenService.RefreshTokens(req.RefreshToken)
//     if err != nil {
//         return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
//     }

//     return c.JSON(http.StatusOK, newTokens)
// }
