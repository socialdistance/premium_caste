package models

import (
	"time"

	"github.com/google/uuid"
)

// Gallery представляет собой модель галереи
type Gallery struct {
	ID              uuid.UUID   `json:"id"`                // Уникальный идентификатор галереи
	Title           string      `json:"title"`             // Заголовок галереи
	Slug            string      `json:"slug"`              // Уникальный URL-идентификатор
	Description     string      `json:"description"`       // Описание галереи
	Images          []string    `json:"images"`            // Массив путей/URL изображений
	CoverImageIndex int         `json:"cover_image_index"` // Индекс обложки в массиве images
	AuthorID        uuid.UUID   `json:"author_id"`         // ID автора галереи
	Status          string      `json:"status"`            // Статус галереи (например, "draft", "published")
	PublishedAt     *time.Time  `json:"published_at"`      // Дата публикации (может быть nil)
	CreatedAt       time.Time   `json:"created_at"`        // Дата создания
	UpdatedAt       time.Time   `json:"updated_at"`        // Дата последнего обновления
	Metadata        interface{} `json:"metadata"`          // Дополнительные метаданные (JSONB)
	Tags            []string    `json:"tags"`              // Массив тегов
}
