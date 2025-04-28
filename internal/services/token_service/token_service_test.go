package services

import (
	"context"
	"errors"
	"premium_caste/internal/domain/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) SaveRefreshToken(ctx context.Context, userID, token string, exp time.Duration) error {
	args := m.Called(ctx, userID, token, exp)
	return args.Error(0)
}

func (m *MockTokenRepository) GetRefreshToken(ctx context.Context, userID, token string) (bool, error) {
	args := m.Called(ctx, userID, token)
	return args.Bool(0), args.Error(1)
}

func (m *MockTokenRepository) DeleteRefreshToken(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}

func (m *MockTokenRepository) DeleteAllUserTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

var (
	testUser = models.User{
		ID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Email: "test@example.com",
	}
	testCtx = context.Background()
)

func TestGenerateTokens_Success(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	repo.On("SaveRefreshToken", testCtx, testUser.ID.String(), mock.Anything, mock.Anything).
		Return(nil)

	tokens, err := service.GenerateTokens(testUser)

	assert.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	repo.AssertExpectations(t)
}

func TestGenerateTokens_RepoError(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	expectedErr := errors.New("storage error")
	repo.On("SaveRefreshToken", testCtx, testUser.ID.String(), mock.Anything, mock.Anything).
		Return(expectedErr)

	tokens, err := service.GenerateTokens(testUser)

	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, tokens)
	repo.AssertExpectations(t)
}

func TestRefreshTokens_Success(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	// Генерируем валидный refresh token
	refreshToken, _ := NewToken(testUser, time.Hour)

	// Настраиваем ожидания мока
	repo.On("GetRefreshToken", testCtx, testUser.ID.String(), refreshToken).
		Return(true, nil)
	repo.On("DeleteRefreshToken", testCtx, testUser.ID.String(), refreshToken).
		Return(nil)
	repo.On("SaveRefreshToken", testCtx, testUser.ID.String(), mock.Anything, mock.Anything).
		Return(nil)

	newTokens, err := service.RefreshTokens(refreshToken)

	assert.NoError(t, err)
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	repo.AssertExpectations(t)
}

func TestRefreshTokens_InvalidToken(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	_, err := service.RefreshTokens("invalid.token.string")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestRefreshTokens_TokenNotInStorage(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	refreshToken, _ := NewToken(testUser, time.Hour)

	repo.On("GetRefreshToken", testCtx, testUser.ID.String(), refreshToken).
		Return(false, nil)

	_, err := service.RefreshTokens(refreshToken)

	assert.ErrorIs(t, err, ErrInvalidToken)
	repo.AssertExpectations(t)
}

func TestRefreshTokens_StorageError(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	refreshToken, _ := NewToken(testUser, time.Hour)
	expectedErr := errors.New("storage error")

	repo.On("GetRefreshToken", testCtx, testUser.ID.String(), refreshToken).
		Return(false, expectedErr)

	_, err := service.RefreshTokens(refreshToken)

	assert.ErrorContains(t, err, "invalid token")
	repo.AssertExpectations(t)
}

func TestRefreshTokens_ExpiredToken(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	expiredToken, _ := NewToken(testUser, -time.Hour)

	repo.On("GetRefreshToken", mock.Anything, mock.Anything, mock.Anything).
		Return(false, nil).Maybe()

	_, err := service.RefreshTokens(expiredToken)

	assert.ErrorContains(t, err, "invalid token")
	repo.AssertExpectations(t)
}

func TestRefreshTokens_DeleteTokenError(t *testing.T) {
	repo := new(MockTokenRepository)
	service := NewTokenService(repo)

	refreshToken, _ := NewToken(testUser, time.Hour)
	expectedErr := errors.New("delete error")

	repo.On("GetRefreshToken", testCtx, testUser.ID.String(), refreshToken).
		Return(true, nil)
	repo.On("DeleteRefreshToken", testCtx, testUser.ID.String(), refreshToken).
		Return(expectedErr)

	_, err := service.RefreshTokens(refreshToken)

	assert.ErrorIs(t, err, expectedErr)
	repo.AssertExpectations(t)
}
