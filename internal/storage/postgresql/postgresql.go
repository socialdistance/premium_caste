package postgresql

import (
	"context"
	"fmt"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/storage"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

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
func (s *Storage) SaveUser(ctx context.Context, name, email, phone string, password []byte, permissionId int, basketId uuid.UUID) (int64, error) {
	const op = "storage.postgresql.SaveUser"

	builder := sq.Insert(userTabe).Columns(
		"name",
		"email",
		"phone",
		"password",
		"permission_id",
		"basket_id",
		"registration_date",
		"last_login",
	)

	builder = builder.Values(name, email, phone, password, permissionId, basketId, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))

	query, args, err := builder.Suffix("RETURNING id").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return 0, fmt.Errorf("%s: can't build sql:%w", op, err)
	}

	rows, err := s.db.Query(s.ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("%s: row err: %w", op, err)
	}
	defer rows.Close()

	var ID int64
	for rows.Next() {
		scanErr := rows.Scan(&ID)
		if scanErr != nil {
			return 0, fmt.Errorf("%s can't scan id: %s", op, scanErr.Error())
		}
	}

	return ID, nil
}

// User returns user by email
// Rewrite to sq
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgresql.User"

	sql := `SELECT id, name, email, password, permission_id, basket_id FROM users WHERE email = $1`

	rows, err := s.db.Query(ctx, sql, email)
	if err != nil {
		return models.User{}, fmt.Errorf("%s, %w", op, err)
	}

	defer rows.Close()

	var user models.User

	for rows.Next() {
		err = rows.Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.Permission_id, &user.Basket_id)
		if err != nil {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
	}

	return user, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	panic("implement me")
}

// Basket returns table basket by user id

// func AddMediaToGroup(db *sql.DB, groupID, mediaID uuid.UUID) error {
//     tx, err := db.Begin()
//     if err != nil {
//         return fmt.Errorf("failed to begin transaction: %w", err)
//     }
//     defer tx.Rollback() // Откатываем в случае ошибки

//     // Проверяем существование группы
//     var groupExists bool
//     err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM media_groups WHERE id = $1)`, groupID).Scan(&groupExists)
//     if err != nil {
//         return fmt.Errorf("failed to check group existence: %w", err)
//     }
//     if !groupExists {
//         return fmt.Errorf("media group %s does not exist", groupID)
//     }

//     // Проверяем существование медиа
//     var mediaExists bool
//     err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM media WHERE id = $1)`, mediaID).Scan(&mediaExists)
//     if err != nil {
//         return fmt.Errorf("failed to check media existence: %w", err)
//     }
//     if !mediaExists {
//         return fmt.Errorf("media file %s does not exist", mediaID)
//     }

//     // Добавляем связь
//     _, err = tx.Exec(`
//         INSERT INTO media_group_items (group_id, media_id)
//         VALUES ($1, $2)
//         ON CONFLICT (group_id, media_id) DO NOTHING`,
//         groupID, mediaID)
//     if err != nil {
//         return fmt.Errorf("failed to add media to group: %w", err)
//     }

//     return tx.Commit()
// }
