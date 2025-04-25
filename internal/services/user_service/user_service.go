package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/lib/logger/sl"
	"premium_caste/internal/repository"
	services "premium_caste/internal/services/auth_service"
	"premium_caste/internal/storage"
	"premium_caste/internal/transport/http/dto"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExist          = errors.New("user already exist")
	ErrUserNotFound       = errors.New("user not found")
)

type UserService struct {
	log         *slog.Logger
	repo        repository.UserRepository
	authService *services.TokenService
}

func NewUserService(log *slog.Logger, repo repository.UserRepository, authService *services.TokenService) *UserService {
	return &UserService{
		log:         log,
		repo:        repo,
		authService: authService,
	}
}

func (u *UserService) Login(ctx context.Context, email, password string) (*models.TokenPair, error) {
	const op = "user_service.Login"

	log := u.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	log.Info("attempting to login user")

	user, err := u.repo.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))

			return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		u.log.Error("failed to get user", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(password)); err != nil {
		u.log.Info("invalid credentials", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	log.Info("user logged in successfully")

	token, err := u.authService.GenerateTokens(user)
	if err != nil {
		u.log.Error("failed to generate token", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (u *UserService) RegisterNewUser(ctx context.Context, input dto.UserRegisterInput) (uuid.UUID, error) {
	const op = "user_service.RegisterNewUser"

	log := u.log.With(
		slog.String("op", op),
		slog.String("email", input.Email),
	)

	log.Info("register user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	user := models.User{
		Name:     input.Name,
		Email:    input.Email,
		Phone:    input.Phone,
		Password: passHash,
		IsAdmin:  input.IsAdmin,
		BasketID: uuid.New(),
	}

	id, err := u.repo.SaveUser(ctx, user)
	if err != nil {
		if errors.Is(err, ErrUserExist) {
			log.Warn("user already exist", slog.Any("error", err.Error()))

			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrUserExist)
		}

		log.Error("failed to save user", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user register")

	return id, nil
}

func (u *UserService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	const op = "user_service.IsAdmin"

	log := u.log.With(
		slog.String("op", op),
		slog.Any("user_id", userID),
	)

	log.Info("checking if user is admin")

	isAdmin, err := u.repo.IsAdmin(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked if user is admin", slog.Bool("is_admin", isAdmin))

	return isAdmin, nil
}
