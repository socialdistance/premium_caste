package app

import (
	"context"
	httpapp "premium_caste/internal/app/http"
	"premium_caste/internal/services/auth"
	"premium_caste/internal/storage/postgresql"

	"log/slog"
)

type App struct {
	HTTPServer httpapp.Server
}

func New(log *slog.Logger, storagePath string, httpHost, httpPort string) *App {
	storage, err := postgresql.New(context.Background(), storagePath)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage)

	return nil
}
