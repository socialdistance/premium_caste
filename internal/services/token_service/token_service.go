package services

import (
	"context"
	"errors"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/repository"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidTokenClaims = errors.New("invalid token claims")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenNotInStorage  = errors.New("token not found in storage")
)

const (
	AccessTokenExpire  = 15 * time.Minute
	RefreshTokenExpire = 7 * 24 * time.Hour
	SecretKey          = "test"
)

type TokenService struct {
	repo repository.TokenRepository
}

func NewTokenService(repo repository.TokenRepository) *TokenService {
	return &TokenService{repo: repo}
}

func (s *TokenService) GenerateTokens(user models.User) (*models.TokenPair, error) {
	accessToken, err := NewToken(user, AccessTokenExpire)
	if err != nil {
		return nil, err
	}

	refreshToken, err := NewToken(user, RefreshTokenExpire)
	if err != nil {
		return nil, err
	}

	err = s.repo.SaveRefreshToken(context.Background(), user.ID.String(), refreshToken, RefreshTokenExpire)
	if err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *TokenService) RefreshTokens(refreshToken string) (*models.TokenPair, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(refreshToken, jwt.MapClaims{})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, ok := claims["uid"].(string)
	if !ok {
		return nil, ErrInvalidTokenClaims
	}

	exists, err := s.repo.GetRefreshToken(context.Background(), userID, refreshToken)
	if err != nil || !exists {
		return nil, ErrInvalidToken
	}

	if err := s.repo.DeleteRefreshToken(context.Background(), userID, refreshToken); err != nil {
		return nil, err
	}

	user := models.User{
		ID:    uuid.MustParse(userID),
		Email: claims["email"].(string),
	}

	return s.GenerateTokens(user)
}

func NewToken(user models.User, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["email"] = user.Email
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(duration).Unix()

	return token.SignedString([]byte(SecretKey))
}
