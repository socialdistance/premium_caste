package storage

import (
	"context"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// FileStorage интерфейс для работы с файловым хранилищем
type FileStorage interface {
	Save(ctx context.Context, file *multipart.FileHeader, subPath string) (filePath string, fileSize int64, err error)
	Delete(ctx context.Context, filePath string) error
	GetFullPath(relativePath string) string
	BaseURL() string
}

// LocalFileStorage реализация для локальной файловой системы
type LocalFileStorage struct {
	baseDir string // Базовый каталог для хранения (например: "./uploads")
	baseURL string // Базовый URL для доступа к файлам (например: "http://localhost:8080/uploads")
}

func NewLocalFileStorage(baseDir, baseURL string) (*LocalFileStorage, error) {
	// Создаем директорию, если она не существует
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	return &LocalFileStorage{
		baseDir: baseDir,
		baseURL: baseURL,
	}, nil
}

func (s *LocalFileStorage) Save(ctx context.Context, file *multipart.FileHeader, subPath string) (string, int64, error) {
	// Создаем полный путь к файлу
	filePath := filepath.Join(s.baseDir, subPath, file.Filename)

	// Создаем все необходимые поддиректории
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", 0, err
	}

	// Открываем исходный файл
	src, err := file.Open()
	if err != nil {
		return "", 0, err
	}
	defer src.Close()

	// Создаем новый файл
	dst, err := os.Create(filePath)
	if err != nil {
		return "", 0, err
	}
	defer dst.Close()

	// Копируем содержимое
	size, err := io.Copy(dst, src)
	if err != nil {
		return "", 0, err
	}

	// Возвращаем относительный путь и размер
	relativePath := filepath.Join(subPath, file.Filename)
	return relativePath, size, nil
}

// Delete удаляет файл из хранилища
func (s *LocalFileStorage) Delete(ctx context.Context, filePath string) error {
	fullPath := filepath.Join(s.baseDir, filePath)
	return os.Remove(fullPath)
}

// GetFullPath возвращает полный путь к файлу на диске
func (s *LocalFileStorage) GetFullPath(relativePath string) string {
	return filepath.Join(s.baseDir, relativePath)
}

// BaseURL возвращает базовый URL для доступа к файлам
func (s *LocalFileStorage) BaseURL() string {
	return s.baseURL
}

// func (s *LocalFileStorage) validateFile(file *multipart.FileHeader) error {
// 	// Проверка размера файла
// 	if s.config.MaxSize > 0 && file.Size > s.config.MaxSize {
// 		return storage.ErrFileTooLarge
// 	}

// 	// Проверка MIME-типа
// 	if !isAllowedType(file.Header.Get("Content-Type")) {
// 		return storage.ErrInvalidFileType
// 	}

// 	return nil
// }

// func generateFilename(originalName string) string {
// 	ext := filepath.Ext(originalName)
// 	return uuid.New().String() + ext
// }

// func (s *LocalFileStorage) Save(ctx context.Context, file *multipart.FileHeader, subPath string) (string, int64, error) {
// 	logger := log.FromContext(ctx)
// 	logger.Info("Saving file", "filename", file.Filename)

// 	// ... остальная реализация
// }
