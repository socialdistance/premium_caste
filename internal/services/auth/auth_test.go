package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/services/auth/mocks"
	"premium_caste/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type testUser struct {
	email    string
	password string
	hash     []byte
}

func createTestUser(t *testing.T) testUser {
	password := "securepassword123"
	return testUser{
		email:    "test@example.com",
		password: password,
		hash:     generateTestHash(t, password),
	}
}

func setupAuthService(t *testing.T) (*Auth, *mocks.UserProvider, *mocks.UserSaver) {
	mockUserProvider := new(mocks.UserProvider)
	mockUserSaver := new(mocks.UserSaver)

	return New(
		slog.Default(),
		mockUserSaver,
		mockUserProvider,
		time.Hour*24,
	), mockUserProvider, mockUserSaver
}

func generateTestHash(t *testing.T, password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate hash: %v", err)
	}
	return hash
}

func TestAuth_Login_Success(t *testing.T) {
	authService, mockUserProvider, _ := setupAuthService(t)
	testUser := createTestUser(t)

	mockUserProvider.On("User", mock.Anything, testUser.email).
		Return(models.User{
			Email:    testUser.email,
			Password: testUser.hash,
		}, nil)

	token, err := authService.Login(context.Background(), testUser.email, testUser.password)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	mockUserProvider.AssertExpectations(t)
}

func TestAuth_Login_UserNotFound(t *testing.T) {
	authService, mockUserProvider, _ := setupAuthService(t)

	mockUserProvider.On("User", mock.Anything, "notfound@example.com").
		Return(models.User{}, storage.ErrUserNotFound)

	_, err := authService.Login(context.Background(), "notfound@example.com", "password")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
	mockUserProvider.AssertExpectations(t)
}

func TestAuth_Login_InvalidPassword(t *testing.T) {
	authService, mockUserProvider, _ := setupAuthService(t)

	// Генерируем хеш для "valid_password"
	validHash := generateTestHash(t, "valid_password")

	expectedUser := models.User{
		Email:    "user@example.com",
		Password: validHash, // Хеш от "valid_password"
	}

	mockUserProvider.On("User", mock.Anything, "user@example.com").
		Return(expectedUser, nil)

	// Пытаемся войти с НЕправильным паролем
	_, err := authService.Login(context.Background(), "user@example.com", "wrong_password")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
	mockUserProvider.AssertExpectations(t)
}

func TestAuth_RegisterNewUser_Success(t *testing.T) {
	authService, _, mockUserSaver := setupAuthService(t)

	// Настраиваем ожидания
	mockUserSaver.On("SaveUser",
		mock.Anything,                    // ctx
		"Alice",                          // name
		"alice@example.com",              // email
		"+123456789",                     // phone
		mock.AnythingOfType("[]uint8"),   // passHash
		1,                                // permission_id
		mock.AnythingOfType("uuid.UUID"), // basket_id
	).Return(int64(1), nil)

	userID, err := authService.RegisterNewUser(
		context.Background(),
		"Alice",
		"alice@example.com",
		"+123456789",
		"password123",
		1,
	)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), userID)
	mockUserSaver.AssertExpectations(t)
}

func TestAuth_RegisterNewUser_AlreadyExists(t *testing.T) {
	authService, _, mockUserSaver := setupAuthService(t)

	mockUserSaver.On("SaveUser",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(int64(0), storage.ErrUserExists)

	_, err := authService.RegisterNewUser(
		context.Background(),
		"Bob",
		"bob@example.com",
		"+987654321",
		"password123",
		2,
	)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUserExist))
	mockUserSaver.AssertExpectations(t)
}

func TestAuth_IsAdmin_Success(t *testing.T) {
	authService, mockUserProvider, _ := setupAuthService(t)

	// Настраиваем ожидания
	mockUserProvider.On("IsAdmin", mock.Anything, int64(1)).
		Return(true, nil)

	isAdmin, err := authService.IsAdmin(context.Background(), 1)

	assert.NoError(t, err)
	assert.True(t, isAdmin)
	mockUserProvider.AssertExpectations(t)
}

func TestAuth_IsAdmin_NotAdmin(t *testing.T) {
	authService, mockUserProvider, _ := setupAuthService(t)

	mockUserProvider.On("IsAdmin", mock.Anything, int64(2)).
		Return(false, nil)

	isAdmin, err := authService.IsAdmin(context.Background(), 2)

	assert.NoError(t, err)
	assert.False(t, isAdmin)
	mockUserProvider.AssertExpectations(t)
}

func TestAuth_IsAdmin_Error(t *testing.T) {
	authService, mockUserProvider, _ := setupAuthService(t)

	expectedErr := errors.New("database error")
	mockUserProvider.On("IsAdmin", mock.Anything, int64(3)).
		Return(false, expectedErr)

	_, err := authService.IsAdmin(context.Background(), 3)

	assert.Error(t, err)
	assert.EqualError(t, err, "Auth.IsAdmin: database error")
	mockUserProvider.AssertExpectations(t)
}
