package services

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/storage"
	"premium_caste/internal/transport/http/dto"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) SaveUser(ctx context.Context, user models.User) (uuid.UUID, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockUserRepository) User(ctx context.Context, email string) (models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockUserRepository) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateTokens(user models.User) (*models.TokenPair, error) {
	args := m.Called(user)
	return args.Get(0).(*models.TokenPair), args.Error(1)
}

func (m *MockTokenService) RefreshTokens(refreshToken string) (*models.TokenPair, error) {
	args := m.Called(refreshToken)
	return args.Get(0).(*models.TokenPair), args.Error(1)
}

func createTestContext() echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestUserService_Login(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)
	log := slog.Default()

	c := createTestContext()

	service := NewUserService(log, mockRepo, mockToken)

	testEmail := "test@example.com"
	testPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	testUser := models.User{
		Email:    testEmail,
		Password: hashedPassword,
	}

	expectedTokens := &models.TokenPair{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
	}

	t.Run("successful login", func(t *testing.T) {
		mockRepo.On("User", ctx, testEmail).Return(testUser, nil).Once()
		mockToken.On("GenerateTokens", testUser).Return(expectedTokens, nil).Once()
		mockToken.On("RefreshTokens", testUser).Return(expectedTokens, nil).Once()

		token, err := service.Login(ctx, c, testEmail, testPassword)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Проверяем что токен валиден
		// _, err = jwt.Parse(myToken, func(token *jwt.Token) (interface{}, error) {
		// 	return []byte(myKey), nil
		// })

		// assert.NoError(t, err)
	})

	t.Run("invalid password", func(t *testing.T) {
		mockRepo.On("User", ctx, testEmail).Return(testUser, nil).Once()

		_, err := service.Login(ctx, c, testEmail, "wrong_password")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.On("User", ctx, "nonexistent@example.com").
			Return(models.User{}, storage.ErrUserNotFound).Once()

		_, err := service.Login(ctx, c, "nonexistent@example.com", testPassword)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("User", ctx, testEmail).
			Return(models.User{}, errors.New("db error")).Once()

		_, err := service.Login(ctx, c, testEmail, testPassword)
		assert.ErrorContains(t, err, "db error")
	})
}

func TestUserService_RegisterNewUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)
	log := slog.Default()
	service := NewUserService(log, mockRepo, mockToken)

	// Тестовые данные
	testInput := dto.UserRegisterInput{
		Name:     "Test User",
		Email:    "test@example.com",
		Phone:    "+1234567890",
		Password: "password123",
		IsAdmin:  false,
	}

	t.Run("successful registration", func(t *testing.T) {
		expectedID := uuid.New()
		mockRepo.On("SaveUser", ctx, mock.AnythingOfType("models.User")).
			Return(expectedID, nil).Once()

		id, err := service.RegisterNewUser(ctx, testInput)
		require.NoError(t, err)
		assert.Equal(t, expectedID, id)
	})

	t.Run("user already exists", func(t *testing.T) {
		mockRepo.On("SaveUser", ctx, mock.Anything).
			Return(uuid.Nil, storage.ErrUserExists).Once()

		_, err := service.RegisterNewUser(ctx, testInput)
		assert.ErrorIs(t, err, storage.ErrUserExists)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("SaveUser", ctx, mock.Anything).
			Return(uuid.Nil, errors.New("db error")).Once()

		_, err := service.RegisterNewUser(ctx, testInput)
		assert.ErrorContains(t, err, "db error")
	})

	t.Run("invalid password hash", func(t *testing.T) {
		// Тест на слишком длинный пароль (bcrypt имеет ограничение 72 байта)
		longPassInput := testInput
		longPassInput.Password = string(make([]byte, 100))

		_, err := service.RegisterNewUser(ctx, longPassInput)
		assert.Error(t, err)
	})
}

func TestUserService_IsAdmin(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockUserRepository)
	mockToken := new(MockTokenService)
	log := slog.Default()
	service := NewUserService(log, mockRepo, mockToken)

	testUserID := uuid.New()

	t.Run("user is admin", func(t *testing.T) {
		mockRepo.On("IsAdmin", ctx, testUserID).Return(true, nil).Once()

		isAdmin, err := service.IsAdmin(ctx, testUserID)
		require.NoError(t, err)
		assert.True(t, isAdmin)
	})

	t.Run("user is not admin", func(t *testing.T) {
		mockRepo.On("IsAdmin", ctx, testUserID).Return(false, nil).Once()

		isAdmin, err := service.IsAdmin(ctx, testUserID)
		require.NoError(t, err)
		assert.False(t, isAdmin)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.On("IsAdmin", ctx, testUserID).
			Return(false, errors.New("db error")).Once()

		_, err := service.IsAdmin(ctx, testUserID)
		assert.ErrorContains(t, err, "db error")
	})
}
