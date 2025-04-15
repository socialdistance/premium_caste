package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MediaType string

type Metadata map[string]interface{}

const (
	MediaTypePhoto    MediaType = "photo"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
)

// Media представляет медиафайл в системе
type Media struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UploaderID       uuid.UUID `json:"uploader_id" db:"uploader_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	MediaType        MediaType `json:"media_type" db:"media_type"`
	OriginalFilename string    `json:"original_filename" db:"original_filename"`
	StoragePath      string    `json:"storage_path" db:"storage_path"`
	FileSize         int64     `json:"file_size" db:"file_size"`
	MimeType         string    `json:"mime_type,omitempty" db:"mime_type"`
	Width            *int      `json:"width,omitempty" db:"width"`
	Height           *int      `json:"height,omitempty" db:"height"`
	Duration         *int      `json:"duration,omitempty" db:"duration"`
	IsPublic         bool      `json:"is_public" db:"is_public"`
	Metadata         Metadata  `json:"metadata,omitempty" db:"metadata"`
}

// MediaGroup представляет группу медиафайлов
type MediaGroup struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OwnerID     uuid.UUID `json:"owner_id" db:"owner_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	Description string    `json:"description,omitempty" db:"description"`
}

// MediaGroupItem представляет связь между медиа и группой
type MediaGroupItem struct {
	GroupID   uuid.UUID `json:"group_id" db:"group_id"`
	MediaID   uuid.UUID `json:"media_id" db:"media_id"`
	Position  int       `json:"position" db:"position"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// MediaWithGroups объединяет информацию о медиа и его группах
type MediaWithGroups struct {
	Media
	Groups []MediaGroup `json:"groups,omitempty"`
}

// Value реализует интерфейс driver.Valuer для сериализации Metadata в JSONB
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan реализует интерфейс sql.Scanner для десериализации JSONB в Metadata
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(b), m)
	}
	return json.Unmarshal(b, m)
}

// NewMedia создает новый экземпляр Media с заполненными обязательными полями
func NewMedia(uploaderID uuid.UUID, mediaType MediaType, filename, path string, size int64) *Media {
	return &Media{
		ID:               uuid.New(),
		UploaderID:       uploaderID,
		CreatedAt:        time.Now().UTC(),
		MediaType:        mediaType,
		OriginalFilename: filename,
		StoragePath:      path,
		FileSize:         size,
		IsPublic:         false,
		Metadata:         make(Metadata),
	}
}

// Validate проверяет корректность данных медиафайла
func (m *Media) Validate() error {
	var validationErrors []string

	// Проверка обязательных полей
	if m.UploaderID == uuid.Nil {
		validationErrors = append(validationErrors, "uploader ID is required")
	}
	if m.OriginalFilename == "" {
		validationErrors = append(validationErrors, "original filename is required")
	}
	if len(m.OriginalFilename) > 255 {
		validationErrors = append(validationErrors, "original filename must be 255 characters or less")
	}
	if m.StoragePath == "" {
		validationErrors = append(validationErrors, "storage path is required")
	}
	if m.FileSize <= 0 {
		validationErrors = append(validationErrors, "file size must be positive")
	}

	// Валидация типа медиа
	switch m.MediaType {
	case MediaTypePhoto, MediaTypeVideo, MediaTypeAudio, MediaTypeDocument:
		// Дополнительная валидация для специфичных типов
		if m.MediaType == MediaTypePhoto || m.MediaType == MediaTypeVideo {
			if m.Width == nil || m.Height == nil {
				validationErrors = append(validationErrors, "width and height are required for photos and videos")
			} else if *m.Width <= 0 || *m.Height <= 0 {
				validationErrors = append(validationErrors, "width and height must be positive values")
			}
		}

		if m.MediaType == MediaTypeVideo && m.Duration == nil {
			validationErrors = append(validationErrors, "duration is required for videos")
		}
	default:
		validTypes := []string{
			string(MediaTypePhoto),
			string(MediaTypeVideo),
			string(MediaTypeAudio),
			string(MediaTypeDocument),
		}
		validationErrors = append(validationErrors,
			fmt.Sprintf("invalid media type '%s', must be one of: %v",
				m.MediaType, validTypes))
	}

	// Валидация MIME-типа
	if m.MimeType != "" && len(m.MimeType) > 100 {
		validationErrors = append(validationErrors, "mime type must be 100 characters or less")
	}

	// Валидация метаданных
	if m.Metadata != nil {
		if jsonSize, err := json.Marshal(m.Metadata); err == nil {
			if len(jsonSize) > 1*1024*1024 { // 1MB
				validationErrors = append(validationErrors, "metadata too large (max 1MB)")
			}
		} else {
			validationErrors = append(validationErrors,
				fmt.Sprintf("invalid metadata format: %v", err))
		}
	}

	// Возвращаем все ошибки одной структурой
	if len(validationErrors) > 0 {
		return &MediaValidationError{
			Errors: validationErrors,
		}
	}

	return nil
}

// MediaValidationError кастомный тип ошибки для валидации
type MediaValidationError struct {
	Errors []string
}

func (e *MediaValidationError) Error() string {
	return fmt.Sprintf("media validation failed: %s", strings.Join(e.Errors, "; "))
}

// IsMediaValidationError проверяет, является ли ошибка ошибкой валидации
func IsMediaValidationError(err error) bool {
	_, ok := err.(*MediaValidationError)
	return ok
}
