package services

import (
	"context"
	"fmt"
	"log/slog"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/repository"
	"premium_caste/internal/transport/http/dto"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BlogService struct {
	log  *slog.Logger
	repo repository.BlogRepository
}

func NewBlogService(log *slog.Logger, repo repository.BlogRepository) *BlogService {
	return &BlogService{log: log, repo: repo}
}

// CreatePost создает новый пост с валидацией и обработкой slug
func (s *BlogService) CreatePost(ctx context.Context, req dto.CreateBlogPostRequest) (*dto.BlogPostResponse, error) {
	const op = "blog_service.CreatePost"
	log := s.log.With(
		slog.String("op", op),
		slog.String("author_id", req.AuthorID.String()),
	)

	log.Info("creating new blog post", slog.String("title", req.Title))

	// Валидация обязательных полей
	if req.Title == "" {
		log.Error("post title is required")
		return nil, fmt.Errorf("post title is required")
	}
	if req.AuthorID == uuid.Nil {
		log.Error("author ID is required")
		return nil, fmt.Errorf("author ID is required")
	}

	post := models.BlogPost{
		Title:           req.Title,
		Slug:            req.Slug,
		Excerpt:         req.Excerpt,
		Content:         req.Content,
		FeaturedImageID: req.FeaturedImageID,
		AuthorID:        req.AuthorID,
		Status:          req.Status,
		PublishedAt:     req.PublishedAt,
		Metadata:        req.Metadata,
	}

	// Генерация slug, если не указан
	if post.Slug == "" {
		post.Slug = generateSlug(post.Title)
		log.Debug("generated slug", slog.String("slug", post.Slug))
	}

	// Установка статуса по умолчанию
	if post.Status == "" {
		post.Status = "draft"
		log.Debug("set default status", slog.String("status", post.Status))
	}

	// Установка временных меток
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	// Если пост публикуется, устанавливаем published_at
	if post.Status == "published" && post.PublishedAt == nil {
		post.PublishedAt = &now
		log.Debug("set published_at", slog.Time("published_at", now))
	}

	// Сохранение в репозитории
	id, err := s.repo.SaveBlogPost(ctx, post)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			log.Warn("slug conflict detected, generating unique slug")
			post.Slug = generateUniqueSlug(post.Slug)
			id, err = s.repo.SaveBlogPost(ctx, post)
			if err != nil {
				log.Error("failed to create post after slug conflict", slog.Any("err", err))
				return nil, fmt.Errorf("failed to create post after slug conflict: %w", err)
			}
		} else {
			log.Error("failed to create post", slog.Any("err", err))
			return nil, fmt.Errorf("failed to create post: %w", err)
		}
	}

	log.Info("post created successfully", slog.String("post_id", id.String()))
	return s.toPostResponse(ctx, id)
}

// UpdatePost обновляет пост с валидацией и обработкой slug
func (s *BlogService) UpdatePost(ctx context.Context, postID uuid.UUID, req dto.UpdateBlogPostRequest) (*dto.BlogPostResponse, error) {
	const op = "blog_service.UpdatePost"
	log := s.log.With(
		slog.String("op", op),
		slog.String("post_id", postID.String()),
	)

	log.Info("updating blog post")

	// Проверяем существование поста
	existingPost, err := s.repo.GetBlogPostByID(ctx, postID)
	if err != nil {
		log.Error("failed to get post", slog.Any("err", err))
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	updates := make(map[string]interface{})

	// Обработка обновляемых полей
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Slug != nil {
		updates["slug"] = *req.Slug
	}
	if req.Excerpt != nil {
		updates["excerpt"] = *req.Excerpt
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.FeaturedImageID != nil {
		updates["featured_image_id"] = *req.FeaturedImageID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.PublishedAt != nil {
		updates["published_at"] = *req.PublishedAt
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}

	// Обработка slug при обновлении
	if slug, ok := updates["slug"].(string); ok && slug != existingPost.Slug {
		if slug == "" {
			if title, ok := updates["title"].(string); ok {
				updates["slug"] = generateSlug(title)
			} else {
				updates["slug"] = generateSlug(existingPost.Title)
			}
			log.Debug("generated new slug", slog.String("slug", updates["slug"].(string)))
		}
	}

	// Обработка статуса published
	if status, ok := updates["status"].(string); ok && status == "published" {
		if existingPost.Status != "published" {
			if _, ok := updates["published_at"]; !ok {
				now := time.Now()
				updates["published_at"] = &now
				log.Debug("set published_at", slog.Time("published_at", now))
			}
		}
	}

	// Обновление временной метки
	updates["updated_at"] = time.Now()

	// Вызов репозитория
	err = s.repo.UpdateBlogPostFields(ctx, postID, updates)
	if err != nil {
		log.Error("failed to update post", slog.Any("err", err))
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	log.Info("post updated successfully")
	return s.toPostResponse(ctx, postID)
}

// GetPostByID возвращает пост по ID
func (s *BlogService) GetPostByID(ctx context.Context, id uuid.UUID) (*dto.BlogPostResponse, error) {
	const op = "blog_service.GetPostByID"
	log := s.log.With(
		slog.String("op", op),
		slog.String("post_id", id.String()),
	)

	log.Info("getting blog post")

	post, err := s.repo.GetBlogPostByID(ctx, id)
	if err != nil {
		log.Error("failed to get post", slog.Any("err", err))
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	log.Info("post retrieved successfully")
	return s.mapToPostResponse(post), nil
}

// ListPosts возвращает список постов с пагинацией и фильтрацией
func (s *BlogService) ListPosts(ctx context.Context, statusFilter string, page, perPage int) (*dto.BlogPostListResponse, error) {
	const op = "blog_service.ListPosts"
	log := s.log.With(
		slog.String("op", op),
		slog.String("status_filter", statusFilter),
		slog.Int("page", page),
		slog.Int("per_page", perPage),
	)

	log.Info("listing blog posts")

	// Валидация параметров
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	posts, total, err := s.repo.GetBlogPosts(ctx, statusFilter, page, perPage)
	if err != nil {
		log.Error("failed to list posts", slog.Any("err", err))
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}

	response := &dto.BlogPostListResponse{
		Posts:      make([]dto.BlogPostResponse, 0, len(posts)),
		TotalCount: total,
		Page:       page,
		PerPage:    perPage,
	}

	for _, post := range posts {
		response.Posts = append(response.Posts, *s.mapToPostResponse(&post))
	}

	log.Info("posts listed successfully", slog.Int("count", len(posts)))
	return response, nil
}

// PublishPost публикует пост (устанавливает статус published)
func (s *BlogService) PublishPost(ctx context.Context, postID uuid.UUID) (*dto.BlogPostResponse, error) {
	const op = "blog_service.PublishPost"
	log := s.log.With(
		slog.String("op", op),
		slog.String("post_id", postID.String()),
	)

	log.Info("publishing blog post")

	now := time.Now()
	err := s.repo.UpdateBlogPostFields(ctx, postID, map[string]interface{}{
		"status":       "published",
		"published_at": &now,
		"updated_at":   now,
	})
	if err != nil {
		log.Error("failed to publish post", slog.Any("err", err))
		return nil, fmt.Errorf("failed to publish post: %w", err)
	}

	log.Info("post published successfully")
	return s.toPostResponse(ctx, postID)
}

// ArchivePost отправляет пост в архив
func (s *BlogService) ArchivePost(ctx context.Context, postID uuid.UUID) (*dto.BlogPostResponse, error) {
	const op = "blog_service.ArchivePost"
	log := s.log.With(
		slog.String("op", op),
		slog.String("post_id", postID.String()),
	)

	log.Info("archiving blog post")

	err := s.repo.SoftDeleteBlogPost(ctx, postID)
	if err != nil {
		log.Error("failed to archive post", slog.Any("err", err))
		return nil, fmt.Errorf("failed to archive post: %w", err)
	}

	log.Info("post archived successfully")
	return s.toPostResponse(ctx, postID)
}

// DeletePost удаляет пост (физическое удаление)
func (s *BlogService) DeletePost(ctx context.Context, postID uuid.UUID) error {
	const op = "blog_service.DeletePost"
	log := s.log.With(
		slog.String("op", op),
		slog.String("post_id", postID.String()),
	)

	log.Info("deleting blog post")

	err := s.repo.DeleteBlogPost(ctx, postID)
	if err != nil {
		log.Error("failed to delete post", slog.Any("err", err))
		return fmt.Errorf("failed to delete post: %w", err)
	}

	log.Info("post deleted successfully")
	return nil
}

// AddMediaGroup добавляет медиа-группу к посту
func (s *BlogService) AddMediaGroup(ctx context.Context, postID uuid.UUID, req dto.AddMediaGroupRequest) (*dto.PostMediaGroupsResponse, error) {
	const op = "blog_service.AddMediaGroup"
	log := s.log.With(
		slog.String("op", op),
		slog.String("post_id", postID.String()),
		slog.String("group_id", req.GroupID.String()),
		slog.String("relation_type", req.RelationType),
	)

	log.Info("adding media group to post")

	// Валидация relationType
	if req.RelationType != "content" && req.RelationType != "gallery" && req.RelationType != "attachment" {
		log.Error("invalid relation type")
		return nil, fmt.Errorf("invalid relation type")
	}

	err := s.repo.AddMediaGroupToPost(ctx, postID, req.GroupID, req.RelationType)
	if err != nil {
		log.Error("failed to add media group", slog.Any("err", err))
		return nil, fmt.Errorf("failed to add media group: %w", err)
	}

	log.Info("media group added successfully")
	return nil, err
}

// GetPostMediaGroups возвращает медиа-группы поста
func (s *BlogService) GetPostMediaGroups(ctx context.Context, postID uuid.UUID, relationType string) (*dto.PostMediaGroupsResponse, error) {
	const op = "blog_service.GetPostMediaGroups"
	log := s.log.With(
		slog.String("op", op),
		slog.String("post_id", postID.String()),
		slog.String("relation_type", relationType),
	)

	log.Info("getting post media groups")

	groups, err := s.repo.GetPostMediaGroups(ctx, postID, relationType)
	if err != nil {
		log.Error("failed to get media groups", slog.Any("err", err))
		return nil, fmt.Errorf("failed to get media groups: %w", err)
	}

	response := &dto.PostMediaGroupsResponse{
		PostID: postID,
		Groups: make([]dto.MediaGroupResponse, 0, len(groups)),
	}

	for _, group := range groups {
		response.Groups = append(response.Groups, dto.MediaGroupResponse{
			GroupID:      group.GroupID,
			RelationType: group.RelationType,
			AddedAt:      group.CreatedAt,
		})
	}

	log.Info("media groups retrieved successfully", slog.Int("count", len(groups)))
	return response, nil
}

// Вспомогательные функции
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "'", "")
	slug = strings.ReplaceAll(slug, `"`, "")
	return slug
}

func generateUniqueSlug(base string) string {
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func (s *BlogService) toPostResponse(ctx context.Context, postID uuid.UUID) (*dto.BlogPostResponse, error) {
	post, err := s.repo.GetBlogPostByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	return s.mapToPostResponse(post), nil
}

func (s *BlogService) mapToPostResponse(post *models.BlogPost) *dto.BlogPostResponse {
	return &dto.BlogPostResponse{
		ID:              post.ID,
		Title:           post.Title,
		Slug:            post.Slug,
		Excerpt:         post.Excerpt,
		Content:         post.Content,
		FeaturedImageID: post.FeaturedImageID,
		AuthorID:        post.AuthorID,
		Status:          post.Status,
		PublishedAt:     post.PublishedAt,
		CreatedAt:       post.CreatedAt,
		UpdatedAt:       post.UpdatedAt,
		Metadata:        post.Metadata,
	}
}
