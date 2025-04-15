package service

import (
	"context"
	"fmt"
	"premium_caste/internal/domain/models"
	"premium_caste/pkg/dto"
	"time"

	"github.com/google/uuid"
)

// t

type MediaService struct {
	repo        repository.MediaRepository
	fileStorage storage.FileStorage
}

func NewMediaService(repo repository.MediaRepository, fileStorage storage.FileStorage) *MediaService {
	return &MediaService{
		repo:        repo,
		fileStorage: fileStorage,
	}
}

func (s *MediaService) UploadMedia(ctx context.Context, input dto.MediaUploadInput) (*models.Media, error) {
	// 1. Сохраняем файл в хранилище
	filePath, fileSize, err := s.fileStorage.Save(ctx, input.File)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
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
		return nil, fmt.Errorf("media validation failed: %w", err)
	}

	// 4. Сохраняем в БД
	createdMedia, err := s.repo.CreateMedia(ctx, media)
	if err != nil {
		// Удаляем файл если не удалось сохранить в БД
		_ = s.fileStorage.Delete(ctx, filePath)
		return nil, fmt.Errorf("failed to save media to database: %w", err)
	}

	return createdMedia, nil
}
