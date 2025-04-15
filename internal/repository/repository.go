package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Repository struct {
	db    *pgxpool.Pool
	User  UserRepository
	Media MediaRepository
}

func NewRepository(ctx context.Context, dsn string) (*Repository, error) {
	db, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Repository{
		User:  NewUserRepository(db),
		Media: NewMediaRepository(db),
	}, nil
}

func (r *Repository) Close() {
	r.db.Close()
}
