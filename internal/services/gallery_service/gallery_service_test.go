package services

import (
	"context"
	"errors"
	"log/slog"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/transport/http/dto"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockGalleryRepository struct {
	mock.Mock
}

func (m *MockGalleryRepository) CreateGallery(ctx context.Context, gallery models.Gallery) (uuid.UUID, error) {
	args := m.Called(ctx, gallery)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockGalleryRepository) UpdateGallery(ctx context.Context, gallery models.Gallery) error {
	args := m.Called(ctx, gallery)
	return args.Error(0)
}

func (m *MockGalleryRepository) UpdateGalleryStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockGalleryRepository) DeleteGallery(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGalleryRepository) GetGalleryByID(ctx context.Context, id uuid.UUID) (models.Gallery, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.Gallery), args.Error(1)
}

func (m *MockGalleryRepository) GetGalleries(ctx context.Context, statusFilter string, page, perPage int) ([]models.Gallery, int, error) {
	args := m.Called(ctx, statusFilter, page, perPage)
	return args.Get(0).([]models.Gallery), args.Int(1), args.Error(2)
}

func TestGalleryService_CreateGallery(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	testUUID := uuid.New()
	gallery := dto.CreateGalleryRequest{
		Title:    "Test Gallery",
		Slug:     "test-gallery",
		AuthorID: uuid.New(),
	}

	tests := []struct {
		name        string
		gallery     dto.CreateGalleryRequest
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:    "successful creation",
			gallery: gallery,
			mockSetup: func() {
				mockRepo.On("CreateGallery", ctx, gallery).
					Return(testUUID, nil).Once()
			},
			wantError: false,
		},
		{
			name:    "missing title",
			gallery: dto.CreateGalleryRequest{},
			mockSetup: func() {
				// Нет вызова репозитория, так как валидация происходит до него
			},
			wantError:   true,
			expectedErr: "title is required",
		},
		{
			name:    "repository error",
			gallery: gallery,
			mockSetup: func() {
				mockRepo.On("CreateGallery", ctx, gallery).
					Return(uuid.Nil, errors.New("repository error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to create gallery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			id, err := service.CreateGallery(ctx, tt.gallery)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Equal(t, uuid.Nil, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testUUID, id)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGalleryService_UpdateGallery(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	gallery := dto.UpdateGalleryRequest{
		ID:    uuid.New(),
		Title: "Updated Gallery",
	}

	tests := []struct {
		name        string
		gallery     dto.UpdateGalleryRequest
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:    "successful update",
			gallery: gallery,
			mockSetup: func() {
				mockRepo.On("UpdateGallery", ctx, gallery).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:    "missing title",
			gallery: dto.UpdateGalleryRequest{ID: uuid.New()},
			mockSetup: func() {
				// Нет вызова репозитория, так как валидация происходит до него
			},
			wantError:   true,
			expectedErr: "title is required",
		},
		{
			name:    "repository error",
			gallery: gallery,
			mockSetup: func() {
				mockRepo.On("UpdateGallery", ctx, gallery).
					Return(errors.New("repository error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to update gallery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := service.UpdateGallery(ctx, tt.gallery)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGalleryService_UpdateGalleryStatus(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	id := uuid.New()
	status := "published"

	tests := []struct {
		name        string
		id          uuid.UUID
		status      string
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:   "successful status update",
			id:     id,
			status: status,
			mockSetup: func() {
				mockRepo.On("UpdateGalleryStatus", ctx, id, status).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:   "invalid status",
			id:     id,
			status: "invalid",
			mockSetup: func() {
				// Нет вызова репозитория, так как валидация происходит до него
			},
			wantError:   true,
			expectedErr: "invalid status",
		},
		{
			name:   "repository error",
			id:     id,
			status: status,
			mockSetup: func() {
				mockRepo.On("UpdateGalleryStatus", ctx, id, status).
					Return(errors.New("repository error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to update gallery status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := service.UpdateGalleryStatus(ctx, tt.id, tt.status)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGalleryService_DeleteGallery(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	id := uuid.New()

	tests := []struct {
		name        string
		id          uuid.UUID
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name: "successful deletion",
			id:   id,
			mockSetup: func() {
				mockRepo.On("DeleteGallery", ctx, id).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name: "repository error",
			id:   id,
			mockSetup: func() {
				mockRepo.On("DeleteGallery", ctx, id).
					Return(errors.New("repository error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to delete gallery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := service.DeleteGallery(ctx, tt.id)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGalleryService_GetGalleryByID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	id := uuid.New()
	gallery := models.Gallery{
		ID:    id,
		Title: "Test Gallery",
	}

	tests := []struct {
		name        string
		id          uuid.UUID
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name: "successful retrieval",
			id:   id,
			mockSetup: func() {
				mockRepo.On("GetGalleryByID", ctx, id).
					Return(gallery, nil).Once()
			},
			wantError: false,
		},
		{
			name: "repository error",
			id:   id,
			mockSetup: func() {
				mockRepo.On("GetGalleryByID", ctx, id).
					Return(models.Gallery{}, errors.New("repository error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to get gallery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.GetGalleryByID(ctx, tt.id)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, gallery.ID, resp.ID)
				assert.Equal(t, gallery.Title, resp.Title)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGalleryService_GetGalleries(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	galleries := []models.Gallery{
		{ID: uuid.New(), Title: "Gallery 1", Images: []string{"img2.jpg"}},
		{ID: uuid.New(), Title: "Gallery 2", Images: []string{"img2.jpg"}},
	}
	total := 2

	tests := []struct {
		name         string
		statusFilter string
		page         int
		perPage      int
		mockSetup    func()
		wantError    bool
		expectedErr  string
	}{
		{
			name:         "successful retrieval",
			statusFilter: "published",
			page:         1,
			perPage:      10,
			mockSetup: func() {
				mockRepo.On("GetGalleries", ctx, "published", 1, 10).
					Return(galleries, total, nil).Once()
			},
			wantError: false,
		},
		{
			name:         "repository error",
			statusFilter: "published",
			page:         1,
			perPage:      10,
			mockSetup: func() {
				mockRepo.On("GetGalleries", ctx, "published", 1, 10).
					Return(galleries, 0, errors.New("repository error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to get galleries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, totalCount, err := service.GetGalleries(ctx, tt.statusFilter, tt.page, tt.perPage)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
				assert.Equal(t, 0, totalCount)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, total, totalCount)
				assert.Equal(t, len(galleries), len(resp))
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
