package repository_test

import (
	"context"
	"fmt"
	"premium_caste/internal/domain/models"
	"premium_caste/internal/repository"
	redisapp "premium_caste/internal/storage/redis"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testCtx = context.Background()
)

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

func TestMediaRepo_CreateMedia(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewMediaRepository(db)

	tests := []struct {
		name    string
		media   *models.Media
		wantErr bool
	}{
		{
			name: "successful creation",
			media: &models.Media{
				ID:               uuid.New(),
				UploaderID:       uuid.New(),
				CreatedAt:        time.Now().UTC(),
				MediaType:        "image",
				OriginalFilename: "test.jpg",
				StoragePath:      "uploads/test.jpg",
				FileSize:         1024,
				MimeType:         "image/jpeg",
				IsPublic:         true,
				Metadata:         models.Metadata{"author": "test"},
			},
		},
		{
			name: "duplicate id",
			media: &models.Media{
				ID: uuid.Nil, // Будет заменено на существующий UUID
			},
			wantErr: true,
		},
	}

	// Сначала создаем медиа для теста на дубликат
	existingMedia := &models.Media{
		ID:               uuid.New(),
		UploaderID:       uuid.New(),
		MediaType:        "image",
		OriginalFilename: "existing.jpg",
		StoragePath:      "uploads/existing.jpg",
		FileSize:         2048,
	}
	_, err := repo.CreateMedia(testCtx, existingMedia)
	require.NoError(t, err)
	tests[1].media.ID = existingMedia.ID

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.CreateMedia(testCtx, tt.media)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.media.ID, got.ID)

			// Проверяем, что данные действительно записались в БД
			var count int
			err = db.QueryRow(testCtx,
				"SELECT COUNT(*) FROM media WHERE id = $1",
				tt.media.ID).Scan(&count)
			require.NoError(t, err)
			require.Equal(t, 1, count)
		})
	}
}

func TestMediaRepo_GroupOperations(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewMediaRepository(db)

	// Создаем тестовые данные
	ownerID := uuid.New()
	media1 := mustCreateMedia(t, repo, &models.Media{
		OriginalFilename: "media1.jpg",
		UploaderID:       ownerID,
	})
	media2 := mustCreateMedia(t, repo, &models.Media{
		OriginalFilename: "media2.jpg",
		UploaderID:       ownerID,
	})

	t.Run("create media group", func(t *testing.T) {
		err := repo.AddMediaGroup(testCtx, ownerID, "test group")
		require.NoError(t, err)

		var count int
		err = db.QueryRow(testCtx,
			"SELECT COUNT(*) FROM media_groups WHERE owner_id = $1",
			ownerID).Scan(&count)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("add media to group", func(t *testing.T) {
		// Создаем группу
		groupID := mustCreateGroup(t, db, ownerID)

		// Добавляем медиа в группу
		err := repo.AddMediaGroupItems(testCtx, groupID, media1.ID)
		require.NoError(t, err)

		// Проверяем связь в БД
		var position int
		err = db.QueryRow(testCtx, `
			SELECT position FROM media_group_items 
			WHERE group_id = $1 AND media_id = $2`,
			groupID, media1.ID).Scan(&position)
		require.NoError(t, err)
		require.Equal(t, 1, position)

		// Добавляем второе медиа и проверяем позицию
		err = repo.AddMediaGroupItems(testCtx, groupID, media2.ID)
		require.NoError(t, err)

		err = db.QueryRow(testCtx, `
			SELECT position FROM media_group_items 
			WHERE group_id = $1 AND media_id = $2`,
			groupID, media2.ID).Scan(&position)
		require.NoError(t, err)
		require.Equal(t, 2, position)
	})

	t.Run("get media by group id", func(t *testing.T) {
		groupID := mustCreateGroup(t, db, ownerID)
		mustAddToGroup(t, repo, groupID, media1.ID)
		mustAddToGroup(t, repo, groupID, media2.ID)

		mediaList, err := repo.GetMediaByGroupID(testCtx, groupID)
		require.NoError(t, err)
		require.Len(t, mediaList, 2)
		require.Equal(t, media1.ID, mediaList[0].ID)
		require.Equal(t, media2.ID, mediaList[1].ID)
	})
}

// Вспомогательные функции
func mustCreateMedia(t *testing.T, repo *repository.MediaRepo, m *models.Media) *models.Media {
	m.ID = uuid.New()
	m.CreatedAt = time.Now().UTC()
	if m.MediaType == "" {
		m.MediaType = "image"
	}
	if m.StoragePath == "" {
		m.StoragePath = "uploads/" + m.ID.String()
	}

	created, err := repo.CreateMedia(testCtx, m)
	require.NoError(t, err)
	return created
}

func mustCreateGroup(t *testing.T, db *pgxpool.Pool, ownerID uuid.UUID) uuid.UUID {
	var groupID uuid.UUID
	err := db.QueryRow(testCtx, `
		INSERT INTO media_groups (owner_id, description)
		VALUES ($1, 'test group')
		RETURNING id`,
		ownerID).Scan(&groupID)
	require.NoError(t, err)
	return groupID
}

func mustAddToGroup(t *testing.T, repo *repository.MediaRepo, groupID, mediaID uuid.UUID) {
	err := repo.AddMediaGroupItems(testCtx, groupID, mediaID)
	require.NoError(t, err)
}

func TestMediaRepo_UpdateAndFind(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewMediaRepository(db)

	// Создаем тестовое медиа
	media := mustCreateMedia(t, repo, &models.Media{
		OriginalFilename: "original.jpg",
		IsPublic:         false,
		Metadata:         models.Metadata{"old": "data"},
	})

	t.Run("update media", func(t *testing.T) {
		update := &models.Media{
			ID:               media.ID,
			OriginalFilename: "updated.jpg",
			IsPublic:         true,
			Metadata:         models.Metadata{"new": "data"},
		}

		err := repo.UpdateMedia(testCtx, update)
		require.NoError(t, err)

		// Проверяем обновленные данные
		var (
			filename string
			isPublic bool
			metadata map[string]interface{}
		)
		err = db.QueryRow(testCtx, `
			SELECT original_filename, is_public, metadata 
			FROM media WHERE id = $1`,
			media.ID).Scan(&filename, &isPublic, &metadata)
		require.NoError(t, err)
		require.Equal(t, "updated.jpg", filename)
		require.True(t, isPublic)
		require.Equal(t, "data", metadata["new"])
	})

	t.Run("find by id", func(t *testing.T) {
		found, err := repo.FindByID(testCtx, media.ID)
		require.NoError(t, err)
		require.Equal(t, media.ID, found.ID)
		require.Equal(t, "updated.jpg", found.OriginalFilename)
	})

	t.Run("find non-existent", func(t *testing.T) {
		_, err := repo.FindByID(testCtx, uuid.New())
		require.Error(t, err)
	})
}

func TestMediaRepo_TransactionHandling(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewMediaRepository(db)

	groupID := mustCreateGroup(t, db, uuid.New())
	mediaID := uuid.New() // Несуществующий медиа-файл

	t.Run("transaction rollback on error", func(t *testing.T) {
		err := repo.AddMediaGroupItems(testCtx, groupID, mediaID)
		require.Error(t, err)

		// Проверяем, что в группе нет элементов
		var count int
		err = db.QueryRow(testCtx, `
			SELECT COUNT(*) FROM media_group_items 
			WHERE group_id = $1`,
			groupID).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})
}

func TestUserRepository_SaveUser(t *testing.T) {
	pool := setupTestDB(t)

	repo := repository.NewUserRepository(pool)

	t.Run("successful user creation", func(t *testing.T) {
		user := models.User{
			Name:     "Test User",
			Email:    "test@example.com",
			Phone:    "+1234567890",
			Password: []byte("securepassword"),
			IsAdmin:  false,
			BasketID: uuid.New(),
		}

		id, err := repo.SaveUser(testCtx, user)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, id)

		var count int
		err = pool.QueryRow(testCtx, "SELECT COUNT(*) FROM users WHERE email = $1", user.Email).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("duplicate email", func(t *testing.T) {
		user := models.User{
			Name:     "Duplicate User",
			Email:    "duplicate@example.com",
			Password: []byte("password"),
		}

		_, err := repo.SaveUser(testCtx, user)
		require.NoError(t, err)

		_, err = repo.SaveUser(testCtx, user)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
	})
}

func TestUserRepository_User(t *testing.T) {
	pool := setupTestDB(t)

	repo := repository.NewUserRepository(pool)

	testUser := models.User{
		ID:       uuid.New(),
		Name:     "Existing User",
		Email:    "existing@example.com",
		Password: []byte("hashedpassword"),
		IsAdmin:  false,
		BasketID: uuid.New(),
	}

	_, err := pool.Exec(testCtx,
		"INSERT INTO users (id, name, email, password, is_admin, basket_id) VALUES ($1, $2, $3, $4, $5, $6)",
		testUser.ID, testUser.Name, testUser.Email, testUser.Password, testUser.IsAdmin, testUser.BasketID,
	)
	require.NoError(t, err)

	t.Run("existing user", func(t *testing.T) {
		user, err := repo.User(testCtx, testUser.Email)
		require.NoError(t, err)

		assert.Equal(t, testUser.ID, user.ID)
		assert.Equal(t, testUser.Name, user.Name)
		assert.Equal(t, testUser.Email, user.Email)
		assert.Equal(t, testUser.Password, user.Password)
		assert.Equal(t, testUser.IsAdmin, user.IsAdmin)
		assert.Equal(t, testUser.BasketID, user.BasketID)
	})

	// t.Run("non-existent user", func(t *testing.T) {
	// 	_, err := repo.User(testCtx, "nonexistent@example.com")
	// 	require.Error(t, err)
	// 	assert.ErrorIs(t, err, storage.ErrUserNotFound)
	// })

	// t.Run("empty email", func(t *testing.T) {
	// 	_, err := repo.User(testCtx, "")
	// 	require.Error(t, err)
	// 	assert.ErrorIs(t, err, storage.ErrUserNotFound)
	// })
}

func TestUserRepository_IsAdmin(t *testing.T) {
	pool := setupTestDB(t)

	repo := repository.NewUserRepository(pool)

	testUser := []models.User{
		{
			ID:       uuid.New(),
			Name:     "IsAdmin User",
			Email:    "admin@example.com",
			Password: []byte("hashedpassword"),
			IsAdmin:  true,
			BasketID: uuid.New(),
		},
		{
			ID:       uuid.New(),
			Name:     "Not admin User",
			Email:    "not_admin@example.com",
			Password: []byte("hashedpassword"),
			IsAdmin:  false,
			BasketID: uuid.New(),
		},
	}

	_, err := pool.Exec(testCtx,
		"INSERT INTO users (id, name, email, password, is_admin, basket_id) VALUES ($1, $2, $3, $4, $5, $6), ($7, $8, $9, $10, $11, $12)",
		testUser[0].ID, testUser[0].Name, testUser[0].Email, testUser[0].Password, testUser[0].IsAdmin, testUser[0].BasketID,
		testUser[1].ID, testUser[1].Name, testUser[1].Email, testUser[1].Password, testUser[1].IsAdmin, testUser[1].BasketID,
	)
	require.NoError(t, err)

	t.Run("user is admin", func(t *testing.T) {
		isAdmin, err := repo.IsAdmin(testCtx, testUser[0].ID)
		require.NoError(t, err)

		assert.Equal(t, testUser[0].IsAdmin, isAdmin)
	})

	t.Run("user is not admin", func(t *testing.T) {
		isAdmin, err := repo.IsAdmin(testCtx, testUser[1].ID)
		require.NoError(t, err)

		assert.Equal(t, testUser[1].IsAdmin, isAdmin)
	})
}

func NewMockClient() (*redisapp.Client, redismock.ClientMock) {
	db, mock := redismock.NewClientMock()
	return &redisapp.Client{Client: db}, mock
}

func setupRepo() (*repository.RedisTokenRepo, redismock.ClientMock) {
	db, mock := NewMockClient()
	return &repository.RedisTokenRepo{Client: db}, mock
}

func TestSaveRefreshToken(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupRepo()
	userID := uuid.New()
	token := "test_token"
	exp := 24 * time.Hour

	t.Run("successful save", func(t *testing.T) {
		mock.ExpectSet(refreshTokenKey(userID.String(), token), "1", exp).SetVal("OK")
		err := repo.SaveRefreshToken(ctx, userID.String(), token, exp)
		assert.NoError(t, err)
	})

	t.Run("redis error", func(t *testing.T) {
		mock.ExpectSet(refreshTokenKey(userID.String(), token), "1", exp).SetErr(redis.ErrClosed)
		err := repo.SaveRefreshToken(ctx, userID.String(), token, exp)
		assert.ErrorIs(t, err, redis.ErrClosed)
	})
}

func TestGetRefreshToken(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupRepo()
	userID := "user123"
	token := "test_token"

	t.Run("token exists", func(t *testing.T) {
		mock.ExpectGet(refreshTokenKey(userID, token)).SetVal("1")
		exists, err := repo.GetRefreshToken(ctx, userID, token)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("token not exists", func(t *testing.T) {
		mock.ExpectGet(refreshTokenKey(userID, token)).RedisNil()
		exists, err := repo.GetRefreshToken(ctx, userID, token)
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("redis error", func(t *testing.T) {
		mock.ExpectGet(refreshTokenKey(userID, token)).SetErr(redis.ErrClosed)
		_, err := repo.GetRefreshToken(ctx, userID, token)
		assert.ErrorIs(t, err, redis.ErrClosed)
	})
}

func TestDeleteRefreshToken(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupRepo()
	userID := "user123"
	token := "test_token"

	t.Run("successful delete", func(t *testing.T) {
		mock.ExpectDel(refreshTokenKey(userID, token)).SetVal(1)
		err := repo.DeleteRefreshToken(ctx, userID, token)
		assert.NoError(t, err)
	})

	t.Run("redis error", func(t *testing.T) {
		mock.ExpectDel(refreshTokenKey(userID, token)).SetErr(redis.ErrClosed)
		err := repo.DeleteRefreshToken(ctx, userID, token)
		assert.ErrorIs(t, err, redis.ErrClosed)
	})
}

func TestDeleteAllUserTokens(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupRepo()
	userID := "user123"

	t.Run("successful delete all", func(t *testing.T) {
		pattern := refreshTokenKey(userID, "*")
		mock.ExpectKeys(pattern).SetVal([]string{"token1", "token2"})
		mock.ExpectDel("token1", "token2").SetVal(2)
		err := repo.DeleteAllUserTokens(ctx, userID)
		assert.NoError(t, err)
	})

	t.Run("keys error", func(t *testing.T) {
		pattern := refreshTokenKey(userID, "*")
		mock.ExpectKeys(pattern).SetErr(redis.ErrClosed)
		err := repo.DeleteAllUserTokens(ctx, userID)
		assert.ErrorIs(t, err, redis.ErrClosed)
	})

	t.Run("del error", func(t *testing.T) {
		pattern := refreshTokenKey(userID, "*")
		mock.ExpectKeys(pattern).SetVal([]string{"token1"})
		mock.ExpectDel("token1").SetErr(redis.ErrClosed)
		err := repo.DeleteAllUserTokens(ctx, userID)
		assert.ErrorIs(t, err, redis.ErrClosed)
	})
}

func refreshTokenKey(userID, token string) string {
	return "refresh:" + userID + ":" + token
}
