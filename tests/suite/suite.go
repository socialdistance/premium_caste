package suite

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"premium_caste/internal/config"
	"premium_caste/internal/services/auth"
	"premium_caste/internal/services/auth/mocks"

	"github.com/stretchr/testify/mock"
)

type Suite struct {
	*testing.T
	Cfg         *config.Config
	AuthService auth.Auth
}

func New(t *testing.T) (context.Context, *Suite) {
	t.Helper()
	t.Parallel()

	cfg := config.MustLoadPath(configPath())

	log := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Duration(time.Hour))

	usrSaver := mocks.NewUserSaver(t)
	usrProvider := mocks.NewUserProvider(t)

	var monkeyID int64 = 15

	usrSaver.On("SaveUser", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int"), mock.Anything).Return(monkeyID, nil)

	// usrProvider.On("User", mock.Anything, mock.AnythingOfType("string")).Return(, errors.New("Error"))

	authService := auth.New(log, usrSaver, usrProvider, time.Duration(time.Hour))

	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	return ctx, &Suite{
		T:           t,
		Cfg:         cfg,
		AuthService: *authService,
	}
}

func configPath() string {
	const key = "CONFIG_PATH"

	if v := os.Getenv(key); v != "" {
		return v
	}

	return "../config/config.yaml"
}
