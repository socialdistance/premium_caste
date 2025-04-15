package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"premium_caste/internal/lib/jwt"
	"premium_caste/internal/lib/logger/sl"
	"premium_caste/internal/repository"
	"premium_caste/internal/storage"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExist          = errors.New("user already exist")
	ErrUserNotFound       = errors.New("user not found")
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// func (s *UserService) RegisterUser(ctx context.Context, input UserRegisterInput) (uuid.UUID, error) {
// 	user := models.User{
// 		Name:         input.Name,
// 		Email:        input.Email,
// 		Phone:        input.Phone,
// 		Password:     input.Password,
// 		PermissionID: input.PermissionID,
// 		BasketID:     uuid.New(),
// 	}

// 	id, err := s.repo.SaveUser(ctx, user)
// 	if err != nil {
// 		return uuid.Nil, fmt.Errorf("failed to register user: %w", err)
// 	}

// 	return id, nil
// }

func (a *UserService) Login(ctx context.Context, email, password string) (string, error) {
	const op = "auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	log.Info("attempting to login user")

	user, err := a.usrProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", sl.Err(err))

			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		a.log.Error("failed to get user", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(password)); err != nil {
		a.log.Info("invalid credentials", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	log.Info("user logged in successfully")

	token, err := jwt.NewToken(user, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate token", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

func (a *Auth) RegisterNewUser(ctx context.Context, name, email, phone, pass string, permission_id int) (uuid.UUID, error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("register user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	basket_id := uuid.New()

	id, err := a.usrSaver.SaveUser(ctx, name, email, phone, passHash, permission_id, basket_id)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exist", slog.Any("error", err.Error()))

			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrUserExist)
		}

		log.Error("failed to save user", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user register")

	return id, nil
}

func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "Auth.IsAdmin"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)

	log.Info("checking if user is admin")

	isAdmin, err := a.usrProvider.IsAdmin(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked if user is admin", slog.Bool("is_admin", isAdmin))

	return isAdmin, nil
}
