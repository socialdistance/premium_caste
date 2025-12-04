package repository

import (
	"context"
	"time"

	"premium_caste/internal/domain/models"

	"github.com/google/uuid"
)

type UserRepository interface {
	SaveUser(ctx context.Context, user models.User) (uuid.UUID, error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	// User(ctx context.Context, email string) (models.User, error)
	UserByIdentifier(ctx context.Context, identifier string) (models.User, error)
	GetUserById(ctx context.Context, userID uuid.UUID) (models.User, error)
}

type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, userID, token string, exp time.Duration) error
	GetRefreshToken(ctx context.Context, userID, token string) (bool, error)
	DeleteRefreshToken(ctx context.Context, userID, token string) error
	DeleteAllUserTokens(ctx context.Context, userID string) error
}

type MediaRepository interface {
	CreateMedia(ctx context.Context, media *models.Media) (*models.Media, error)
	CreateMultipleMedia(ctx context.Context, medias []*models.Media) ([]*models.Media, error)
	AddMediaGroup(ctx context.Context, ownerID uuid.UUID, description string) (uuid.UUID, error)
	AddMediaGroupItems(ctx context.Context, groupID uuid.UUID, mediaIDs []uuid.UUID) error
	UpdateMedia(ctx context.Context, media *models.Media) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error)
	GetMediaByGroupID(ctx context.Context, groupID uuid.UUID) ([]models.Media, error)
	GetAllImages(ctx context.Context, limit int) ([]models.Media, int, error)
	GetImages(ctx context.Context) ([]models.Media, error)
}

type BlogRepository interface {
	SaveBlogPost(ctx context.Context, blogPost models.BlogPost) (uuid.UUID, error)
	UpdateBlogPostFields(ctx context.Context, postID uuid.UUID, updates map[string]interface{}) error
	DeleteBlogPost(ctx context.Context, postID uuid.UUID) error
	SoftDeleteBlogPost(ctx context.Context, postID uuid.UUID) error
	AddMediaGroupToPost(ctx context.Context, postID, groupID uuid.UUID, relationType string) error
	GetPostMediaGroups(ctx context.Context, postID uuid.UUID, relationType string) ([]uuid.UUID, error)
	GetBlogPosts(ctx context.Context, statusFilter string, page int, perPage int) ([]models.BlogPost, int, error)
	GetBlogPostByID(ctx context.Context, postID uuid.UUID) (*models.BlogPost, error)
}

type GalleryRepository interface {
	CreateGallery(ctx context.Context, gallery models.Gallery) (uuid.UUID, error)
	UpdateGallery(ctx context.Context, gallery models.Gallery) error
	UpdateGalleryStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteGallery(ctx context.Context, id uuid.UUID) error
	GetGalleryByID(ctx context.Context, id uuid.UUID) (models.Gallery, error)
	GetGalleries(ctx context.Context, statusFilter string, page int, perPage int) ([]models.Gallery, int, error)
	GetGalleriesByTags(ctx context.Context, tags []string, matchAll bool) ([]models.Gallery, error)
	AddTags(ctx context.Context, galleryID string, tags []string) error
	RemoveTags(ctx context.Context, galleryID string, tagsToRemove []string) error
	UpdateTags(ctx context.Context, galleryID string, tags []string) error
	HasTags(ctx context.Context, galleryID string, tags []string) (bool, error)
	GetTags(ctx context.Context, galleryID string) ([]string, error)
}
