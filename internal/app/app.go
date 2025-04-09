package app

import (
	"context"
	"premium_caste/internal/services/auth"
	"premium_caste/internal/storage/postgresql"
	"time"

	"log/slog"
	httpapp "premium_caste/internal/app/http"
)

type App struct {
	HTTPServer httpapp.Server
}

func New(log *slog.Logger, storagePath string, httpHost, httpPort string, tokenTTL time.Duration) *App {
	storage, err := postgresql.New(context.Background(), storagePath)
	if err != nil {
		panic(err)
	}

	_ = auth.New(log, storage, storage, tokenTTL)

	return nil
}
