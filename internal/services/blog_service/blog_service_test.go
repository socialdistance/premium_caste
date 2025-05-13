package services

import (
	"context"
	"errors"
	"log/slog"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/transport/http/dto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBlogRepository реализация мок-репозитория
type MockBlogRepository struct {
	mock.Mock
}

func (m *MockBlogRepository) SaveBlogPost(ctx context.Context, post models.BlogPost) (uuid.UUID, error) {
	args := m.Called(ctx, post)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockBlogRepository) GetBlogPostByID(ctx context.Context, id uuid.UUID) (*models.BlogPost, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BlogPost), args.Error(1)
}

func (m *MockBlogRepository) UpdateBlogPostFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockBlogRepository) GetBlogPosts(ctx context.Context, statusFilter string, page, perPage int) ([]models.BlogPost, int, error) {
	args := m.Called(ctx, statusFilter, page, perPage)
	return args.Get(0).([]models.BlogPost), args.Int(1), args.Error(2)
}

func (m *MockBlogRepository) SoftDeleteBlogPost(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBlogRepository) DeleteBlogPost(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBlogRepository) AddMediaGroupToPost(ctx context.Context, postID, groupID uuid.UUID, relationType string) error {
	args := m.Called(ctx, postID, groupID, relationType)
	return args.Error(0)
}

func (m *MockBlogRepository) GetPostMediaGroups(ctx context.Context, postID uuid.UUID, relationType string) ([]uuid.UUID, error) {
	args := m.Called(ctx, postID, relationType)
	return args.Get(0).([]uuid.UUID), args.Error(0)
}

func TestBlogService_CreatePost(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	testUUID := uuid.MustParse("b3c87987-ba25-4c7b-8070-f74ef402fe7c")
	authorID := uuid.New()
	mockPost := &models.BlogPost{
		ID:        testUUID,
		Title:     "Test Post",
		Slug:      "test-post",
		AuthorID:  authorID,
		Content:   "This is test post content",
		Excerpt:   "Test post excerpt",
		Status:    "published",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		req         dto.CreateBlogPostRequest
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name: "successful creation with auto slug",
			req: dto.CreateBlogPostRequest{
				Title:    "Test Post",
				AuthorID: authorID,
			},
			mockSetup: func() {
				mockRepo.On("SaveBlogPost", ctx, mock.AnythingOfType("models.BlogPost")).
					Return(testUUID, nil).Once()

				mockRepo.On("GetBlogPostByID", ctx, testUUID).
					Return(mockPost, nil).
					Once()
			},
			wantError: false,
		},
		{
			name: "missing title",
			req: dto.CreateBlogPostRequest{
				AuthorID: uuid.New(),
			},
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "post title is required",
		},
		{
			name: "missing author ID",
			req: dto.CreateBlogPostRequest{
				Title: "Test Post",
			},
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "author ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.CreatePost(ctx, tt.req)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.req.Slug != "" {
					assert.True(t, strings.HasPrefix(resp.Slug, tt.req.Slug))
				} else {
					assert.NotEmpty(t, resp.Slug)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBlogService_UpdatePost(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	postID := uuid.New()
	existingPost := &models.BlogPost{
		ID:        postID,
		Title:     "Existing Post",
		Slug:      "existing-post",
		Status:    "draft",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	tests := []struct {
		name        string
		postID      uuid.UUID
		req         dto.UpdateBlogPostRequest
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:   "successful update title",
			postID: postID,
			req: dto.UpdateBlogPostRequest{
				Title: stringPtr("Updated Title"),
			},
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(existingPost, nil).Twice()
				mockRepo.On("UpdateBlogPostFields", ctx, postID, mock.Anything).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:   "auto slug generation when empty",
			postID: postID,
			req: dto.UpdateBlogPostRequest{
				Title: stringPtr("New Title"),
				Slug:  stringPtr(""),
			},
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(existingPost, nil).Twice()
				mockRepo.On("UpdateBlogPostFields", ctx, postID, mock.Anything).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:   "publish post with auto published_at",
			postID: postID,
			req: dto.UpdateBlogPostRequest{
				Status: stringPtr("published"),
			},
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(existingPost, nil).Twice()
				mockRepo.On("UpdateBlogPostFields", ctx, postID, mock.Anything).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:   "post not found",
			postID: postID,
			req:    dto.UpdateBlogPostRequest{},
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(nil, errors.New("Not found")).Once()
			},
			wantError:   true,
			expectedErr: "failed to get post",
		},
		{
			name:   "repository update error",
			postID: postID,
			req: dto.UpdateBlogPostRequest{
				Title: stringPtr("Updated Title"),
			},
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(existingPost, nil).Once()
				mockRepo.On("UpdateBlogPostFields", ctx, postID, mock.Anything).
					Return(errors.New("update error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to update post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.UpdatePost(ctx, tt.postID, tt.req)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBlogService_GetPostByID(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	postID := uuid.New()
	expectedPost := &models.BlogPost{
		ID:        postID,
		Title:     "Test Post",
		Slug:      "test-post",
		Status:    "published",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		postID      uuid.UUID
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:   "successful get",
			postID: postID,
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(expectedPost, nil).Once()
			},
			wantError: false,
		},
		{
			name:   "post not found",
			postID: postID,
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(nil, errors.New("Not found")).Once()
			},
			wantError:   true,
			expectedErr: "failed to get post",
		},
		{
			name:   "repository error",
			postID: postID,
			mockSetup: func() {
				mockRepo.On("GetBlogPostByID", ctx, postID).
					Return(nil, errors.New("db error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to get post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.GetPostByID(ctx, tt.postID)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.postID, resp.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBlogService_ListPosts(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	now := time.Now()
	posts := []models.BlogPost{
		{
			ID:        uuid.New(),
			Title:     "Post 1",
			Status:    "published",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Post 2",
			Status:    "published",
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
	}

	tests := []struct {
		name        string
		status      string
		page        int
		perPage     int
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:    "successful list",
			status:  "published",
			page:    1,
			perPage: 10,
			mockSetup: func() {
				mockRepo.On("GetBlogPosts", ctx, "published", 1, 10).
					Return(posts, 2, nil).Once()
			},
			wantError: false,
		},
		{
			name:    "invalid page correction",
			status:  "published",
			page:    0,
			perPage: 0,
			mockSetup: func() {
				mockRepo.On("GetBlogPosts", ctx, "published", 1, 10).
					Return(posts, 2, nil).Once()
			},
			wantError: false,
		},
		{
			name:    "repository error",
			status:  "published",
			page:    1,
			perPage: 10,
			mockSetup: func() {
				mockRepo.On("GetBlogPosts", ctx, "published", 1, 10).
					Return([]models.BlogPost{}, 0, errors.New("db error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to list posts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.ListPosts(ctx, tt.status, tt.page, tt.perPage)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Posts, len(posts))
				assert.Equal(t, 2, resp.TotalCount)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBlogService_PublishPost(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	testUUID := uuid.MustParse("b3c87987-ba25-4c7b-8070-f74ef402fe7c")
	authorID := uuid.New()
	mockPost := &models.BlogPost{
		ID:        testUUID,
		Title:     "Test Post",
		Slug:      "test-post",
		AuthorID:  authorID,
		Content:   "This is test post content",
		Excerpt:   "Test post excerpt",
		Status:    "published",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		postID      uuid.UUID
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:   "successful publish",
			postID: testUUID,
			mockSetup: func() {
				mockRepo.On("UpdateBlogPostFields", ctx, testUUID, mock.Anything).
					Return(nil).Once()

				mockRepo.On("GetBlogPostByID", ctx, testUUID).
					Return(mockPost, nil).
					Once()
			},
			wantError: false,
		},
		{
			name:   "repository error",
			postID: testUUID,
			mockSetup: func() {
				mockRepo.On("UpdateBlogPostFields", ctx, testUUID, mock.Anything).
					Return(errors.New("update error")).Once()

			},
			wantError:   true,
			expectedErr: "failed to publish post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.PublishPost(ctx, tt.postID)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBlogService_ArchivePost(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	testUUID := uuid.MustParse("b3c87987-ba25-4c7b-8070-f74ef402fe7c")
	authorID := uuid.New()
	mockPost := &models.BlogPost{
		ID:        testUUID,
		Title:     "Test Post",
		Slug:      "test-post",
		AuthorID:  authorID,
		Content:   "This is test post content",
		Excerpt:   "Test post excerpt",
		Status:    "published",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		postID      uuid.UUID
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:   "successful archive",
			postID: testUUID,
			mockSetup: func() {
				mockRepo.On("SoftDeleteBlogPost", ctx, testUUID).
					Return(nil).Once()

				mockRepo.On("GetBlogPostByID", ctx, testUUID).
					Return(mockPost, nil).
					Once()
			},
			wantError: false,
		},
		{
			name:   "repository error",
			postID: testUUID,
			mockSetup: func() {
				mockRepo.On("SoftDeleteBlogPost", ctx, testUUID).
					Return(errors.New("archive error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to archive post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := service.ArchivePost(ctx, tt.postID)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestBlogService_DeletePost(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	postID := uuid.New()

	tests := []struct {
		name        string
		postID      uuid.UUID
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:   "successful delete",
			postID: postID,
			mockSetup: func() {
				mockRepo.On("DeleteBlogPost", ctx, postID).
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:   "repository error",
			postID: postID,
			mockSetup: func() {
				mockRepo.On("DeleteBlogPost", ctx, postID).
					Return(errors.New("delete error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to delete post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := service.DeletePost(ctx, tt.postID)

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

func TestBlogService_AddMediaGroup(t *testing.T) {
	ctx := context.Background()
	log := slog.Default()
	mockRepo := new(MockBlogRepository)
	service := NewBlogService(log, mockRepo)

	postID := uuid.New()
	groupID := uuid.New()

	tests := []struct {
		name        string
		postID      uuid.UUID
		req         dto.AddMediaGroupRequest
		mockSetup   func()
		wantError   bool
		expectedErr string
	}{
		{
			name:   "successful add media group",
			postID: postID,
			req: dto.AddMediaGroupRequest{
				GroupID:      groupID,
				RelationType: "gallery",
			},
			mockSetup: func() {
				mockRepo.On("AddMediaGroupToPost", ctx, postID, groupID, "gallery").
					Return(nil).Once()
			},
			wantError: false,
		},
		{
			name:   "invalid relation type",
			postID: postID,
			req: dto.AddMediaGroupRequest{
				GroupID:      groupID,
				RelationType: "invalid",
			},
			mockSetup:   func() {},
			wantError:   true,
			expectedErr: "invalid relation type",
		},
		{
			name:   "repository error",
			postID: postID,
			req: dto.AddMediaGroupRequest{
				GroupID:      groupID,
				RelationType: "gallery",
			},
			mockSetup: func() {
				mockRepo.On("AddMediaGroupToPost", ctx, postID, groupID, "gallery").
					Return(errors.New("add error")).Once()
			},
			wantError:   true,
			expectedErr: "failed to add media group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			_, err := service.AddMediaGroup(ctx, tt.postID, tt.req)

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

func stringPtr(s string) *string {
	return &s
}
