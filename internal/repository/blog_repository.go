package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"premium_caste/internal/domain/models"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
)

type BlogRepo struct {
	db *pgxpool.Pool
	sb sq.StatementBuilderType
}

func NewBlogRepository(db *pgxpool.Pool) *BlogRepo {
	return &BlogRepo{
		db: db,
		sb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (b *BlogRepo) SaveBlogPost(ctx context.Context, blogPost models.BlogPost) (uuid.UUID, error) {
	const op = "repository.blog_repository.SaveBlogPost"

	query, args, err := b.sb.Insert("blog_posts").
		Columns(
			"title",
			"slug",
			"excerpt",
			"content",
			"featured_image_id",
			"author_id",
			"status",
			"metadata",
		).
		Values(
			blogPost.Title,
			blogPost.Slug,
			blogPost.Excerpt,
			blogPost.Content,
			blogPost.FeaturedImageID,
			blogPost.AuthorID,
			blogPost.Status,
			blogPost.Metadata,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	var id uuid.UUID
	err = b.db.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (b *BlogRepo) UpdateBlogPostFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error {
	const op = "repository.blog_repository.UpdateBlogPostFields"

	allowedFields := map[string]bool{
		"title":             true,
		"slug":              true,
		"excerpt":           true,
		"content":           true,
		"featured_image_id": true,
		"status":            true,
		"published_at":      true,
		"metadata":          true,
	}

	if len(updates) == 0 {
		return fmt.Errorf("%s: no fields to update", op)
	}

	updateBuilder := b.sb.Update("blog_posts").
		Set("updated_at", time.Now())

	for field, value := range updates {
		if !allowedFields[field] {
			return fmt.Errorf("%s: field '%s' is not allowed for update", op, field)
		}

		updateBuilder = updateBuilder.Set(field, value)
	}

	updateBuilder = updateBuilder.Where(sq.Eq{"id": postID})

	query, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = b.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil

	// Обновление заголовка и контента
	// err := repo.UpdateBlogPostFields(ctx, postID, map[string]interface{}{
	//     "title":   "Новый заголовок",
	//     "content": "Обновленный текст поста",
	// })

	// // Обновление статуса и даты публикации
	// err := repo.UpdateBlogPostFields(ctx, postID, map[string]interface{}{
	//     "status":       "published",
	//     "published_at": time.Now(),
	// })
}

// DeleteBlogPost -> обычное удаление из базы данных
func (b *BlogRepo) DeleteBlogPost(ctx context.Context, postID uuid.UUID) error {
	const op = "repository.blog_repository.DeleteBlogPost"

	query, args, err := b.sb.Delete("blog_posts").
		Where(sq.Eq{"id": postID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	result, err := b.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("%s: post with id %s not found", op, postID)
	}

	return nil
}

// Softdelete -> не удаляем пост, а помещаем его в архив
func (b *BlogRepo) SoftDeleteBlogPost(ctx context.Context, postID uuid.UUID) error {
	const op = "repository.blog_repository.SoftDeleteBlogPost"

	query, args, err := b.sb.Update("blog_posts").
		Set("status", "archived").
		Where(sq.Eq{"id": postID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = b.db.Exec(ctx, query, args...)
	return err
}

func (b *BlogRepo) GetBlogPostByID(ctx context.Context, postID uuid.UUID) (*models.BlogPost, error) {
	const op = "repository.blog_repository.GetBlogPostByID"

	queryBuilder := b.sb.Select(
		"id", "title", "slug", "excerpt", "content",
		"featured_image_id", "author_id", "status",
		"published_at", "created_at", "updated_at",
		"metadata",
	).
		From("blog_posts").
		Where(sq.Eq{"id": postID})

	sqlQuery, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	var post models.BlogPost
	var publishedAt sql.NullTime

	// Выполнение запроса
	err = b.db.QueryRow(ctx, sqlQuery, args...).Scan(
		&post.ID,
		&post.Title,
		&post.Slug,
		&post.Excerpt,
		&post.Content,
		&post.FeaturedImageID,
		&post.AuthorID,
		&post.Status,
		&publishedAt,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.Metadata,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s post not found: %w", op, err)
		}
		return nil, fmt.Errorf("%s failed to get post: %w", op, err)
	}

	if publishedAt.Valid {
		post.PublishedAt = &publishedAt.Time
	}

	return &post, nil
}

func (b *BlogRepo) GetBlogPosts(
	ctx context.Context,
	statusFilter string, // "all", "draft", "published", "archived"
	page int,
	perPage int,
) ([]models.BlogPost, int, error) {
	const op = "repository.blog_repository.GetBlogPosts"

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// Строим базовый запрос
	queryBuilder := b.sb.Select(
		"id", "title", "slug", "excerpt", "content", "featured_image_id", "author_id", "status", "published_at", "created_at", "updated_at",
	).From("blog_posts")

	// Применяем фильтр по статусу
	switch statusFilter {
	case "draft", "published", "archived":
		queryBuilder = queryBuilder.Where(sq.Eq{"status": statusFilter})
	case "all":

	default:
		return nil, 0, fmt.Errorf("%s: invalid status filter '%s'", op, statusFilter)
	}

	// Получаем общее количество постов (для пагинации)
	totalCount, err := b.getTotalCount(ctx)
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
	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	// Сканируем результаты
	var posts []models.BlogPost
	for rows.Next() {
		var post models.BlogPost
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Slug,
			&post.Excerpt,
			&post.Content,
			&post.FeaturedImageID,
			&post.AuthorID,
			&post.Status,
			&post.PublishedAt,
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", op, err)
		}
		posts = append(posts, post)
	}

	return posts, totalCount, nil

	// Получить первые 10 опубликованных постов
	// posts, total, err := repo.GetBlogPosts(ctx, "published", 1, 10)

	// // Получить черновики (страница 2, по 5 записей)
	// drafts, total, err := repo.GetBlogPosts(ctx, "draft", 2, 5)
}

func (b *BlogRepo) getTotalCount(ctx context.Context) (int, error) {
	queryBuilder := sq.Select("COUNT(*)").
		From("blog_posts")

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

// Добавление связи между постом и медиа-группой
// relationType -> content/gallery/attachment
func (b *BlogRepo) AddMediaGroupToPost(ctx context.Context, postID, groupID uuid.UUID, relationType string) error {
	const op = "repository.blog_repository.AddMediaGroupToPost"

	query, args, err := b.sb.Insert("post_media_groups").
		Columns("post_id", "group_id", "relation_type").
		Values(postID, groupID, relationType).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = b.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Получение всех медиа-групп поста с фильтрацией по типу связи
func (b *BlogRepo) GetPostMediaGroups(ctx context.Context, postID uuid.UUID, relationType string) ([]uuid.UUID, error) {
	const op = "repository.blog_repository.GetPostMediaGroups"

	queryBuilder := b.sb.Select("group_id").
		From("post_media_groups").
		Where(sq.Eq{"post_id": postID})

	if relationType != "" {
		queryBuilder = queryBuilder.Where(sq.Eq{"relation_type": relationType})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := b.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var groupIDs []uuid.UUID
	for rows.Next() {
		var groupID uuid.UUID
		if err := rows.Scan(&groupID); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		groupIDs = append(groupIDs, groupID)
	}

	return groupIDs, nil
}
