package storage

import (
	"context"
	"fmt"
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
	GetBaseDir() string
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
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}

	filePath := filepath.Join(s.baseDir, subPath, file.Filename)

	select {
	case <-ctx.Done():
		return "", 0, ctx.Err()
	default:
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return "", 0, fmt.Errorf("failed to create directories: %w", err)
		}
	}

	src, err := file.Open()
	if err != nil {
		return "", 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Создаем целевой файл
	dst, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	done := make(chan struct{})
	var size int64
	var copyErr error

	go func() {
		size, copyErr = io.Copy(dst, src)
		close(done)
	}()

	select {
	case <-done:
		if copyErr != nil {
			_ = os.Remove(filePath)
			return "", 0, fmt.Errorf("failed to copy file: %w", copyErr)
		}
	case <-ctx.Done():
		_ = os.Remove(filePath)
		return "", 0, ctx.Err()
	}

	return filepath.Join(subPath, file.Filename), size, nil
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

func (s *LocalFileStorage) GetBaseDir() string {
	return s.baseDir
}
