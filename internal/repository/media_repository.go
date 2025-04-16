package repository

import (
	"context"
	"fmt"
	"time"

	"premium_caste/internal/domain/models"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type MediaRepo struct {
	db *pgxpool.Pool
	sb sq.StatementBuilderType
}

func NewMediaRepository(db *pgxpool.Pool) *MediaRepo {
	return &MediaRepo{
		db: db,
		sb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *MediaRepo) CreateMedia(ctx context.Context, media *models.Media) (*models.Media, error) {
	const op = "storage.postgresql.CreateMedia"

	query, args, err := r.sb.Insert("media").
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

	row := r.db.QueryRow(ctx, query, args...)

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
func (r *MediaRepo) AddMedia(ctx context.Context, groupID, mediaID uuid.UUID) error {
	const op = "storage.postgresql.AddMediaToGroup"

	query, args, err := r.sb.Insert("media_group_items").
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

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to add media to group:%s %w", op, err)
	}

	return nil
}

func (r *MediaRepo) UpdateMedia(ctx context.Context, media *models.Media) error {
	const op = "storage.postgresql.UpdateMedia"

	query, args, err := r.sb.Update("media").
		Set("original_filename", media.OriginalFilename).
		Set("is_public", media.IsPublic).
		Set("metadata", media.Metadata).
		Where(sq.Eq{"id": media.ID}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, query, args...)
	return err
}

func (r *MediaRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	const op = "storage.postgresql.FindByID"

	query, args, err := r.sb.Select("*").
		From("media").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRow(ctx, query, args...)

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
