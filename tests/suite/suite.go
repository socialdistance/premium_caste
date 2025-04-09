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

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Duration(time.Hour))

	authService := auth.New(&slog.Logger{}, &mocks.UserSaver{}, &mocks.UserProvider{}, time.Duration(time.Hour))

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
