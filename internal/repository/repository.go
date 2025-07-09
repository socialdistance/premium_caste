package repository

import (
	"context"
	"fmt"
	redisapp "premium_caste/internal/storage/redis"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	User    UserRepository
	Media   MediaRepository
	Token   TokenRepository
	Blog    BlogRepository
	Gallery GalleryRepository
}

func NewRepository(ctx context.Context, dsn string, redis *redisapp.Client) (*Repository, error) {
	db, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Repository{
		User:    NewUserRepository(db),
		Media:   NewMediaRepository(db),
		Token:   NewRedisTokenRepo(redis),
		Blog:    NewBlogRepository(db),
		Gallery: NewGalleryRepo(db),
	}, nil
}

func (r *Repository) Close() error {
	if r != nil && r.db != nil {
		r.db.Close()
	}

	return nil
}
