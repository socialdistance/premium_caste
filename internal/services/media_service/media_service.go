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

	log.Info("upload media")

	filePath, fileSize, err := s.fileStorage.Save(ctx, input.File, filepath.Join("user_uploads", input.UploaderID.String()))
	if err != nil {
		log.Error("failed to save file", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// 2. Создаем доменную модель
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

	// 3. Валидация
	if err := media.Validate(); err != nil {
		// Удаляем сохраненный файл при ошибке валидации
		_ = s.fileStorage.Delete(ctx, filePath)
		log.Error("media validation failed", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// 4. Сохраняем в БД
	createdMedia, err := s.repo.CreateMedia(ctx, media)
	if err != nil {
		// Удаляем файл если не удалось сохранить в БД
		_ = s.fileStorage.Delete(ctx, filePath)
		log.Error("failed to save media to database", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return createdMedia, nil
}
