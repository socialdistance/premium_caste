package app

import (
	httpapp "premium_caste/internal/app/http"

	"log/slog"
	"time"
)

type App struct {
	HTTPServer httpapp.Server
}

func New(log *slog.Logger, grpcPort int, storagePath string, httpHost, httpPort string, token string, tokenTTL time.Duration) *App {

	return nil
}
