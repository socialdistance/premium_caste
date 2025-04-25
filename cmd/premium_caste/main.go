package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"premium_caste/internal/app"
	"premium_caste/internal/config"
	redisapp "premium_caste/internal/storage/redis"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	redisClient := redisapp.NewClient(cfg.Redis.RedisAddr, cfg.Redis.RedisPassword, cfg.Redis.RedisDB)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		fmt.Println(err)
		panic("Failed to connect to Redis:")
	}

	application := app.New(log, redisClient, cfg.DSN, cfg.HTTP.Host, cfg.HTTP.Port, cfg.TokenTTL, cfg.FileStorage.BaseDir, cfg.FileStorage.BaseURL)

	go func() {
		application.HTTPServer.BuildRouters()
		application.HTTPServer.MustRun()
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	application.HTTPServer.Stop()
	application.Repo.Close()
	redisClient.Close()

	log.Info("Gracefully stopped")
	log.Info("application stop")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		)
	}

	return log
}
