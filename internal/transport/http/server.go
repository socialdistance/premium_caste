package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/lib/logger/sl"
	"premium_caste/internal/transport/http/dto"
	"premium_caste/internal/transport/http/dto/request"
	"premium_caste/internal/transport/http/dto/response"

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
	mediaService MediaService
}

func NewRouter(log *slog.Logger, userService UserService, mediaService MediaService) *Routers {
	return &Routers{
		log:          log,
		UserService:  userService,
		mediaService: mediaService,
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
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "Данные для входа"
// @Success 200 {object} response.Response{data=map[string]string} "Успешный вход (токен)"
// @Failure 400 {object} response.ErrorResponse "Неверный формат запроса"
// @Failure 401 {object} response.ErrorResponse "Ошибка аутентификации"
// @Router /auth/login [post]
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
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.UserRegisterInput true "Данные для регистрации"
// @Success 201 {object} response.Response{data=object{user_id=string}} "Успешная регистрация"
// @Failure 400 {object} response.ErrorResponse "Неверный формат запроса"
// @Failure 409 {object} response.ErrorResponse "Пользователь уже существует"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /auth/register [post]
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

	// Вызываем сервисный метод
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
