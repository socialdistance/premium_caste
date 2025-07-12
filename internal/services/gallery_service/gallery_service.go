package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/repository"
	"premium_caste/internal/transport/http/dto"
	"strings"

	"github.com/google/uuid"
)

type GalleryService struct {
	log  *slog.Logger
	repo repository.GalleryRepository
}

func NewGalleryService(log *slog.Logger, repo repository.GalleryRepository) *GalleryService {
	return &GalleryService{
		log:  log,
		repo: repo,
	}
}

// CreateGallery создает новую галерею
func (s *GalleryService) CreateGallery(ctx context.Context, req dto.CreateGalleryRequest) (uuid.UUID, error) {
	const op = "service.GalleryService.CreateGallery"
	log := s.log.With(
		slog.String("op", op),
		slog.String("title", req.Title),
	)

	log.Info("creating gallery")

	// Валидация данных галереи
	if req.Title == "" {
		log.Error("title is required")
		return uuid.Nil, fmt.Errorf("title is required")
	}

	if len(req.Images) == 0 {
		return uuid.Nil, fmt.Errorf("images are required")
	}

	if req.AuthorID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("author_id is required")
	}

	gallery := models.Gallery{
		Title:           req.Title,
		Slug:            req.Slug,
		Description:     req.Description,
		Images:          req.Images,
		CoverImageIndex: req.CoverImageIndex,
		AuthorID:        req.AuthorID,
		Status:          req.Status,
		Tags:            req.Tags,
		Metadata:        req.Metadata,
	}

	id, err := s.repo.CreateGallery(ctx, gallery)
	if err != nil {
		log.Error("failed to create gallery", slog.Any("err", err))
		return uuid.Nil, fmt.Errorf("failed to create gallery: %w", err)
	}

	log.Info("gallery created successfully", slog.String("id", id.String()))
	return id, nil
}

// UpdateGallery обновляет данные галереи
func (s *GalleryService) UpdateGallery(ctx context.Context, req dto.UpdateGalleryRequest) error {
	const op = "service.GalleryService.UpdateGallery"
	log := s.log.With(
		slog.String("op", op),
		slog.String("gallery_id", req.ID.String()),
	)

	log.Info("updating gallery")

	// Валидация данных галереи
	if req.Title == "" {
		log.Error("title is required")
		return fmt.Errorf("title is required")
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}

	if req.Metadata == nil {
		req.Metadata = map[string]interface{}{}
	}

	if len(req.Images) == 0 {
		req.Images = []string{}
	}

	gallery := models.Gallery{
		ID:          req.ID,
		Title:       req.Title,
		Slug:        req.Slug,
		Images:      req.Images,
		Description: req.Description,
		Status:      req.Status,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
	}

	err := s.repo.UpdateGallery(ctx, gallery)
	if err != nil {
		log.Error("failed to update gallery", slog.Any("err", err))
		return fmt.Errorf("failed to update gallery: %w", err)
	}

	log.Info("gallery updated successfully")
	return nil
}

// UpdateGalleryStatus обновляет статус галереи
func (s *GalleryService) UpdateGalleryStatus(ctx context.Context, id uuid.UUID, status string) error {
	const op = "service.GalleryService.UpdateGalleryStatus"
	log := s.log.With(
		slog.String("op", op),
		slog.String("gallery_id", id.String()),
		slog.String("status", status),
	)

	log.Info("updating gallery status")

	// Валидация статуса
	if status != "draft" && status != "published" && status != "archived" {
		log.Error("invalid status", slog.String("status", status))
		return fmt.Errorf("invalid status: %s", status)
	}

	err := s.repo.UpdateGalleryStatus(ctx, id, status)
	if err != nil {
		log.Error("failed to update gallery status", slog.Any("err", err))
		return fmt.Errorf("failed to update gallery status: %w", err)
	}

	log.Info("gallery status updated successfully")
	return nil
}

// DeleteGallery удаляет галерею
func (s *GalleryService) DeleteGallery(ctx context.Context, id uuid.UUID) error {
	const op = "service.GalleryService.DeleteGallery"
	log := s.log.With(
		slog.String("op", op),
		slog.String("gallery_id", id.String()),
	)

	log.Info("deleting gallery")

	err := s.repo.DeleteGallery(ctx, id)
	if err != nil {
		log.Error("failed to delete gallery", slog.Any("err", err))
		return fmt.Errorf("failed to delete gallery: %w", err)
	}

	log.Info("gallery deleted successfully")
	return nil
}

// GetGalleryByID возвращает галерею по ID
func (s *GalleryService) GetGalleryByID(ctx context.Context, id uuid.UUID) (*dto.GalleryResponse, error) {
	const op = "service.GalleryService.GetGalleryByID"
	log := s.log.With(
		slog.String("op", op),
		slog.String("gallery_id", id.String()),
	)

	log.Info("getting gallery")

	gallery, err := s.repo.GetGalleryByID(ctx, id)
	if err != nil {
		log.Error("failed to get gallery", slog.Any("err", err))
		return nil, fmt.Errorf("failed to get gallery: %w", err)
	}

	log.Info("gallery retrieved successfully")
	return s.mapToGalleryResponse(gallery), nil
}

// GetGalleries возвращает список галерей с пагинацией
func (s *GalleryService) GetGalleries(
	ctx context.Context,
	statusFilter string,
	page int,
	perPage int,
) ([]dto.GalleryResponse, int, error) {
	const op = "service.GalleryService.GetGalleries"
	log := s.log.With(
		slog.String("op", op),
		slog.String("status_filter", statusFilter),
		slog.Int("page", page),
		slog.Int("per_page", perPage),
	)

	log.Info("getting galleries")

	galleries, total, err := s.repo.GetGalleries(ctx, statusFilter, page, perPage)
	if err != nil {
		log.Error("failed to get galleries", slog.Any("err", err))
		return nil, 0, fmt.Errorf("failed to get galleries: %w", err)
	}

	// Преобразуем модели в DTO
	var galleryResponses []dto.GalleryResponse
	for _, gallery := range galleries {
		galleryResponses = append(galleryResponses, *s.mapToGalleryResponse(gallery))
	}

	log.Info("galleries retrieved successfully", slog.Int("total", total))
	return galleryResponses, total, nil
}

func (s *GalleryService) GetGalleriesByTags(ctx context.Context, tags []string, matchAll bool) ([]dto.GalleryResponse, error) {
	const op = "service.TagService.GetGalleriesByTags"
	log := s.log.With(
		slog.String("op", op),
		slog.Any("tags", tags),
		slog.Bool("match_all", matchAll),
	)

	log.Info("getting galleries by tags")

	// 1. Получаем галереи из репозитория
	galleries, err := s.repo.GetGalleriesByTags(ctx, tags, matchAll)
	if err != nil {
		log.Error("failed to get galleries by tags",
			slog.Any("err", err),
			slog.Any("input_tags", tags),
		)
		return nil, fmt.Errorf("%s: failed to get galleries: %w", op, err)
	}

	// 2. Преобразуем модели в DTO
	galleryResponses := make([]dto.GalleryResponse, 0, len(galleries))
	for _, gallery := range galleries {
		galleryResponses = append(galleryResponses, *s.mapToGalleryResponse(gallery))
	}

	log.Info("galleries by tags retrieved successfully",
		slog.Int("count", len(galleryResponses)),
	)
	return galleryResponses, nil
}

func (s *GalleryService) AddTags(ctx context.Context, galleryID string, tags []string) error {
	const op = "service.GalleryService.AddTags"
	log := s.log.With(
		slog.String("op", op),
		slog.String("galleryID", galleryID),
		slog.Any("tags", tags),
	)

	log.Info("adding tags to gallery")

	if err := validateGalleryID(galleryID); err != nil {
		log.Error("invalid gallery ID", slog.Any("err", err))
		return err
	}

	cleanedTags, err := normalizeTags(tags)
	if err != nil {
		log.Error("failed to normalize tags", slog.Any("err", err))
		return err
	}

	if len(cleanedTags) == 0 {
		log.Info("no tags to add")
		return nil
	}

	if err := s.repo.AddTags(ctx, galleryID, cleanedTags); err != nil {
		log.Error("failed to add tags", slog.Any("err", err))
		return err
	}

	log.Info("tags added successfully")
	return nil
}

func (s *GalleryService) RemoveTags(ctx context.Context, galleryID string, tagsToRemove []string) error {
	const op = "service.GalleryService.RemoveTags"
	log := s.log.With(
		slog.String("op", op),
		slog.String("galleryID", galleryID),
		slog.Any("tagsToRemove", tagsToRemove),
	)

	log.Info("removing tags from gallery")

	if err := validateGalleryID(galleryID); err != nil {
		log.Error("invalid gallery ID", slog.Any("err", err))
		return err
	}

	if len(tagsToRemove) == 0 {
		log.Info("no tags to remove")
		return nil
	}

	if err := s.repo.RemoveTags(ctx, galleryID, tagsToRemove); err != nil {
		log.Error("failed to remove tags", slog.Any("err", err))
		return err
	}

	log.Info("tags removed successfully")
	return nil
}

func (s *GalleryService) ReplaceTags(ctx context.Context, galleryID string, newTags []string) error {
	const op = "service.GalleryService.ReplaceTags"
	log := s.log.With(
		slog.String("op", op),
		slog.String("galleryID", galleryID),
		slog.Any("newTags", newTags),
	)

	log.Info("replacing tags in gallery")

	if err := validateGalleryID(galleryID); err != nil {
		log.Error("invalid gallery ID", slog.Any("err", err))
		return err
	}

	cleanedTags, err := normalizeTags(newTags)
	if err != nil {
		log.Error("failed to normalize tags", slog.Any("err", err))
		return err
	}

	if err := s.repo.UpdateTags(ctx, galleryID, cleanedTags); err != nil {
		log.Error("failed to replace tags", slog.Any("err", err))
		return err
	}

	log.Info("tags replaced successfully")
	return nil
}

func (s *GalleryService) GetTags(ctx context.Context, galleryID string) ([]string, error) {
	const op = "service.GalleryService.GetTags"
	log := s.log.With(
		slog.String("op", op),
		slog.String("galleryID", galleryID),
	)

	log.Info("getting tags for gallery")

	if err := validateGalleryID(galleryID); err != nil {
		log.Error("invalid gallery ID", slog.Any("err", err))
		return nil, err
	}

	tags, err := s.repo.GetTags(ctx, galleryID)
	if err != nil {
		log.Error("failed to get tags", slog.Any("err", err))
		return nil, err
	}

	log.Info("tags retrieved successfully", slog.Int("count", len(tags)))
	return tags, nil
}

func (s *GalleryService) HasTags(ctx context.Context, galleryID string, tags []string) (bool, error) {
	const op = "service.GalleryService.HasTags"
	log := s.log.With(
		slog.String("op", op),
		slog.String("galleryID", galleryID),
		slog.Any("tags", tags),
	)

	log.Info("checking if gallery has tags")

	if err := validateGalleryID(galleryID); err != nil {
		log.Error("invalid gallery ID", slog.Any("err", err))
		return false, err
	}

	if len(tags) == 0 {
		log.Info("no tags to check")
		return false, nil
	}

	hasTags, err := s.repo.HasTags(ctx, galleryID, tags)
	if err != nil {
		log.Error("failed to check tags", slog.Any("err", err))
		return false, err
	}

	log.Info("tags check completed", slog.Bool("hasTags", hasTags))
	return hasTags, nil
}

// validateGalleryID проверяет корректность UUID галереи
func validateGalleryID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("некорректный идентификатор галереи")
	}
	return nil
}

// normalizeTags приводит теги к единому формату и валидирует их
func normalizeTags(tags []string) ([]string, error) {
	var result []string
	seen := make(map[string]bool)

	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}

		if len(trimmed) > 50 {
			return nil, errors.New("тег не может быть длиннее 50 символов")
		}

		lower := strings.ToLower(trimmed)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, lower)
		}
	}

	return result, nil
}

// mapToGalleryResponse преобразует модель галереи в DTO
func (s *GalleryService) mapToGalleryResponse(gallery models.Gallery) *dto.GalleryResponse {
	return &dto.GalleryResponse{
		ID:              gallery.ID,
		Title:           gallery.Title,
		Slug:            gallery.Slug,
		Description:     gallery.Description,
		Images:          gallery.Images,
		CoverImageIndex: gallery.CoverImageIndex,
		AuthorID:        gallery.AuthorID,
		Status:          gallery.Status,
		PublishedAt:     gallery.PublishedAt,
		CreatedAt:       gallery.CreatedAt,
		UpdatedAt:       gallery.UpdatedAt,
		Metadata:        gallery.Metadata,
		Tags:            gallery.Tags,
	}
}
