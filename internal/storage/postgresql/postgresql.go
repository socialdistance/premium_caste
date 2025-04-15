package postgresql

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/storage"
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
func (s *Storage) SaveUser(ctx context.Context, name, email, phone string, password []byte, permissionId int, basketId uuid.UUID) (uuid.UUID, error) {
	const op = "storage.postgresql.SaveUser"

	builder := sq.Insert(userTabe).Columns(
		"name",
		"email",
		"phone",
		"password",
		"permission_id",
		"basket_id",
		"last_login",
	)

	builder = builder.Values(name, email, phone, password, permissionId, basketId, time.Now().Format(time.RFC3339))

	query, args, err := builder.Suffix("RETURNING id").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: can't build sql:%w", op, err)
	}

	rows, err := s.db.Query(s.ctx, query, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: row err: %w", op, err)
	}
	defer rows.Close()

	var ID uuid.UUID
	for rows.Next() {
		scanErr := rows.Scan(&ID)
		if scanErr != nil {
			return uuid.Nil, fmt.Errorf("%s can't scan id: %s", op, scanErr.Error())
		}
	}

	return ID, nil
}

// User returns user by email
// Rewrite to sq
func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgresql.User"

	// sql1 := `SELECT id, name, email, password, permission_id, basket_id FROM users WHERE email = $1`

	sql, args, err := sq.Select("id", "name", "email", "password", "permission_id", "basket_id").From("users").Where(sq.Eq{"email": email}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return models.User{}, fmt.Errorf("%s: can't build sql:%w", op, err)
	}

	rows, err := s.db.Query(ctx, sql, args...)
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

func (s *Storage) CreateMedia(ctx context.Context, media *models.Media) (*models.Media, error) {
	const op = "storage.postgresql.CreateMedia"

	query, args, err := sq.Insert("media").
		Columns(
			"id",
			"uploader_id",
			"created_at",
			"media_type",
			"original_filename",
			"storage_path",
			"file_size",
			"mime_type",
			"width",
			"height",
			"duration",
			"is_public",
			"metadata",
		).
		Values(
			media.ID,
			media.UploaderID,
			media.CreatedAt,
			media.MediaType,
			media.OriginalFilename,
			media.StoragePath,
			media.FileSize,
			media.MimeType,
			media.Width,
			media.Height,
			media.Duration,
			media.IsPublic,
			media.Metadata,
		).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %s %w", op, err)
	}

	row := s.db.QueryRow(ctx, query, args...)

	var createdMedia models.Media
	err = row.Scan(
		&createdMedia.ID,
		&createdMedia.UploaderID,
		&createdMedia.CreatedAt,
		&createdMedia.MediaType,
		&createdMedia.OriginalFilename,
		&createdMedia.StoragePath,
		&createdMedia.FileSize,
		&createdMedia.MimeType,
		&createdMedia.Width,
		&createdMedia.Height,
		&createdMedia.Duration,
		&createdMedia.IsPublic,
		&createdMedia.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create media: %s %w", op, err)
	}

	return &createdMedia, nil
}

// AddMediaToGroup добавляет медиа в группу (связь many-to-many)
func (s *Storage) AddMediaToGroup(ctx context.Context, groupID, mediaID uuid.UUID) error {
	const op = "storage.postgresql.AddMediaToGroup"

	query, args, err := sq.Insert("media_group_items").
		Columns("group_id", "media_id", "position", "created_at").
		Values(
			groupID,
			mediaID,
			0, // Позиция по умолчанию
			time.Now().UTC(),
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %s %w", op, err)
	}

	_, err = s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to add media to group:%s %w", op, err)
	}

	return nil
}

func (s *Storage) UpdateMedia(ctx context.Context, media *models.Media) error {
	const op = "storage.postgresql.UpdateMedia"

	query, args, err := sq.Update("media").
		Set("original_filename", media.OriginalFilename).
		Set("is_public", media.IsPublic).
		Set("metadata", media.Metadata).
		Where(sq.Eq{"id": media.ID}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, query, args...)
	return err
}

func (s *Storage) FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	const op = "storage.postgresql.FindByID"

	query, args, err := sq.Select("*").
		From("media").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRow(ctx, query, args...)

	var media models.Media
	err = row.Scan(
		&media.ID,
		&media.UploaderID,
		&media.MediaType,
		&media.OriginalFilename,
		&media.StoragePath,
		&media.FileSize,
		&media.MimeType,
		&media.Width,
		&media.Height,
		&media.Duration,
		&media.IsPublic,
		&media.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %s %w", op, err)
	}

	return &media, nil
}

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
