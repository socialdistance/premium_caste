package app

import (
	"context"
	"log/slog"
	"time"

	httpapp "premium_caste/internal/app/http"
	"premium_caste/internal/repository"
	blog "premium_caste/internal/services/blog_service"
	gallery "premium_caste/internal/services/gallery_service"
	media "premium_caste/internal/services/media_service"
	tokenapp "premium_caste/internal/services/token_service"
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

	tokenService := tokenapp.NewTokenService(repo.Token)
	blogService := blog.NewBlogService(log, repo.Blog)
	userSerivce := user.NewUserService(log, repo.User, tokenService)
	mediaService := media.NewMediaService(log, repo.Media, fileStorage)
	galleryService := gallery.NewGalleryService(log, repo.Gallery)

	httpRouters := httprouters.NewRouter(log, userSerivce, mediaService, tokenService, blogService, galleryService)
	httpApp := httpapp.New(log, token, httpHost, httpPort, httpRouters)

	return &App{
		HTTPServer: *httpApp,
		Repo:       *repo,
	}
}
