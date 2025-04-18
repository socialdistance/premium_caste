package repository

import (
	"context"
	"fmt"
	"time"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/storage"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
	sb sq.StatementBuilderType
}

func NewUserRepository(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		db: db,
		sb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *UserRepo) SaveUser(ctx context.Context, user models.User) (uuid.UUID, error) {
	query, args, err := r.sb.Insert("users").
		Columns(
			"name",
			"email",
			"phone",
			"password",
			"permission_id",
			"basket_id",
			"last_login",
		).
		Values(
			user.Name,
			user.Email,
			user.Phone,
			user.Password,
			user.PermissionID,
			user.BasketID,
			time.Now().UTC(),
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w", err)
	}

	var id uuid.UUID
	err = r.db.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w", err)
	}

	return id, nil
}

func (r *UserRepo) User(ctx context.Context, email string) (models.User, error) {
	sql, args, err := r.sb.Select("id", "name", "email", "password", "permission_id", "basket_id").From("users").Where(sq.Eq{"email": email}).ToSql()
	if err != nil {
		return models.User{}, fmt.Errorf("can't build sql:%w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return models.User{}, fmt.Errorf("%w", err)
	}

	defer rows.Close()

	var user models.User

	for rows.Next() {
		err = rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.PermissionID, &user.BasketID)
		if err != nil {
			return models.User{}, fmt.Errorf("%w", storage.ErrUserNotFound)
		}
	}

	return user, nil
}

func (r *UserRepo) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	panic("implement me")
}
