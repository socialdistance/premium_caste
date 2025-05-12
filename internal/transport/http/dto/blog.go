package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateBlogPostRequest struct {
	Title           string         `json:"title" validate:"required,min=3,max=100"`
	Slug            string         `json:"slug,omitempty" validate:"omitempty,slug"`
	Excerpt         string         `json:"excerpt,omitempty" validate:"omitempty,max=255"`
	Content         string         `json:"content" validate:"required"`
	FeaturedImageID uuid.UUID      `json:"featured_image_id,omitempty" swaggertype:"string" format:"uuid"`
	AuthorID        uuid.UUID      `json:"author_id" validate:"required" swaggertype:"string" format:"uuid"`
	Status          string         `json:"status,omitempty" validate:"omitempty,oneof=draft published archived"`
	PublishedAt     *time.Time     `json:"published_at,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type CreateBlogPostResponse struct {
	ID          uuid.UUID  `json:"id" swaggertype:"string" format:"uuid"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type UpdateBlogPostRequest struct {
	Title           *string        `json:"title,omitempty" validate:"omitempty,min=3,max=100"`
	Slug            *string        `json:"slug,omitempty" validate:"omitempty,slug"`
	Excerpt         *string        `json:"excerpt,omitempty" validate:"omitempty,max=255"`
	Content         *string        `json:"content,omitempty"`
	FeaturedImageID *uuid.UUID     `json:"featured_image_id,omitempty" swaggertype:"string" format:"uuid"`
	Status          *string        `json:"status,omitempty" validate:"omitempty,oneof=draft published archived"`
	PublishedAt     *time.Time     `json:"published_at,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type UpdateBlogPostResponse struct {
	ID          uuid.UUID  `json:"id" swaggertype:"string" format:"uuid"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type BlogPostResponse struct {
	ID              uuid.UUID      `json:"id" swaggertype:"string" format:"uuid"`
	Title           string         `json:"title"`
	Slug            string         `json:"slug"`
	Excerpt         string         `json:"excerpt,omitempty"`
	Content         string         `json:"content"`
	FeaturedImageID uuid.UUID      `json:"featured_image_id,omitempty" swaggertype:"string" format:"uuid"`
	AuthorID        uuid.UUID      `json:"author_id" swaggertype:"string" format:"uuid"`
	Status          string         `json:"status"`
	PublishedAt     *time.Time     `json:"published_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type BlogPostListResponse struct {
	Posts      []BlogPostResponse `json:"posts"`
	TotalCount int                `json:"total_count"`
	Page       int                `json:"page"`
	PerPage    int                `json:"per_page"`
}

type AddMediaGroupRequest struct {
	GroupID      uuid.UUID `json:"group_id" validate:"required" swaggertype:"string" format:"uuid"`
	RelationType string    `json:"relation_type" validate:"required,oneof=content gallery attachment"`
}

type MediaGroupResponse struct {
	GroupID      uuid.UUID `json:"group_id" swaggertype:"string" format:"uuid"`
	RelationType string    `json:"relation_type"`
	AddedAt      time.Time `json:"added_at"`
}

type PostMediaGroupsResponse struct {
	PostID uuid.UUID            `json:"post_id" swaggertype:"string" format:"uuid"`
	Groups []MediaGroupResponse `json:"groups"`
}

type ChangePostStatusRequest struct {
	Status      string     `json:"status" validate:"required,oneof=published draft archived"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type SlugAvailabilityRequest struct {
	Slug string `json:"slug" validate:"required,slug"`
}

type SlugAvailabilityResponse struct {
	Available bool `json:"available"`
}
