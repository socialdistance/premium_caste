package services_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"premium_caste/internal/domain/models"
	services "premium_caste/internal/services/media_service"
	"premium_caste/internal/transport/http/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Мок репозитория
type MockMediaRepository struct {
	mock.Mock
}

func (m *MockMediaRepository) AddMedia(ctx context.Context, groupID, mediaID uuid.UUID) error {
	args := m.Called(ctx, groupID, mediaID)
	return args.Error(0)
}

// FindByID ищет медиа по ID
func (m *MockMediaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Media), args.Error(1)
}

// UpdateMedia обновляет данные медиа
func (m *MockMediaRepository) UpdateMedia(ctx context.Context, media *models.Media) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *MockMediaRepository) CreateMedia(ctx context.Context, media *models.Media) (*models.Media, error) {
	args := m.Called(ctx, media)
	return args.Get(0).(*models.Media), args.Error(1)
}

// Мок хранилища файлов
type MockFileStorage struct {
	mock.Mock
}

// BaseURL implements storage.FileStorage.
func (m *MockFileStorage) BaseURL() string {
	args := m.Called()
	return args.String(0)
}

// GetBaseDir implements storage.FileStorage.
func (m *MockFileStorage) GetBaseDir() string {
	args := m.Called()
	return args.String(0)
}

// GetFullPath implements storage.FileStorage.
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
	return args.Error(0)
}

// Вспомогательная функция для создания тестового файла
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
	log := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{}))
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
		// Настраиваем ожидания
		expectedPath := "user_uploads/" + uploaderID.String() + "/test.jpg"
		mockStorage.On("Save", ctx, testFile, "user_uploads/"+uploaderID.String()).
			Return(expectedPath, int64(11), nil).Once()

		expectedMedia := &models.Media{ID: uuid.New()}
		mockRepo.On("CreateMedia", ctx, mock.AnythingOfType("*models.Media")).
			Return(expectedMedia, nil).Once()

		// Вызываем метод
		result, err := service.UploadMedia(ctx, validInput)

		// Проверяем результаты
		require.NoError(t, err)
		assert.Equal(t, expectedMedia, result)
		mockStorage.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("file storage save failure", func(t *testing.T) {
		mockStorage.On("Save", ctx, testFile, mock.Anything).
			Return("", int64(0), errors.New("storage error")).Once()

		_, err := service.UploadMedia(ctx, validInput)
		assert.ErrorContains(t, err, "storage error")
		mockStorage.AssertExpectations(t)
	})

	t.Run("media validation failure", func(t *testing.T) {
		invalidInput := validInput
		invalidInput.MediaType = "invalid_type"

		expectedPath := "user_uploads/" + uploaderID.String() + "/test.jpg"
		mockStorage.On("Save", ctx, testFile, mock.Anything).
			Return(expectedPath, int64(11), nil).Once()
		mockStorage.On("Delete", ctx, expectedPath).
			Return(nil).Once()

		_, err := service.UploadMedia(ctx, invalidInput)
		assert.ErrorContains(t, err, "validation failed")
		mockStorage.AssertExpectations(t)
	})

	t.Run("database save failure", func(t *testing.T) {
		expectedPath := "user_uploads/" + uploaderID.String() + "/test.jpg"
		mockStorage.On("Save", ctx, testFile, mock.Anything).
			Return(expectedPath, int64(11), nil).Once()
		mockRepo.On("CreateMedia", ctx, mock.Anything).
			Return(nil, errors.New("db error")).Once()
		mockStorage.On("Delete", ctx, expectedPath).
			Return(nil).Once()

		_, err := service.UploadMedia(ctx, validInput)
		assert.ErrorContains(t, err, "db error")
		mockStorage.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("delete failure after validation error", func(t *testing.T) {
		invalidInput := validInput
		invalidInput.MediaType = "invalid_type"

		expectedPath := "user_uploads/" + uploaderID.String() + "/test.jpg"
		mockStorage.On("Save", ctx, testFile, mock.Anything).
			Return(expectedPath, int64(11), nil).Once()
		mockStorage.On("Delete", ctx, expectedPath).
			Return(errors.New("delete failed")).Once()

		_, err := service.UploadMedia(ctx, invalidInput)
		assert.ErrorContains(t, err, "validation failed")
		mockStorage.AssertExpectations(t)
	})
}

func TestFileStorageMethods(t *testing.T) {
	storageMock := new(MockFileStorage)

	// Настройка ожиданий
	storageMock.On("BaseURL").Return("https://storage.example.com")
	storageMock.On("GetBaseDir").Return("/data/storage")
	storageMock.On("GetFullPath", "images/1.jpg").Return("/data/storage/images/1.jpg")

	// Проверка BaseURL
	assert.Equal(t, "https://storage.example.com", storageMock.BaseURL())

	// Проверка GetBaseDir
	assert.Equal(t, "/data/storage", storageMock.GetBaseDir())

	// Проверка GetFullPath
	assert.Equal(t, "/data/storage/images/1.jpg", storageMock.GetFullPath("images/1.jpg"))

	// Верификация вызовов
	storageMock.AssertExpectations(t)
}

func TestMediaRepositoryMethods(t *testing.T) {
	repoMock := new(MockMediaRepository)
	testMedia := &models.Media{ID: uuid.New()}
	groupID := uuid.New()
	mediaID := uuid.New()

	// Настройка ожиданий
	repoMock.On("AddMedia", mock.Anything, groupID, mediaID).Return(nil)
	repoMock.On("FindByID", mock.Anything, testMedia.ID).Return(testMedia, nil)
	repoMock.On("UpdateMedia", mock.Anything, testMedia).Return(nil)

	// Проверка AddMedia
	assert.NoError(t, repoMock.AddMedia(context.Background(), groupID, mediaID))

	// Проверка FindByID
	found, err := repoMock.FindByID(context.Background(), testMedia.ID)
	assert.NoError(t, err)
	assert.Equal(t, testMedia, found)

	// Проверка UpdateMedia
	assert.NoError(t, repoMock.UpdateMedia(context.Background(), testMedia))

	// Верификация вызовов
	repoMock.AssertExpectations(t)
}
