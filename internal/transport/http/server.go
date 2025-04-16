package http

import "log/slog"

type Storage interface {
}

type Routers struct {
	log     *slog.Logger
	storage Storage
}

func NewRouter(log *slog.Logger, storage Storage) *Routers {
	return &Routers{
		log:     log,
		storage: storage,
	}
}
