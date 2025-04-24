package app

import (
	"context"
	"log/slog"
	"time"

	httpapp "premium_caste/internal/app/http"
	"premium_caste/internal/repository"
	media "premium_caste/internal/services/media_service"
	user "premium_caste/internal/services/user_service"
	storage "premium_caste/internal/storage/filestorage"
	redisapp "premium_caste/internal/storage/redis"

	httprouters "premium_caste/internal/transport/http"
)

type App struct {
	HTTPServer httpapp.Server
	Repo       repository.Repository
}

func New(log *slog.Logger, redisClient *redisapp.Client, storagePath string, httpHost, httpPort string, tokenTTL time.Duration, baseDir, baseURL string) *App {
	ctx := context.Background()
	token := "test"

	repo, err := repository.NewRepository(ctx, storagePath, redisClient)
	if err != nil {
		panic("not init repo")
	}

	fileStorage, err := storage.NewLocalFileStorage(baseDir, baseURL)
	if err != nil {
		panic("not init file storage")
	}

	userSerivce := user.NewUserService(log, repo.User, tokenTTL)
	mediaService := media.NewMediaService(log, repo.Media, fileStorage)
	// authService := auth.NewTokenService(repo.Token)

	httpRouters := httprouters.NewRouter(log, userSerivce, mediaService)
	httpApp := httpapp.New(log, token, httpHost, httpPort, httpRouters)

	return &App{
		HTTPServer: *httpApp,
		Repo:       *repo,
	}
}
