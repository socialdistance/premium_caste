package services

import (
	"context"
	"fmt"
	"log/slog"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/repository"
	"premium_caste/internal/transport/http/dto"

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
func (s *GalleryService) UpdateGallery(ctx context.Context, gallery models.Gallery) error {
	const op = "service.GalleryService.UpdateGallery"
	log := s.log.With(
		slog.String("op", op),
		slog.String("gallery_id", gallery.ID.String()),
	)

	log.Info("updating gallery")

	// Валидация данных галереи
	if gallery.Title == "" {
		log.Error("title is required")
		return fmt.Errorf("title is required")
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
