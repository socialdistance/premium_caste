package http_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"premium_caste/internal/lib/logger/sl"
	"premium_caste/internal/repository"
	storage "premium_caste/internal/storage/filestorage"
	httpapp "premium_caste/internal/transport/http"
	"strings"
	"testing"
	"time"

	media "premium_caste/internal/services/media_service"
	user "premium_caste/internal/services/user_service"

	"log/slog"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testCtx = context.Background()
)

type IntegrationTestSuite struct {
	suite.Suite
	db       *pgxpool.Pool
	echo     *echo.Echo
	server   *httptest.Server
	baseURL  string
	baseDir  string
	tokenTTL time.Duration
	repo     repository.Repository
}

func (s *IntegrationTestSuite) SetupSuite() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	db := setupTestDB(s.T())
	s.db = db

	e := echo.New()
	s.echo = e

	fileStorage, err := storage.NewLocalFileStorage("./uploads", s.baseURL)
	require.NoError(s.T(), err, "Failed to initialize file storage")

	userSerivce := user.NewUserService(log, s.repo.User, s.tokenTTL)
	mediaService := media.NewMediaService(log, s.repo.Media, fileStorage)

	router := httpapp.NewRouter(log, userSerivce, mediaService)

	s.server = httptest.NewServer(router)
	s.baseURL = s.server.URL

	resp, err := http.Get("http://127.0.0.1:33083/health")
	if err != nil {
		log.Error("Server not responding:", sl.Err(err))
	}
	defer resp.Body.Close()
	fmt.Println("Status:", resp.Status)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.server.Close()
	// require.NoError(s.T(), s.db.Close())
}

func (s *IntegrationTestSuite) SetupTest() {
	// Начало транзакции для каждого теста
	tx, err := s.db.Begin(testCtx)
	require.NoError(s.T(), err)

	// Можно сохранить транзакцию в контексте теста
	s.T().Cleanup(func() {
		// Откатываем транзакцию после теста
		require.NoError(s.T(), tx.Rollback(testCtx))
	})
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	connStr := fmt.Sprintf(
		"postgres://test:test@localhost:%s/testdb?sslmode=disable",
		port.Port(),
	)

	// Даем время на инициализацию БД
	time.Sleep(2 * time.Second)

	pool, err := pgxpool.Connect(ctx, connStr)
	require.NoError(t, err)

	// Применяем миграции
	err = applyMigrations(pool)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		pgContainer.Terminate(ctx)
	})

	return pool
}

func applyMigrations(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS media (
			id UUID PRIMARY KEY,
			uploader_id UUID NOT NULL,
			created_at TIMESTAMP NOT NULL,
			media_type TEXT NOT NULL,
			original_filename TEXT NOT NULL,
			storage_path TEXT NOT NULL,
			file_size BIGINT NOT NULL,
			mime_type TEXT,
			width INT,
			height INT,
			duration INT,
			is_public BOOLEAN NOT NULL DEFAULT false,
			metadata JSONB
		);
		
		CREATE TABLE IF NOT EXISTS media_groups (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			owner_id UUID NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		
		CREATE TABLE IF NOT EXISTS media_group_items (
			group_id UUID REFERENCES media_groups(id),
			media_id UUID REFERENCES media(id),
			position INT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (group_id, media_id)
		);

		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			phone TEXT,
			password TEXT NOT NULL,
			is_admin BOOLEAN,
			basket_id UUID,
			last_login TIMESTAMP WITH TIME ZONE
		);
	`)
	return err
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) TestUserRegistration() {
	// Подготовка тестовых данных
	ts := httptest.NewServer(s.echo)
	defer ts.Close()

	// 2. Формирование запроса
	body := `{"email":"test@example.com","password":"secret"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 3. Отправка с таймаутом
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// 4. Проверка ответа
	data, _ := io.ReadAll(resp.Body)
	fmt.Println(data)
}

// func (s *IntegrationTestSuite) TestMediaUpload() {
// 	// Создаем тестового пользователя
// 	userID := s.createTestUser()

// 	// Подготовка multipart запроса
// 	body := new(bytes.Buffer)
// 	writer := multipart.NewWriter(body)

// 	// Добавляем файл
// 	part, err := writer.CreateFormFile("file", "test.jpg")
// 	s.Require().NoError(err)
// 	_, err = part.Write([]byte("test content"))
// 	s.Require().NoError(err)

// 	// Добавляем остальные поля
// 	writer.WriteField("uploader_id", userID)
// 	writer.WriteField("media_type", "image")
// 	writer.Close()

// 	// Создаем запрос
// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		s.baseURL+"/api/v1/media/upload",
// 		body,
// 	)
// 	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())

// 	// Выполняем запрос
// 	rec := httptest.NewRecorder()
// 	s.echo.ServeHTTP(rec, req)

// 	// Проверки
// 	s.Equal(http.StatusCreated, rec.Code)

// 	var media map[string]interface{}
// 	s.NoError(json.Unmarshal(rec.Body.Bytes(), &media))
// 	s.NotEmpty(media["id"])
// }

// func (s *IntegrationTestSuite) createTestUser() string {
// 	// Реализация создания тестового пользователя в БД
// 	// Возвращает ID созданного пользователя
// 	return "user-uuid"
// }
