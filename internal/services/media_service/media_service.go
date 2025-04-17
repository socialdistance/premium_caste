package services

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/lib/logger/sl"
	"premium_caste/internal/repository"
	storage "premium_caste/internal/storage/filestorage"
	"premium_caste/internal/transport/http/dto"

	"time"

	"github.com/google/uuid"
)

type MediaService struct {
	log         *slog.Logger
	repo        repository.MediaRepository
	fileStorage storage.FileStorage
}

func NewMediaService(log *slog.Logger, repo repository.MediaRepository, fileStorage storage.FileStorage) *MediaService {
	return &MediaService{
		log:         log,
		repo:        repo,
		fileStorage: fileStorage,
	}
}

func (s *MediaService) UploadMedia(ctx context.Context, input dto.MediaUploadInput) (*models.Media, error) {
	const op = "media_service.UploadMedia"

	log := s.log.With(
		slog.String("op", op),
		slog.String("media_type", input.MediaType),
	)

	// log.Info("Upload media")

	filePath, fileSize, err := s.fileStorage.Save(ctx, input.File, filepath.Join("user_uploads", input.UploaderID.String()))
	if err != nil {
		log.Error("failed to save file", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	media := &models.Media{
		ID:               uuid.New(),
		UploaderID:       input.UploaderID,
		CreatedAt:        time.Now().UTC(),
		MediaType:        models.MediaType(input.MediaType),
		OriginalFilename: input.File.Filename,
		StoragePath:      filePath,
		FileSize:         fileSize,
		MimeType:         input.File.Header.Get("Content-Type"),
		Width:            input.Width,
		Height:           input.Height,
		Duration:         input.Duration,
		IsPublic:         input.IsPublic,
		Metadata:         input.CustomMetadata,
	}

	if err := media.Validate(); err != nil {
		if delErr := s.cleanupFile(ctx, filePath, log); delErr != nil {
			log.Error("failed to delete file after validation error",
				sl.Err(err), slog.String("delete_error", delErr.Error()))
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	createdMedia, err := s.repo.CreateMedia(ctx, media)
	if err != nil {
		if delErr := s.cleanupFile(ctx, filePath, log); delErr != nil {
			log.Error("failed to delete file after db error",
				sl.Err(err), slog.String("delete_error", delErr.Error()))
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return createdMedia, nil
}

func (s *MediaService) AttachMediaToGroup(ctx context.Context, groupID uuid.UUID, mediaID uuid.UUID) error {
	const op = "media_service.AttachMediaToGroupItems"

	log := s.log.With(
		"op", op,
		"groupID", groupID,
		"mediaID", mediaID,
	)

	if groupID == uuid.Nil {
		return fmt.Errorf("%s: groupID is required", op)
	}
	if mediaID == uuid.Nil {
		return fmt.Errorf("%s: mediaID is required", op)
	}

	if err := s.repo.AddMediaGroupItems(ctx, groupID, mediaID); err != nil {
		return fmt.Errorf("%s: failed to attach media: %w", op, err)
	}

	log.Debug("media attached to group",
		"op", op,
		"groupID", groupID,
		"mediaID", mediaID,
	)

	return nil
}

func (s *MediaService) AttachMedia(ctx context.Context, ownerID uuid.UUID, description string) error {
	const op = "media_service.AttachMedia"

	log := s.log.With(
		"op", op,
		"ownerID", ownerID,
		"description", description,
	)

	if ownerID == uuid.Nil {
		return fmt.Errorf("%s: ownerID is required", op)
	}

	if err := s.repo.AddMediaGroup(ctx, ownerID, description); err != nil {
		return fmt.Errorf("%s: failed to attach media: %w", op, err)
	}

	log.Debug("media attached to group",
		"op", op,
		"groupID", ownerID,
		"mediaID", description,
	)

	return nil
}

func (s *MediaService) cleanupFile(ctx context.Context, path string, log *slog.Logger) error {
	if err := s.fileStorage.Delete(ctx, path); err != nil {
		log.Error("file deletion failed", slog.String("path", path), sl.Err(err))
		return err
	}
	return nil
}
