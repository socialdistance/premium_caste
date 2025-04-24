package dto

import (
	"mime/multipart"
	"premium_caste/internal/domain/models"
	"time"

	"github.com/google/uuid"
)

type MediaUploadInput struct {
	UploaderID     uuid.UUID             `json:"uploader_id" validate:"required"`
	File           *multipart.FileHeader `json:"-" form:"file" validate:"required"`
	MediaType      string                `json:"media_type" validate:"required,oneof=photo video audio document"`
	IsPublic       bool                  `json:"is_public"`
	CustomMetadata map[string]any        `json:"metadata,omitempty"`

	// Опциональные поля для видео
	Duration *int `json:"duration,omitempty" validate:"omitempty,min=1"`

	// Опциональные поля для изображений/видео
	Width  *int `json:"width,omitempty" validate:"required,min=1"`
	Height *int `json:"height,omitempty" validate:"required,min=1"`
}

type CreateMediaGroupRequest struct {
	OwnerID     string `json:"owner_id" validate:"required,uuid"`
	Description string `json:"description"`
}

type AttachMediaRequest struct {
	GroupID string `json:"group_id" validate:"required,uuid"`
	MediaID string `json:"media_id" validate:"required,uuid"`
}

type ListGroupMediaRequest struct {
	GroupID string `json:"group_id" validate:"required,uuid"`
	Limit   int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset  int    `query:"offset" validate:"omitempty,min=0"`
}

// ToDomain преобразует DTO в доменную модель
func (input *MediaUploadInput) ToDomain(filePath string, fileSize int64) models.Media {
	media := models.Media{
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
	return media
}
