package storage_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	storage "premium_caste/internal/storage/filestorage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFileStorage(t *testing.T) (*storage.LocalFileStorage, string) {
	t.Helper()

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "filestorage_test")
	require.NoError(t, err)

	// Создаем хранилище
	fs, err := storage.NewLocalFileStorage(tempDir, "http://test.local")
	require.NoError(t, err)

	return fs, tempDir
}

func cleanupFileStorage(t *testing.T, dir string) {
	t.Helper()
	_ = os.RemoveAll(dir)
}

func createTestFile(t *testing.T, dir, filename, content string) *multipart.FileHeader {
	t.Helper()

	// Создаем временный файл
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	// Создаем multipart форму
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)

	_, err = part.Write([]byte(content))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Парсим multipart запрос
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	file, header, err := req.FormFile("file")
	require.NoError(t, err)
	file.Close()

	return header
}

func TestLocalFileStorage_Save(t *testing.T) {
	fs, tempDir := setupFileStorage(t)
	defer cleanupFileStorage(t, tempDir)

	ctx := context.Background()
	testFile := createTestFile(t, tempDir, "test.txt", "test content")

	t.Run("successful save", func(t *testing.T) {
		// Создаем тестовый файл
		testFile := createTestFile(t, tempDir, "test.txt", "test content")

		filePath, size, err := fs.Save(ctx, testFile, "subdir")
		require.NoError(t, err)

		assert.Equal(t, filepath.Join("subdir", "test.txt"), filePath)
		assert.Equal(t, int64(12), size)

		// Проверяем что файл создан
		fullPath := fs.GetFullPath(filePath)
		_, err = os.Stat(fullPath)
		assert.NoError(t, err)

		// Проверяем содержимое файла
		data, err := os.ReadFile(fullPath)
		require.NoError(t, err)
		assert.Equal(t, "test content", string(data))
	})

	t.Run("save with empty subpath", func(t *testing.T) {
		filePath, _, err := fs.Save(ctx, testFile, "")
		require.NoError(t, err)
		assert.Equal(t, "test.txt", filePath)
	})

	t.Run("save with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		cancel() // Отменяем контекст сразу

		_, _, err := fs.Save(ctx, testFile, "subdir")
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestLocalFileStorage_Delete(t *testing.T) {
	fs, tempDir := setupFileStorage(t)
	defer cleanupFileStorage(t, tempDir)

	ctx := context.Background()
	testFile := createTestFile(t, tempDir, "to_delete.txt", "content")

	t.Run("successful delete", func(t *testing.T) {
		// Сначала сохраняем файл
		filePath, _, err := fs.Save(ctx, testFile, "")
		require.NoError(t, err)

		// Удаляем
		err = fs.Delete(ctx, filePath)
		assert.NoError(t, err)

		// Проверяем что файл удален
		_, err = os.Stat(fs.GetFullPath(filePath))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		err := fs.Delete(ctx, "nonexistent.txt")
		assert.Error(t, err)
	})
}

func TestLocalFileStorage_GetFullPath(t *testing.T) {
	fs, tempDir := setupFileStorage(t)
	defer os.RemoveAll(tempDir)

	t.Run("returns correct path", func(t *testing.T) {
		relPath := "test/file.txt"
		expected := filepath.Join(fs.GetBaseDir(), relPath)
		assert.Equal(t, expected, fs.GetFullPath(relPath))
	})
}

func TestLocalFileStorage_BaseURL(t *testing.T) {
	fs, _ := setupFileStorage(t)
	assert.Equal(t, "http://test.local", fs.BaseURL())
}

func TestNewLocalFileStorage(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		tempDir := t.TempDir() // Автоматически удалится после теста

		fs, err := storage.NewLocalFileStorage(tempDir, "http://test.local")
		require.NoError(t, err)
		assert.NotNil(t, fs)
	})

	t.Run("invalid directory", func(t *testing.T) {
		// Пытаемся создать в несуществующей корневой директории
		_, err := storage.NewLocalFileStorage("/nonexistent/path", "http://test.local")
		assert.Error(t, err)
	})
}

func TestSaveErrorCases(t *testing.T) {
	fs, tempDir := setupFileStorage(t)
	defer cleanupFileStorage(t, tempDir)

	ctx := context.Background()

	t.Run("invalid file header", func(t *testing.T) {
		invalidFile := &multipart.FileHeader{
			Filename: "bad.txt",
		}
		_, _, err := fs.Save(ctx, invalidFile, "")
		assert.Error(t, err)
	})

	t.Run("read-only directory", func(t *testing.T) {
		// Создаем read-only директорию
		roDir := filepath.Join(tempDir, "readonly")
		require.NoError(t, os.Mkdir(roDir, 0444))

		testFile := createTestFile(t, tempDir, "ro_test.txt", "test")
		_, _, err := fs.Save(ctx, testFile, "readonly/subdir")
		assert.Error(t, err)
	})
}

func TestConcurrentSaves(t *testing.T) {
	fs, tempDir := setupFileStorage(t)
	defer cleanupFileStorage(t, tempDir)

	ctx := context.Background()
	testFile := createTestFile(t, tempDir, "concurrent.txt", "data")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, err := fs.Save(ctx, testFile, "concurrent")
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
}
