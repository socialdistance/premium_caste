package postgresql

import (
	"context"
	"fmt"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/storage"

	sq "github.com/Masterminds/squirrel"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Storage struct {
	ctx context.Context
	db  *pgxpool.Pool
}

const (
	// tables
	userTabe        = "users"
	permissionTable = "permissions"
)

func New(ctx context.Context, storagePath string) (*Storage, error) {
	const op = "storage.postgresql.New"

	db, err := pgxpool.Connect(ctx, storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		db:  db,
		ctx: ctx,
	}, nil
}

func (s *Storage) Stop() {
	s.db.Close()
}

// SaveUser saves user to db
func (s *Storage) SaveUser(ctx context.Context) (int64, error) {
	const op = "storage.postgresql.SaveUser"

	builder := sq.Insert(userTabe).Columns()
}

// User returns user by email
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgresql.User"

	sql := `SELECT id, name, email, password, permission_id, basket_id FROM users WHERE email = $1`

	rows, err := s.db.Query(ctx, sql, email)
	if err != nil {
		return models.User{}, fmt.Errorf("%s, %w", op, err)
	}

	defer rows.Close()

	var user models.User

	err = rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Permission_id, &user.Basket_id)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
	}

	return user, nil
}
