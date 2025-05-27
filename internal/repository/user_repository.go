package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/storage"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx"
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
	const op = "repository.user_repository.SaveUser"

	query, args, err := r.sb.Insert("users").
		Columns(
			"name",
			"email",
			"phone",
			"password",
			"is_admin",
			"basket_id",
			"last_login",
		).
		Values(
			user.Name,
			user.Email,
			user.Phone,
			user.Password,
			user.IsAdmin,
			user.BasketID,
			time.Now().UTC(),
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	var id uuid.UUID
	err = r.db.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// func (r *UserRepo) User(ctx context.Context, email string) (models.User, error) {
// 	const op = "repository.user_repository.User"

// 	sql, args, err := r.sb.Select("id", "name", "email", "password", "is_admin", "basket_id").From("users").Where(sq.Eq{"email": email}).ToSql()
// 	if err != nil {
// 		return models.User{}, fmt.Errorf("%s: can't build sql:%w", op, err)
// 	}

// 	rows, err := r.db.Query(ctx, sql, args...)
// 	if err != nil {
// 		return models.User{}, fmt.Errorf("%s: %w", op, err)
// 	}
// 	defer rows.Close()

// 	var user models.User

// 	for rows.Next() {
// 		err = rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.IsAdmin, &user.BasketID)
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
// 		}

// 		// return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
// 	}

// 	return user, nil
// }

func (r *UserRepo) UserByIdentifier(ctx context.Context, identifier string) (models.User, error) {
	const op = "repository.user_repository.UserByIdentifier"

	// Определяем, является ли identifier email'ом или телефоном
	isEmail := strings.Contains(identifier, "@")
	condition := sq.Eq{"email": identifier}
	if !isEmail {
		condition = sq.Eq{"phone": identifier}
	}

	sql, args, err := r.sb.Select("id", "name", "email", "phone", "password", "is_admin", "basket_id").
		From("users").
		Where(condition).
		ToSql()
	if err != nil {
		return models.User{}, fmt.Errorf("%s: can't build sql:%w", op, err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var user models.User
	if rows.Next() {
		err = rows.Scan(&user.ID, &user.Name, &user.Email, &user.Phone, &user.Password, &user.IsAdmin, &user.BasketID)
		if err != nil {
			return models.User{}, fmt.Errorf("%s: %w", op, err)
		}

		return user, nil
	}

	return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
}

func (r *UserRepo) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	const op = "repository.user_repository.IsAdmin"

	sql, args, err := r.sb.Select("is_admin").From("users").Where(sq.Eq{"id": userID}).ToSql()
	if err != nil {
		return false, fmt.Errorf("%s: can't build sql: %w", op, err)
	}

	var isAdmin bool
	err = r.db.QueryRow(ctx, sql, args...).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

func (r *UserRepo) GetUserById(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	const op = "repository.user_repository.GetUserById"

	sql, args, err := r.sb.
		Select(
			"name",
			"email",
			"phone",
			"is_admin",
			"basket_id",
			"registration_date",
			"last_login",
		).
		From("users").
		Where(sq.Eq{"id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: can't build sql: %w", op, err)
	}

	var user models.User

	// Выполняем запрос и сканируем результат
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.IsAdmin,
		&user.BasketID,
		&user.RegistrationDate,
		&user.LastLogin,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}
