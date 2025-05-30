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
	AddMediaGroup(ctx context.Context, ownerID uuid.UUID, description string) error
	AddMediaGroupItems(ctx context.Context, groupID, mediaID uuid.UUID) error
	UpdateMedia(ctx context.Context, media *models.Media) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error)
	GetMediaByGroupID(ctx context.Context, groupID uuid.UUID) ([]models.Media, error)
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
