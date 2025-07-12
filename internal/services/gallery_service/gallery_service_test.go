package services

import (
	"context"
	"errors"
	"log/slog"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/transport/http/dto"
	"strings"
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

func (m *MockGalleryRepository) GetGalleriesByTags(ctx context.Context, tags []string, matchAll bool) ([]models.Gallery, error) {
	args := m.Called(ctx, tags, matchAll)
	return args.Get(0).([]models.Gallery), args.Error(1)
}

func (m *MockGalleryRepository) AddTags(ctx context.Context, galleryID string, tags []string) error {
	args := m.Called(ctx, galleryID, tags)
	return args.Error(0)
}

func (m *MockGalleryRepository) RemoveTags(ctx context.Context, galleryID string, tagsToRemove []string) error {
	args := m.Called(ctx, galleryID, tagsToRemove)
	return args.Error(0)
}

func (m *MockGalleryRepository) UpdateTags(ctx context.Context, galleryID string, tags []string) error {
	args := m.Called(ctx, galleryID, tags)
	return args.Error(0)
}

func (m *MockGalleryRepository) HasTags(ctx context.Context, galleryID string, tags []string) (bool, error) {
	args := m.Called(ctx, galleryID, tags)
	return args.Bool(0), args.Error(1)
}

func (m *MockGalleryRepository) GetTags(ctx context.Context, galleryID string) ([]string, error) {
	args := m.Called(ctx, galleryID)
	return args.Get(0).([]string), args.Error(1)
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

func TestTagService_GetGalleriesByTags(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	gallery1 := models.Gallery{
		ID:    uuid.New(),
		Title: "Gallery with tags",
		Tags:  []string{"nature", "landscape"},
	}

	gallery2 := models.Gallery{
		ID:    uuid.New(),
		Title: "Gallery with single tag",
		Tags:  []string{"art"},
	}

	tests := []struct {
		name      string
		tags      []string
		matchAll  bool
		mockSetup func()
		wantError bool
		expected  []dto.GalleryResponse
		errMsg    string
	}{
		{
			name:     "successful retrieval with AND logic",
			tags:     []string{"nature", "landscape"},
			matchAll: true,
			mockSetup: func() {
				mockRepo.On("GetGalleriesByTags", ctx, []string{"nature", "landscape"}, true).
					Return([]models.Gallery{gallery1}, nil).Once()
			},
			wantError: false,
			expected:  []dto.GalleryResponse{*service.mapToGalleryResponse(gallery1)},
		},
		{
			name:     "successful retrieval with OR logic",
			tags:     []string{"nature", "art"},
			matchAll: false,
			mockSetup: func() {
				mockRepo.On("GetGalleriesByTags", ctx, []string{"nature", "art"}, false).
					Return([]models.Gallery{gallery1, gallery2}, nil).Once()
			},
			wantError: false,
			expected:  []dto.GalleryResponse{*service.mapToGalleryResponse(gallery1), *service.mapToGalleryResponse(gallery2)},
		},
		{
			name:     "repository error",
			tags:     []string{"nature", "landscape"},
			matchAll: true,
			mockSetup: func() {
				mockRepo.On("GetGalleriesByTags", ctx, []string{"nature", "landscape"}, true).
					Return([]models.Gallery{}, errors.New("repository error")).Once()
			},
			wantError: true,
			errMsg:    "failed to get galleries",
		},
		{
			name:     "no matching tags",
			tags:     []string{"unknown"},
			matchAll: true,
			mockSetup: func() {
				mockRepo.On("GetGalleriesByTags", ctx, []string{"unknown"}, true).
					Return([]models.Gallery{}, nil).Once()
			},
			wantError: false,
			expected:  []dto.GalleryResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.GetGalleriesByTags(ctx, tt.tags, tt.matchAll)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, resp)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_AddTags(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	testUUID := uuid.New().String()
	validTags := []string{"art", "design"}
	invalidTags := []string{strings.Repeat("a", 51)}

	tests := []struct {
		name        string
		galleryID   string
		tags        []string
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:      "successful add tags",
			galleryID: testUUID,
			tags:      validTags,
			mockSetup: func() {
				mockRepo.On("AddTags", ctx, testUUID, validTags).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:        "invalid gallery id",
			galleryID:   "invalid-uuid",
			tags:        validTags,
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "некорректный идентификатор галереи",
		},
		{
			name:        "tag too long",
			galleryID:   testUUID,
			tags:        invalidTags,
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "тег не может быть длиннее 50 символов",
		},
		{
			name:      "repository error",
			galleryID: testUUID,
			tags:      validTags,
			mockSetup: func() {
				mockRepo.On("AddTags", ctx, testUUID, validTags).
					Return(errors.New("db error")).Once()
			},
			wantError:   true,
			expectedErr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := service.AddTags(ctx, tt.galleryID, tt.tags)

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

func TestService_RemoveTags(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	testUUID := uuid.New().String()
	tagsToRemove := []string{"old", "tag"}

	tests := []struct {
		name        string
		galleryID   string
		tags        []string
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:      "successful remove tags",
			galleryID: testUUID,
			tags:      tagsToRemove,
			mockSetup: func() {
				mockRepo.On("RemoveTags", ctx, testUUID, tagsToRemove).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:        "invalid gallery id",
			galleryID:   "invalid-uuid",
			tags:        tagsToRemove,
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "некорректный идентификатор галереи",
		},
		{
			name:      "empty tags",
			galleryID: testUUID,
			tags:      []string{},
			mockSetup: func() {},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := service.RemoveTags(ctx, tt.galleryID, tt.tags)

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

func TestService_ReplaceTags(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	testUUID := uuid.New().String()
	newTags := []string{"new", "tags"}

	tests := []struct {
		name        string
		galleryID   string
		tags        []string
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:      "successful replace tags",
			galleryID: testUUID,
			tags:      newTags,
			mockSetup: func() {
				mockRepo.On("UpdateTags", ctx, testUUID, newTags).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:        "invalid gallery id",
			galleryID:   "invalid-uuid",
			tags:        newTags,
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "некорректный идентификатор галереи",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := service.ReplaceTags(ctx, tt.galleryID, tt.tags)

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

func TestService_GetTags(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	testUUID := uuid.New().String()
	expectedTags := []string{"tag1", "tag2"}

	tests := []struct {
		name        string
		galleryID   string
		mockSetup   func()
		expected    []string
		wantError   bool
		expectedErr string
	}{
		{
			name:      "successful get tags",
			galleryID: testUUID,
			mockSetup: func() {
				mockRepo.On("GetTags", ctx, testUUID).
					Return(expectedTags, nil).Once()
			},
			expected:  expectedTags,
			wantError: false,
		},
		{
			name:        "invalid gallery id",
			galleryID:   "invalid-uuid",
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "некорректный идентификатор галереи",
		},
		{
			name:      "repository error",
			galleryID: testUUID,
			mockSetup: func() {
				mockRepo.On("GetTags", ctx, testUUID).
					Return([]string{}, errors.New("db error")).Once()
			},
			wantError:   true,
			expectedErr: "db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			tags, err := service.GetTags(ctx, tt.galleryID)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Empty(t, tags)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, tags)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_HasTags(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockGalleryRepository)
	service := NewGalleryService(slog.Default(), mockRepo)

	testUUID := uuid.New().String()
	tagsToCheck := []string{"required", "tag"}

	tests := []struct {
		name        string
		galleryID   string
		tags        []string
		mockSetup   func()
		expected    bool
		wantError   bool
		expectedErr string
	}{
		{
			name:      "has all tags",
			galleryID: testUUID,
			tags:      tagsToCheck,
			mockSetup: func() {
				mockRepo.On("HasTags", ctx, testUUID, tagsToCheck).
					Return(true, nil).Once()
			},
			expected:  true,
			wantError: false,
		},
		{
			name:        "invalid gallery id",
			galleryID:   "invalid-uuid",
			tags:        tagsToCheck,
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "некорректный идентификатор галереи",
		},
		{
			name:      "empty tags list",
			galleryID: testUUID,
			tags:      []string{},
			mockSetup: func() {},
			expected:  false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			hasTags, err := service.HasTags(ctx, tt.galleryID, tt.tags)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, hasTags)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
