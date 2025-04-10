package auth

import (
	"log/slog"
	"os"
	"premium_caste/internal/services/auth/mocks"
	"testing"
	"time"
)

// https://outcomeschool.com/blog/test-with-testify-and-mockery-in-go
func TestRegisterLogin_Login_HappyPath(t *testing.T) {
	log := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

	mockUserSaver := new(mocks.UserSaver)
	mockUserProvider := new(mocks.UserProvider)

	// mockUser := models.User{

	// }

	mockAuthService := New(log, mockUserSaver, mockUserProvider, time.Duration(time.Hour))

	t.Run("success", func(t *testing.T) {
		// mockUserSaver.On("SaveUser", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int"), mock.Anything).Return(test, nil)
	})

	t.Run("error", func(t *testing.T) {

	})

}
