package services

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"path/filepath"

	"premium_caste/internal/domain/models"
	"premium_caste/internal/lib/logger/sl"
	"premium_caste/internal/repository"
	storage "premium_caste/internal/storage/filestorage"
	"premium_caste/internal/transport/http/dto"

	"time"

	"slices"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
)

type MediaService struct {
	log         *slog.Logger
	repo        repository.MediaRepository
	fileStorage storage.FileStorage
	cache       *cache.Cache
}

func NewMediaService(log *slog.Logger, repo repository.MediaRepository, fileStorage storage.FileStorage) *MediaService {
	return &MediaService{
		log:         log,
		repo:        repo,
		fileStorage: fileStorage,
		cache:       cache.New(5*time.Minute, 10*time.Minute), // Кеш с TTL 5 минут и очисткой каждые 10 минут
	}
}

func (s *MediaService) UploadMultipleMedia(ctx context.Context, inputs []dto.MediaUploadInput) ([]*models.Media, error) {
	const op = "media_service.UploadMultipleMedia"

	log := s.log.With(
		slog.String("op", op),
	)

	log.Info("Upload multiple media", slog.Int("count", len(inputs)))

	// Подготовка данных для хранения
	var (
		fileHeaders = make([]*multipart.FileHeader, 0, len(inputs))
		uploaderID  = uuid.Nil
	)

	// Проверяем что все файлы от одного загрузчика
	for _, input := range inputs {
		if uploaderID == uuid.Nil {
			uploaderID = input.UploaderID
		} else if uploaderID != input.UploaderID {
			return nil, fmt.Errorf("%s: all files must have same uploader ID", op)
		}
		fileHeaders = append(fileHeaders, input.File)
	}

	// Сохраняем файлы
	paths, sizes, err := s.fileStorage.SaveMultiple(ctx, fileHeaders, filepath.Join("uploads", uploaderID.String()))
	if err != nil {
		log.Error("failed to save files", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Создаем модели медиа
	medias := make([]*models.Media, 0, len(inputs))
	for i, input := range inputs {
		media := &models.Media{
			ID:               uuid.New(),
			UploaderID:       input.UploaderID,
			CreatedAt:        time.Now().UTC(),
			MediaType:        models.MediaType(input.MediaType),
			OriginalFilename: input.File.Filename,
			StoragePath:      paths[i],
			FileSize:         sizes[i],
			MimeType:         input.File.Header.Get("Content-Type"),
			Width:            input.Width,
			Height:           input.Height,
			Duration:         input.Duration,
			IsPublic:         input.IsPublic,
			Metadata:         input.CustomMetadata,
		}

		if err := media.Validate(); err != nil {
			// Удаляем все сохраненные файлы при ошибке валидации
			if cleanupErr := s.cleanupFiles(ctx, paths, log); cleanupErr != nil {
				log.Error("failed to cleanup files after validation error",
					sl.Err(err), sl.Err(cleanupErr))
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		medias = append(medias, media)
	}

	// Сохраняем в базу данных

	createdMedias, err := s.repo.CreateMultipleMedia(ctx, medias)
	if err != nil {
		// Удаляем все сохраненные файлы при ошибке базы данных
		if cleanupErr := s.cleanupFiles(ctx, paths, log); cleanupErr != nil {
			log.Error("failed to cleanup files after db error",
				sl.Err(err), sl.Err(cleanupErr))
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return createdMedias, nil
}

func (s *MediaService) UploadMedia(ctx context.Context, input dto.MediaUploadInput) (*models.Media, error) {
	const op = "media_service.UploadMedia"

	log := s.log.With(
		slog.String("op", op),
		slog.String("media_type", input.MediaType),
	)

	log.Info("Upload media")

	filePath, fileSize, err := s.fileStorage.Save(ctx, input.File, filepath.Join("uploads", input.UploaderID.String()))
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

func (s *MediaService) AttachMediaToGroup(ctx context.Context, groupID uuid.UUID, mediaIDs []uuid.UUID) error {
	const op = "media_service.AttachMediaToGroupItems"

	log := s.log.With(
		"op", op,
		"groupID", groupID,
		"mediaIDs", mediaIDs, // Логируем весь массив
	)

	// Валидация параметров
	if groupID == uuid.Nil {
		log.Info("groupID is required", "op", op)
		return fmt.Errorf("%s: groupID is required", op)
	}
	if len(mediaIDs) == 0 {
		log.Info("at least one mediaID is required", "op", op)
		return fmt.Errorf("%s: at least one mediaID is required", op)
	}

	// Проверка на нулевые UUID в массиве
	if slices.Contains(mediaIDs, uuid.Nil) {
		log.Info("mediaID cannot be nil", "op", op)
		return fmt.Errorf("%s: mediaID cannot be nil", op)
	}

	// Вызов репозитория с массивом mediaIDs
	if err := s.repo.AddMediaGroupItems(ctx, groupID, mediaIDs); err != nil {
		log.Error("%s: failed to attach media: %s", op, sl.Err(err))
		return fmt.Errorf("%s: failed to attach media: %w", op, err)
	}

	log.Debug("media files attached to group",
		"op", op,
		"groupID", groupID,
		"mediaCount", len(mediaIDs), // Логируем количество прикрепленных файлов
	)

	return nil
}

func (s *MediaService) AttachMedia(ctx context.Context, ownerID uuid.UUID, description string) (uuid.UUID, error) {
	const op = "media_service.AttachMedia"

	log := s.log.With(
		"op", op,
		"ownerID", ownerID,
		"description", description,
	)

	if ownerID == uuid.Nil {
		log.Info("ownerID is required", "op", op)
		return uuid.Nil, fmt.Errorf("%s: ownerID is required", op)
	}

	groupID, err := s.repo.AddMediaGroup(ctx, ownerID, description)
	if err != nil {
		log.Error("%s: failed to attach media: %s", op, sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: failed to attach media: %w", op, err)
	}

	log.Debug("media attached to group",
		"op", op,
		"groupID", ownerID,
		"description", description,
	)

	return groupID, nil
}

func (s *MediaService) ListGroupMedia(ctx context.Context, groupID uuid.UUID) ([]models.Media, error) {
	const op = "media_service.ListGroupMedia"

	log := s.log.With(
		"op", op,
		"groupID", groupID,
	)

	if groupID == uuid.Nil {
		log.Info("groupID is required", "op", op)
		return []models.Media{}, fmt.Errorf("%s: groupID is required", op)
	}

	media, err := s.repo.GetMediaByGroupID(ctx, groupID)
	if err != nil {
		log.Error("failed get media list: %s %w", op, sl.Err(err))
		return []models.Media{}, fmt.Errorf("failed get media list: %s %s", op, sl.Err(err))
	}

	log.Debug("list media from group",
		"op", op,
		"groupID", groupID,
		"mediaList", media,
	)

	return media, nil
}

func (s *MediaService) cleanupFile(ctx context.Context, path string, log *slog.Logger) error {
	if err := s.fileStorage.Delete(ctx, path); err != nil {
		log.Error("file deletion failed", slog.String("path", path), sl.Err(err))
		return err
	}
	return nil
}

func (s *MediaService) cleanupFiles(ctx context.Context, paths []string, log *slog.Logger) error {
	var errs []error
	for _, path := range paths {
		if err := s.fileStorage.Delete(ctx, path); err != nil {
			errs = append(errs, err)
			log.Error("failed to delete file",
				slog.String("path", path), sl.Err(err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multiple errors during cleanup: %v", errs)
	}
	return nil
}

func (s *MediaService) GetAllImages(ctx context.Context, limit int) ([]models.Media, int, error) {
	const op = "media_service.GetAllImages"

	log := s.log.With(
		"op", op,
	)

	// Формируем ключ для кеша на основе параметров
	cacheKey := fmt.Sprintf("all_images:%d", limit)

	// Пытаемся получить данные из кеша
	if cachedData, found := s.cache.Get(cacheKey); found {
		log.Info("cache hit", "key", cacheKey)
		return cachedData.([]models.Media), 0, nil
	}

	log.Info("cache miss", "key", cacheKey)

	// Если данных нет в кеше, запрашиваем их из репозитория
	media, total, err := s.repo.GetAllImages(ctx, limit)
	if err != nil {
		log.Error("failed get media", "op", op, "error", sl.Err(err))
		return []models.Media{}, 0, fmt.Errorf("failed get media list: %s %w", op, err)
	}

	// Сохраняем данные в кеш
	s.cache.Set(cacheKey, media, cache.DefaultExpiration)

	return media, total, nil
}

// TODO: добавить кеш
// func (s *MediaService) GetAllImages(ctx context.Context, limit int) ([]models.Media, error) {
// 	const op = "media_service.GetAllImages"

// 	log := s.log.With(
// 		"op", op,
// 	)

// 	media, err := s.repo.GetAllImages(ctx, limit)
// 	if err != nil {
// 		log.Error("failed get media: %s %w", op, sl.Err(err))
// 		return []models.Media{}, fmt.Errorf("failed get media list: %s %s", op, sl.Err(err))
// 	}

// 	return media, nil
// }
