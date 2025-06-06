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
	const op = "repository.media_repository.CreateMedia"

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
		return nil, fmt.Errorf("failed to build query:%s %w", op, err)
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

func (r *MediaRepo) AddMediaGroup(ctx context.Context, ownerID uuid.UUID, description string) error {
	const op = "repository.media_repository.AddMedia"

	query, args, err := r.sb.Insert("media_groups").
		Columns("owner_id", "description").
		Values(
			ownerID,
			description,
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
	query, args, err := r.sb.Update("media").
		Set("original_filename", media.OriginalFilename).
		Set("is_public", media.IsPublic).
		Set("metadata", media.Metadata).
		Where(sq.Eq{"id": media.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to update media: %w", err)
	}

	_, err = r.db.Exec(ctx, query, args...)
	return err
}

func (r *MediaRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	const op = "repository.media_repository.FindByID"

	query, args, err := r.sb.Select("*").
		From("media").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to find by id media: %s %w", op, err)
	}

	row := r.db.QueryRow(ctx, query, args...)

	var media models.Media
	err = row.Scan(
		&media.ID,
		&media.UploaderID,
		&media.CreatedAt,
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

func (r *MediaRepo) AddMediaGroupItems(ctx context.Context, groupID, mediaID uuid.UUID) error {
	const op = "repository.media_repository.AddMediaGroupItems"

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %s %w", op, err)
	}
	defer tx.Rollback(ctx)

	var groupExists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM media_groups WHERE id = $1)`,
		groupID).Scan(&groupExists)
	if err != nil {
		return fmt.Errorf("failed to check group existence: %s %w", op, err)
	}
	if !groupExists {
		return fmt.Errorf("%s media group %s does not exist", op, groupID)
	}

	var mediaExists bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM media WHERE id = $1)`,
		mediaID).Scan(&mediaExists)
	if err != nil {
		return fmt.Errorf("%s failed to check media existence: %w", op, err)
	}
	if !mediaExists {
		return fmt.Errorf("%s media file %s does not exist", op, mediaID)
	}

	query, args, err := r.sb.Insert("media_group_items").
		Columns("group_id", "media_id", "position", "created_at").
		Values(
			groupID,
			mediaID,
			sq.Expr("(SELECT COALESCE(MAX(position), 0) + 1 FROM media_group_items WHERE group_id = ?)", groupID),
			time.Now().UTC(),
		).
		Suffix("ON CONFLICT (group_id, media_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("%s failed to build query: %w", op, err)
	}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s failed to add media to group: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s failed to commit transaction: %w", op, err)
	}

	return nil
}

// SELECT m.*
// FROM media m
// JOIN media_group_items mgi ON m.id = mgi.media_id
// WHERE mgi.group_id = 'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22'
// ORDER BY mgi.position;
func (r *MediaRepo) GetMediaByGroupID(ctx context.Context, groupID uuid.UUID) ([]models.Media, error) {
	const op = "repository.media_repository.GetMediaByGroupID"

	query, args, err := r.sb.
		Select(
			"m.id",
			"m.uploader_id",
			"m.created_at",
			"m.media_type",
			"m.original_filename",
			"m.storage_path",
			"m.file_size",
			"m.mime_type",
			"m.width",
			"m.height",
			"m.duration",
			"m.is_public",
			"m.metadata",
		).
		From("media m").
		Join("media_group_items mgi ON m.id = mgi.media_id").
		Where(sq.Eq{"mgi.group_id": groupID}).
		OrderBy("mgi.position").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query:%s %w", op, err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var m models.Media
		err := rows.Scan(
			&m.ID,
			&m.UploaderID,
			&m.CreatedAt,
			&m.MediaType,
			&m.OriginalFilename,
			&m.StoragePath,
			&m.FileSize,
			&m.MimeType,
			&m.Width,
			&m.Height,
			&m.Duration,
			&m.IsPublic,
			&m.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("row scanning failed:%s %w", op, err)
		}

		mediaList = append(mediaList, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error:%s %w", op, err)
	}
	return mediaList, nil
}

// GetAllImages возвращает все загруженные картинки (медиа типа 'photo')
func (r *MediaRepo) GetAllImages(ctx context.Context) ([]models.Media, error) {
	const op = "repository.media_repository.GetAllImages"

	query, args, err := r.sb.
		Select(
			"id",
			"uploader_id",
			"created_at",
			"original_filename",
			"storage_path",
			"file_size",
			"width",
			"height",
			"is_public",
			"metadata",
		).
		From("media").
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}
	defer rows.Close()

	var images []models.Media

	for rows.Next() {
		var img models.Media
		err := rows.Scan(
			&img.ID,
			&img.UploaderID,
			&img.CreatedAt,
			&img.OriginalFilename,
			&img.StoragePath,
			&img.FileSize,
			&img.Width,
			&img.Height,
			&img.IsPublic,
			&img.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to scan row: %w", op, err)
		}
		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	return images, nil
}
