package repository

import (
	"context"
	"errors"
	"fmt"
	"premium_caste/internal/domain/models"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
)

type GalleryRepo struct {
	db *pgxpool.Pool
	sb squirrel.StatementBuilderType
}

func NewGalleryRepo(db *pgxpool.Pool) *GalleryRepo {
	return &GalleryRepo{
		db: db,
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateGallery создает новую галерею и возвращает её ID
func (r *GalleryRepo) CreateGallery(ctx context.Context, gallery models.Gallery) (uuid.UUID, error) {
	const op = "repository.GalleryRepo.CreateGallery"

	query, args, err := r.sb.Insert("galleries").
		Columns(
			"title",
			"slug",
			"description",
			"images",
			"cover_image_index",
			"author_id",
			"status",
			"metadata",
			"tags",
		).
		Values(
			gallery.Title,
			gallery.Slug,
			gallery.Description,
			gallery.Images,
			gallery.CoverImageIndex,
			gallery.AuthorID,
			gallery.Status,
			gallery.Metadata,
			gallery.Tags,
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

// UpdateGallery обновляет данные галереи
func (r *GalleryRepo) UpdateGallery(ctx context.Context, gallery models.Gallery) error {
	const op = "repository.GalleryRepo.UpdateGallery"

	query, args, err := r.sb.Update("galleries").
		Set("title", gallery.Title).
		Set("slug", gallery.Slug).
		Set("description", gallery.Description).
		Set("images", gallery.Images).
		Set("cover_image_index", gallery.CoverImageIndex).
		Set("status", gallery.Status).
		Set("metadata", gallery.Metadata).
		Set("tags", gallery.Tags).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": gallery.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UpdateGalleryStatus обновляет только статус галереи
func (r *GalleryRepo) UpdateGalleryStatus(ctx context.Context, id uuid.UUID, status string) error {
	const op = "repository.GalleryRepo.UpdateGalleryStatus"

	query, args, err := r.sb.Update("galleries").
		Set("status", status).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DeleteGallery удаляет галерею по ID
func (r *GalleryRepo) DeleteGallery(ctx context.Context, id uuid.UUID) error {
	const op = "repository.GalleryRepo.DeleteGallery"

	query, args, err := r.sb.Delete("galleries").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetGalleryByID возвращает галерею по ID
func (r *GalleryRepo) GetGalleryByID(ctx context.Context, id uuid.UUID) (models.Gallery, error) {
	const op = "repository.GalleryRepo.GetGalleryByID"

	query, args, err := r.sb.Select(
		"id",
		"title",
		"slug",
		"description",
		"images",
		"cover_image_index",
		"author_id",
		"status",
		"published_at",
		"created_at",
		"updated_at",
		"metadata",
		"tags",
	).
		From("galleries").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return models.Gallery{}, fmt.Errorf("%s: %w", op, err)
	}

	var gallery models.Gallery
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&gallery.ID,
		&gallery.Title,
		&gallery.Slug,
		&gallery.Description,
		&gallery.Images,
		&gallery.CoverImageIndex,
		&gallery.AuthorID,
		&gallery.Status,
		&gallery.PublishedAt,
		&gallery.CreatedAt,
		&gallery.UpdatedAt,
		&gallery.Metadata,
		&gallery.Tags,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.Gallery{}, fmt.Errorf("%s: %w", op, err)
		}
		return models.Gallery{}, fmt.Errorf("%s: %w", op, err)
	}

	return gallery, nil
}

func (r *GalleryRepo) GetGalleries(
	ctx context.Context,
	statusFilter string, // "all", "draft", "published", "archived"
	page int,
	perPage int,
) ([]models.Gallery, int, error) {
	const op = "repository.GalleryRepo.GetGalleries"

	// Проверка и корректировка параметров пагинации
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// Строим базовый запрос
	queryBuilder := r.sb.Select(
		"id", "title", "slug", "description", "images",
		"cover_image_index", "author_id", "status",
		"metadata", "tags", "created_at", "updated_at",
	).From("galleries")

	// Применяем фильтр по статусу
	switch statusFilter {
	case "draft", "published", "archived":
		queryBuilder = queryBuilder.Where(squirrel.Eq{"status": statusFilter})
	case "all":
		// Без фильтрации
	default:
		return nil, 0, fmt.Errorf("%s: invalid status filter '%s'", op, statusFilter)
	}

	// Получаем общее количество галерей (для пагинации)
	totalCount, err := r.getTotalCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	// Применяем пагинацию
	queryBuilder = queryBuilder.
		OrderBy("created_at DESC").
		Limit(uint64(perPage)).
		Offset(uint64((page - 1) * perPage))

	// Формируем SQL-запрос
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	// Выполняем запрос
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	// Сканируем результаты
	var galleries []models.Gallery
	for rows.Next() {
		var gallery models.Gallery
		err := rows.Scan(
			&gallery.ID,
			&gallery.Title,
			&gallery.Slug,
			&gallery.Description,
			&gallery.Images,
			&gallery.CoverImageIndex,
			&gallery.AuthorID,
			&gallery.Status,
			&gallery.Metadata,
			&gallery.Tags,
			&gallery.CreatedAt,
			&gallery.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", op, err)
		}
		galleries = append(galleries, gallery)
	}

	return galleries, totalCount, nil
}

// Вспомогательная функция для получения общего количества записей
func (b *GalleryRepo) getTotalCount(ctx context.Context) (int, error) {
	queryBuilder := squirrel.Select("COUNT(*)").
		From("galleries")

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("error build query: %w", err)
	}

	var count int
	err = b.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("error execute query: %w (SQL: %s)", err, query)
	}

	return count, nil
}

// GetGalleriesByTags возвращает галереи, отфильтрованные по тегам
func (b *GalleryRepo) GetGalleriesByTags(
	ctx context.Context,
	tags []string, // Теги для фильтрации
	matchAll bool, // true: AND-фильтр (все теги), false: OR-фильтр (любой из тегов)
) ([]models.Gallery, error) {
	const op = "repository.TagRepository.GetGalleriesByTags"

	// Базовый запрос
	queryBuilder := b.sb.Select(
		"id", "title", "slug", "description", "images",
		"cover_image_index", "author_id", "status",
		"metadata", "tags", "created_at", "updated_at",
	).From("galleries")

	// Применяем фильтр по тегам
	if len(tags) > 0 {
		if matchAll {
			// AND-условие: галерея должна содержать ВСЕ указанные теги
			queryBuilder = queryBuilder.Where("tags @> ?", pq.Array(tags))
		} else {
			// OR-условие: галерея должна содержать ЛЮБОЙ из указанных тегов
			queryBuilder = queryBuilder.Where("tags && ?", pq.Array(tags))
		}
	}

	// Формируем SQL-запрос
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Выполняем запрос
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	// Сканируем результаты
	var galleries []models.Gallery
	for rows.Next() {
		var gallery models.Gallery
		err := rows.Scan(
			&gallery.ID,
			&gallery.Title,
			&gallery.Slug,
			&gallery.Description,
			&gallery.Images,
			&gallery.CoverImageIndex,
			&gallery.AuthorID,
			&gallery.Status,
			&gallery.Metadata,
			&gallery.Tags,
			&gallery.CreatedAt,
			&gallery.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		galleries = append(galleries, gallery)
	}

	return galleries, nil
}

// AddTags добавляет теги к галерее
// func (b *GalleryRepo) AddTags(ctx context.Context, galleryID string, tags []string) error {
// 	const op = "repository.TagRepository.AddTags"

// 	query := `
// 		UPDATE galleries
// 		SET tags = array_cat(tags, $1)
// 		WHERE id = $2
// 	`

// 	_, err := b.db.Exec(ctx, query, pq.Array(tags), galleryID)
// 	if err != nil {
// 		return fmt.Errorf("%s: %w", op, err)
// 	}

// 	return nil
// }

// // RemoveTags удаляет теги из галереи
// func (b *GalleryRepo) RemoveTags(ctx context.Context, galleryID string, tags []string) error {
// 	const op = "repository.TagRepository.RemoveTags"

// 	query := `
// 		UPDATE galleries
// 		SET tags = array_remove(tags, unnest($1))
// 		WHERE id = $2
// 	`

// 	_, err := b.db.Exec(ctx, query, pq.Array(tags), galleryID)
// 	if err != nil {
// 		return fmt.Errorf("%s: %w", op, err)
// 	}

// 	return nil
// }

// // UpdateTags полностью обновляет теги галереи
// func (b *GalleryRepo) UpdateTags(ctx context.Context, galleryID string, tags []string) error {
// 	const op = "repository.TagRepository.UpdateTags"

// 	query := `
// 		UPDATE galleries
// 		SET tags = $1
// 		WHERE id = $2
// 	`

// 	_, err := b.db.Exec(ctx, query, pq.Array(tags), galleryID)
// 	if err != nil {
// 		return fmt.Errorf("%s: %w", op, err)
// 	}

// 	return nil
// }

// // HasTags проверяет, содержит ли галерея указанные теги
// func (b *GalleryRepo) HasTags(ctx context.Context, galleryID string, tags []string) (bool, error) {
// 	const op = "repository.TagRepository.HasTags"

// 	query := `
// 		SELECT COUNT(*) > 0
// 		FROM galleries
// 		WHERE id = $1 AND tags @> $2
// 	`

// 	var hasTags bool
// 	err := b.db.QueryRow(ctx, query, galleryID, pq.Array(tags)).Scan(&hasTags)
// 	if err != nil {
// 		return false, fmt.Errorf("%s: %w", op, err)
// 	}

// 	return hasTags, nil
// }

// // GetTags возвращает список тегов галереи
// func (b *GalleryRepo) GetTags(ctx context.Context, galleryID string) ([]string, error) {
// 	const op = "repository.TagRepository.GetTags"

// 	query := `
// 		SELECT tags
// 		FROM galleries
// 		WHERE id = $1
// 	`

// 	var tags []string
// 	err := b.db.QueryRow(ctx, query, galleryID).Scan(pq.Array(&tags))
// 	if err != nil {
// 		return nil, fmt.Errorf("%s: %w", op, err)
// 	}

// 	return tags, nil
// }
