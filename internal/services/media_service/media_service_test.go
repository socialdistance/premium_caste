package services_test

import (
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"premium_caste/internal/domain/models"
	services "premium_caste/internal/services/media_service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockMediaRepository struct {
	mock.Mock
}

func (m *MockMediaRepository) AddMediaGroup(ctx context.Context, ownerID uuid.UUID, description string) error {
	args := m.Called(ctx, ownerID, description)
	return args.Error(0)
}

func (m *MockMediaRepository) AddMediaGroupItems(ctx context.Context, groupID, mediaID uuid.UUID) error {
	args := m.Called(ctx, groupID, mediaID)
	return args.Error(0)
}

func (m *MockMediaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Media), args.Error(1)
}

func (m *MockMediaRepository) UpdateMedia(ctx context.Context, media *models.Media) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *MockMediaRepository) CreateMedia(ctx context.Context, media *models.Media) (*models.Media, error) {
	args := m.Called(ctx, media)
	return args.Get(0).(*models.Media), args.Error(1)
}

type MockFileStorage struct {
	mock.Mock
}

func (m *MockFileStorage) BaseURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockFileStorage) GetBaseDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockFileStorage) GetFullPath(relativePath string) string {
	args := m.Called(relativePath)
	return args.String(0)
}

func (m *MockFileStorage) Save(ctx context.Context, file *multipart.FileHeader, subPath string) (string, int64, error) {
	args := m.Called(ctx, file, subPath)
	return args.String(0), args.Get(1).(int64), args.Error(2)
}

func (m *MockFileStorage) Delete(ctx context.Context, filePath string) error {
	args := m.Called(ctx, filePath)

	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func TestFileStorageMethods(t *testing.T) {
	storageMock := new(MockFileStorage)

	// Настройка ожиданий
	storageMock.On("BaseURL").Return("https://storage.example.com")
	storageMock.On("GetBaseDir").Return("/data/storage")
	storageMock.On("GetFullPath", "images/1.jpg").Return("/data/storage/images/1.jpg")

	assert.Equal(t, "https://storage.example.com", storageMock.BaseURL())

	assert.Equal(t, "/data/storage", storageMock.GetBaseDir())

	assert.Equal(t, "/data/storage/images/1.jpg", storageMock.GetFullPath("images/1.jpg"))

	storageMock.AssertExpectations(t)
}

func TestStorageMethods(t *testing.T) {
	storageMock := new(MockFileStorage)
	ctx := context.Background()

	testPath := "test/path/file.txt"

	storageMock.On("BaseURL").Return("http://example.com")
	storageMock.On("GetBaseDir").Return("/storage")
	storageMock.On("GetFullPath", testPath).Return("/storage/full/path")
	storageMock.On("Save", ctx, mock.AnythingOfType("*multipart.FileHeader"), testPath).Return(testPath, int64(1024), nil).Once()
	storageMock.On("Delete", ctx, testPath).Return(errors.New("error delete"))

	t.Run("BaseURL", func(t *testing.T) {
		url := storageMock.BaseURL()
		assert.Equal(t, "http://example.com", url)
	})

	t.Run("GetBaseDir", func(t *testing.T) {
		dir := storageMock.GetBaseDir()
		assert.Equal(t, "/storage", dir)
	})

	t.Run("GetFullPath", func(t *testing.T) {
		fullPath := storageMock.GetFullPath(testPath)
		assert.Equal(t, "/storage/full/path", fullPath)
	})

	t.Run("Save", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "testfile")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte("test content"))
		require.NoError(t, err)
		tmpFile.Close()

		file, err := os.Open(tmpFile.Name())
		require.NoError(t, err)
		defer file.Close()

		fileHeader := &multipart.FileHeader{
			Filename: filepath.Base(tmpFile.Name()),
			Size:     1024,
		}

		path, size, err := storageMock.Save(ctx, fileHeader, testPath)
		assert.NoError(t, err)
		assert.Equal(t, testPath, path)
		assert.Equal(t, int64(1024), size)
	})
	t.Run("Delete", func(t *testing.T) {
		err := storageMock.Delete(ctx, testPath)
		assert.ErrorContains(t, err, "error delete")
	})

	storageMock.AssertExpectations(t)
}

func TestMediaRepositoryMethods(t *testing.T) {
	repoMock := new(MockMediaRepository)
	testMedia := &models.Media{ID: uuid.New()}
	groupID := uuid.New()
	mediaID := uuid.New()

	repoMock.On("AddMediaGroupItems", mock.Anything, groupID, mediaID).Return(nil)
	repoMock.On("FindByID", mock.Anything, testMedia.ID).Return(testMedia, nil)
	repoMock.On("UpdateMedia", mock.Anything, testMedia).Return(nil)

	assert.NoError(t, repoMock.AddMediaGroupItems(context.Background(), groupID, mediaID))

	found, err := repoMock.FindByID(context.Background(), testMedia.ID)
	assert.NoError(t, err)
	assert.Equal(t, testMedia, found)

	assert.NoError(t, repoMock.UpdateMedia(context.Background(), testMedia))

	repoMock.AssertExpectations(t)
}

func TestAttachMediaToGroup(t *testing.T) {
	mockRepo := new(MockMediaRepository)
	storageMock := new(MockFileStorage)

	log := slog.Default()

	service := services.NewMediaService(log, mockRepo, storageMock)

	validGroupID := uuid.New()
	validMediaID := uuid.New()

	t.Run("Succesfull add media", func(t *testing.T) {
		mockRepo.On("AddMediaGroupItems", mock.Anything, validGroupID, validMediaID).
			Return(nil)

		err := service.AttachMediaToGroup(context.Background(), validGroupID, validMediaID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation error, empty groupID", func(t *testing.T) {
		err := service.AttachMediaToGroup(context.Background(), uuid.Nil, validMediaID)

		assert.ErrorContains(t, err, "groupID is required")
		mockRepo.AssertNotCalled(t, "AddMediaToGroup")
	})

	t.Run("Validation error, empty mediaID", func(t *testing.T) {
		err := service.AttachMediaToGroup(context.Background(), validGroupID, uuid.Nil)

		assert.ErrorContains(t, err, "mediaID is required")
		mockRepo.AssertNotCalled(t, "AddMediaToGroup")
	})
}

func TestAttachMedia(t *testing.T) {
	mockRepo := new(MockMediaRepository)
	storageMock := new(MockFileStorage)

	log := slog.Default()

	service := services.NewMediaService(log, mockRepo, storageMock)

	validOwnerID := uuid.New()
	description := "cats"

	t.Run("Succesfull add media", func(t *testing.T) {
		mockRepo.On("AddMediaGroup", mock.Anything, validOwnerID, description).
			Return(nil)

		err := service.AttachMedia(context.Background(), validOwnerID, description)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation error, empty ownerID", func(t *testing.T) {
		err := service.AttachMedia(context.Background(), uuid.Nil, description)

		assert.ErrorContains(t, err, "ownerID is required")
		mockRepo.AssertNotCalled(t, "AddMediaGroup")
	})
}
