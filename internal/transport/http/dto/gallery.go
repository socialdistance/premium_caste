package dto

import (
	"time"

	"github.com/google/uuid"
)

// GalleryResponse представляет собой DTO для ответа с данными о галерее
type GalleryResponse struct {
	ID              uuid.UUID   `json:"id"`                // Уникальный идентификатор галереи
	Title           string      `json:"title"`             // Название галереи
	Slug            string      `json:"slug"`              // Уникальный URL-идентификатор галереи
	Description     string      `json:"description"`       // Описание галереи
	Images          []string    `json:"images"`            // Список изображений в галерее
	CoverImageIndex int         `json:"cover_image_index"` // Индекс изображения, используемого как обложка
	AuthorID        uuid.UUID   `json:"author_id"`         // Идентификатор автора галереи
	Status          string      `json:"status"`            // Статус галереи (например, "draft", "published", "archived")
	PublishedAt     *time.Time  `json:"published_at"`      // Дата и время публикации (если галерея опубликована)
	CreatedAt       time.Time   `json:"created_at"`        // Дата и время создания галереи
	UpdatedAt       time.Time   `json:"updated_at"`        // Дата и время последнего обновления галереи
	Metadata        interface{} `json:"metadata"`          // Дополнительные метаданные (может быть произвольной структурой)
	Tags            []string    `json:"tags"`              // Список тегов, связанных с галереей
}

type CreateGalleryRequest struct {
	Title           string                 `json:"title" validate:"required"`
	Slug            string                 `json:"slug"`
	Description     string                 `json:"description"`
	Images          []string               `json:"images"`            // Список изображений в галерее
	CoverImageIndex int                    `json:"cover_image_index"` // Индекс изображения, используемого как обложка
	AuthorID        uuid.UUID              `json:"author_id" validate:"required"`
	Status          string                 `json:"status"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type UpdateGalleryRequest struct {
	ID              uuid.UUID              `json:"id" validate:"required"`
	Title           string                 `json:"title" validate:"required"`
	Slug            string                 `json:"slug"`
	Description     string                 `json:"description"`
	Images          []string               `json:"images"`            // Список изображений в галерее
	CoverImageIndex int                    `json:"cover_image_index"` // Индекс изображения, используемого как обложка
	Status          string                 `json:"status"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type UpdateGalleryStatusRequest struct {
	Status string `json:"status" validate:"required"`
}

type GalleryTagsRequest struct {
	Tags []string `json:"tags"`
}
