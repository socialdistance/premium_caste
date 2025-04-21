package repository

import (
	"context"

	"premium_caste/internal/domain/models"

	"github.com/google/uuid"
)

type UserRepository interface {
	SaveUser(ctx context.Context, user models.User) (uuid.UUID, error)
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
	User(ctx context.Context, email string) (models.User, error)
}

type MediaRepository interface {
	CreateMedia(ctx context.Context, media *models.Media) (*models.Media, error)
	AddMediaGroup(ctx context.Context, ownerID uuid.UUID, description string) error
	AddMediaGroupItems(ctx context.Context, groupID, mediaID uuid.UUID) error
	UpdateMedia(ctx context.Context, media *models.Media) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error)
	GetMediaByGroupID(ctx context.Context, groupID uuid.UUID) ([]models.Media, error)
}
