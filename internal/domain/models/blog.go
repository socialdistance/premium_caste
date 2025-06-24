package models

import (
	"time"

	"github.com/google/uuid"
)

type BlogPost struct {
	ID                uuid.UUID              `db:"id" json:"id"`
	Title             string                 `db:"title" json:"title"`
	Slug              string                 `db:"slug" json:"slug"`
	Excerpt           string                 `db:"excerpt" json:"excerpt,omitempty"`
	Content           string                 `db:"content" json:"content"`
	FeaturedImageID   uuid.UUID              `db:"featured_image_id" json:"featured_image_id,omitempty"`
	FeaturedImagePath *string                `json:"featured_image_path"`
	AuthorID          uuid.UUID              `db:"author_id" json:"author_id"`
	Status            string                 `db:"status" json:"status"`
	PublishedAt       *time.Time             `db:"published_at" json:"published_at,omitempty"`
	CreatedAt         time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time              `db:"updated_at" json:"updated_at"`
	Metadata          map[string]any         `db:"metadata" json:"metadata,omitempty"`
	MediaGroups       map[string][]MediaItem `json:"media_groups"`
}

type PostMediaGroup struct {
	PostID       uuid.UUID `db:"post_id" json:"post_id"`
	GroupID      uuid.UUID `db:"group_id" json:"group_id"`
	RelationType string    `db:"relation_type" json:"relation_type"`
}

// Структура для объединения структуры поста
type MediaItem struct {
	ID          uuid.UUID `json:"id"`
	StoragePath string    `json:"storage_path"`
	Position    int       `json:"position"`
	GroupID     uuid.UUID `json:"group_id"`
}
