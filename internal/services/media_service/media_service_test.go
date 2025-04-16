package services_test

import (
	"bytes"
	"context"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"premium_caste/internal/domain/models"
	services "premium_caste/internal/services/media_service"
	"premium_caste/internal/transport/http/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockMediaRepository struct {
	mock.Mock
}

func (m *MockMediaRepository) AddMedia(ctx context.Context, groupID uuid.UUID, mediaID uuid.UUID) error {
	args := m.Called(ctx, groupID, mediaID)
	return args.Error(0)
}

func (m *MockMediaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Media), args.Error(0)
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

func (m *MockFileStorage) GetBaseDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockFileStorage) Save(ctx context.Context, file *multipart.FileHeader, subPath string) (string, int64, error) {
	args := m.Called(ctx, file, subPath)
	return args.String(0), args.Get(1).(int64), args.Error(2)
}

func (m *MockFileStorage) Delete(ctx context.Context, filePath string) error {
	args := m.Called(ctx, filePath)
	return args.Error(0)
}

func (m *MockFileStorage) GetFullPath(relativePath string) string {
	args := m.Called(relativePath)
	return args.String(0)
}

func (m *MockFileStorage) BaseURL() string {
	args := m.Called()
	return args.String(0)
}

func createTestFile(t *testing.T, filename, content string) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)

	_, err = part.Write([]byte(content))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	file, header, err := req.FormFile("file")
	require.NoError(t, err)
	file.Close()

	return header
}

func TestMediaService_UploadMedia(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockMediaRepository)
	mockStorage := new(MockFileStorage)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	service := services.NewMediaService(log, mockRepo, mockStorage)

	uploaderID := uuid.New()
	testFile := createTestFile(t, "test.jpg", "test content")

	validInput := dto.MediaUploadInput{
		File:           testFile,
		UploaderID:     uploaderID,
		MediaType:      "image",
		IsPublic:       true,
		CustomMetadata: map[string]interface{}{"key": "value"},
	}

	t.Run("successful upload", func(t *testing.T) {
		// Настройка ожидаемых вызовов
		expectedPath := filepath.Join("user_uploads", uploaderID.String(), "test.jpg")
		mockStorage.On("Save", ctx, testFile, filepath.Join("user_uploads", uploaderID.String())).
			Return(expectedPath, int64(11), nil).Once()

		expectedMedia := &models.Media{
			ID:          uuid.New(),
			StoragePath: expectedPath,
		}
		mockRepo.On("CreateMedia", ctx, mock.AnythingOfType("*models.Media")).
			Return(expectedMedia, nil).Once()

		// Вызов метода
		result, err := service.UploadMedia(ctx, validInput)

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, expectedMedia, result)
		mockStorage.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	// t.Run("media validation failure", func(t *testing.T) {
	// 	invalidInput := validInput
	// 	invalidInput.MediaType = "invalid_type"

	// 	expectedPath := filepath.Join("user_uploads", uploaderID.String(), "test.jpg")

	// 	// Настройка ожидаемых вызовов
	// 	mockStorage.On("Save", ctx, testFile, filepath.Join("user_uploads", uploaderID.String())).
	// 		Return(expectedPath, int64(11), nil).Once()
	// 	mockStorage.On("Delete", ctx, expectedPath). // Добавляем ожидание вызова Delete
	// 							Return(nil).Once()

	// 	_, err := service.UploadMedia(ctx, invalidInput)
	// 	assert.ErrorContains(t, err, "validation failed")
	// 	mockStorage.AssertExpectations(t)
	// })

	// t.Run("database failure", func(t *testing.T) {
	// 	expectedPath := filepath.Join("user_uploads", uploaderID.String(), "test.jpg")

	// 	// Настройка ожидаемых вызовов
	// 	mockStorage.On("Save", ctx, testFile, filepath.Join("user_uploads", uploaderID.String())).
	// 		Return(expectedPath, int64(11), nil).Once()
	// 	mockStorage.On("Delete", ctx, expectedPath). // Добавляем ожидание вызова Delete
	// 							Return(nil).Once()
	// 	mockRepo.On("CreateMedia", ctx, mock.AnythingOfType("*models.Media")).
	// 		Return(&models.Media{}, errors.New("db error")).Once()

	// 	_, err := service.UploadMedia(ctx, validInput)
	// 	assert.ErrorContains(t, err, "db error")
	// 	mockStorage.AssertExpectations(t)
	// 	mockRepo.AssertExpectations(t)
	// })
}

// func TestMediaService_UploadMedia_Integration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping integration test")
// 	}

// 	ctx := context.Background()
// 	mockRepo := new(MockMediaRepository)
// 	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

// 	// Создаем временную директорию для тестов
// 	tempDir := t.TempDir()
// 	fileStorage, err := storage.NewLocalFileStorage(tempDir, "http://localhost")
// 	require.NoError(t, err)

// 	service := services.NewMediaService(log, mockRepo, fileStorage)

// 	uploaderID := uuid.New()
// 	testFile := createTestFile(t, "test.jpg", "test content")

// 	validInput := dto.MediaUploadInput{
// 		File:       testFile,
// 		UploaderID: uploaderID,
// 		MediaType:  "image",
// 		IsPublic:   true,
// 	}

// 	t.Run("successful upload with real storage", func(t *testing.T) {
// 		expectedMedia := &models.Media{
// 			ID:          uuid.New(),
// 			StoragePath: filepath.Join("user_uploads", uploaderID.String(), "test.jpg"),
// 		}
// 		mockRepo.On("CreateMedia", ctx, mock.AnythingOfType("*models.Media")).
// 			Return(expectedMedia, nil).Once()

// 		result, err := service.UploadMedia(ctx, validInput)
// 		require.NoError(t, err)

// 		// Проверяем что файл действительно создан
// 		fullPath := fileStorage.GetFullPath(result.StoragePath)
// 		_, err = os.Stat(fullPath)
// 		assert.NoError(t, err)
// 	})
// }
