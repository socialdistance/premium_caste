package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"premium_caste/internal/app"
	"premium_caste/internal/config"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	_ = app.New(log, cfg.DSN, cfg.HTTP.Host, cfg.HTTP.Port, cfg.TokenTTL)

	// go func() {
	// 	application.GRPCServer.MustRun()
	// }()

	// go func() {
	// 	application.HTTPServer.BuildRouters()
	// 	application.HTTPServer.MustRun()
	// }()

	// go func() {
	// 	application.FileService.FileRun()
	// }()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	// application.GRPCServer.Stop()
	// application.HTTPServer.Stop()
	// application.FileService.Stop()
	// application.Watcher.Close()

	// log.Info("Gracefully stopped")
	// log.Info("application stop")
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
