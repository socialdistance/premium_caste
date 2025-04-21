package http

import (
	"context"
	"log/slog"
	"net/http"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/transport/http/dto"
	"premium_caste/internal/transport/http/dto/request"
	"premium_caste/internal/transport/http/dto/response"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
	ErrInvalidRequestFormat = response.ErrorResponse{
		Status:  "error",
		Error:   "invalid_request",
		Details: "Invalid request format",
	}

	ErrAuthenticationFailed = response.ErrorResponse{
		Status: "error",
		Error:  "authentication_failed",
	}
)

func (r *Routers) Login(c echo.Context) error {
	var req request.LoginRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrInvalidRequestFormat)
	}

	if err := c.Validate(req); err != nil {
		ErrInvalidRequestFormat.Details = err.Error()
		return c.JSON(http.StatusBadRequest, ErrInvalidRequestFormat)
	}

	token, err := r.UserService.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		ErrAuthenticationFailed.Details = err.Error()
		return c.JSON(http.StatusUnauthorized, ErrAuthenticationFailed)
	}

	return c.JSON(http.StatusOK, response.Response{
		Status: "success",
		Data:   map[string]string{"token": token},
	})
}
