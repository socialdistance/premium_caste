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
			registration_date TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
			last_login TIMESTAMP WITH TIME ZONE
		);

		CREATE TABLE IF NOT EXISTS blog_posts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,                  
			slug VARCHAR(255) UNIQUE NOT NULL,           
			excerpt TEXT,                               
			content TEXT NOT NULL,                       
			featured_image_id UUID,  
			author_id UUID NOT NULL,                     
			status VARCHAR(20) NOT NULL DEFAULT 'draft',  
			published_at TIMESTAMPTZ,                    
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			metadata JSONB                               
		);

		CREATE TABLE IF NOT EXISTS post_media_groups (
			post_id UUID NOT NULL,
			group_id UUID NOT NULL,
			relation_type VARCHAR(30) NOT NULL DEFAULT 'content', 
			PRIMARY KEY (post_id, group_id)
		);
		
		CREATE TABLE galleries (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			images TEXT[] NOT NULL DEFAULT '{}',
			cover_image_index INT DEFAULT 0,      
			author_id UUID NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'draft',
			published_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			metadata JSONB,
			tags VARCHAR(255)[] DEFAULT '{}'      
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
		_, err := repo.AddMediaGroup(testCtx, ownerID, "test group")
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
		err := repo.AddMediaGroupItems(testCtx, groupID, []uuid.UUID{media1.ID, media2.ID})
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
		err = repo.AddMediaGroupItems(testCtx, groupID, []uuid.UUID{media2.ID})
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
	err := repo.AddMediaGroupItems(testCtx, groupID, []uuid.UUID{mediaID})
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
		err := repo.AddMediaGroupItems(testCtx, groupID, []uuid.UUID{mediaID})
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
		ID:               uuid.New(),
		Name:             "Existing User",
		Email:            "existing@example.com",
		Phone:            "+123456789",
		Password:         []byte("hashedpassword"),
		IsAdmin:          false,
		BasketID:         uuid.New(),
		RegistrationDate: time.Now(),
		LastLogin:        time.Now(),
	}

	_, err := pool.Exec(testCtx,
		"INSERT INTO users (id, name, email, phone, password, is_admin, basket_id, registration_date, last_login) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		testUser.ID, testUser.Name, testUser.Email, testUser.Phone, testUser.Password, testUser.IsAdmin, testUser.BasketID, testUser.RegistrationDate, testUser.LastLogin)
	require.NoError(t, err)

	t.Run("existing user", func(t *testing.T) {
		user, err := repo.UserByIdentifier(testCtx, testUser.Email)
		require.NoError(t, err)

		assert.Equal(t, testUser.ID, user.ID)
		assert.Equal(t, testUser.Name, user.Name)
		assert.Equal(t, testUser.Email, user.Email)
		assert.Equal(t, testUser.Password, user.Password)
		assert.Equal(t, testUser.IsAdmin, user.IsAdmin)
		assert.Equal(t, testUser.BasketID, user.BasketID)
	})

	t.Run("get user by id", func(t *testing.T) {
		user, err := repo.GetUserById(testCtx, testUser.ID)
		require.NoError(t, err)

		assert.Equal(t, testUser.Name, user.Name)
		assert.Equal(t, testUser.Email, user.Email)
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

func TestSaveBlogPost(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := repository.NewBlogRepository(pool)

	post := models.BlogPost{
		Title:           "Test Post",
		Slug:            "test-post",
		Excerpt:         "Test excerpt",
		Content:         "Test content",
		FeaturedImageID: uuid.New(),
		AuthorID:        uuid.New(),
	}

	t.Run("successful save", func(t *testing.T) {
		id, err := repo.SaveBlogPost(ctx, post)
		assert.NoError(t, err)
		assert.NotNil(t, id)
	})

	t.Run("empty fields validation", func(t *testing.T) {
		invalidPost := post
		invalidPost.Title = ""
		_, err := repo.SaveBlogPost(ctx, invalidPost)
		assert.Error(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		_, err := repo.SaveBlogPost(ctx, post)
		assert.Error(t, err)
	})
}

func TestUpdateBlogPostFields(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := repository.NewBlogRepository(pool)

	postID := uuid.New()
	now := time.Now()

	t.Run("successful update", func(t *testing.T) {
		err := repo.UpdateBlogPostFields(ctx, postID, map[string]interface{}{
			"title":        "New Title",
			"content":      "new-content",
			"published_at": now,
		})
		assert.NoError(t, err)
	})

	t.Run("successful update one filter", func(t *testing.T) {
		err := repo.UpdateBlogPostFields(ctx, postID, map[string]interface{}{
			"title": "New Title",
		})
		assert.NoError(t, err)
	})

	t.Run("invalid field", func(t *testing.T) {
		err := repo.UpdateBlogPostFields(ctx, postID, map[string]interface{}{
			"invalid_field": "value",
		})
		assert.Error(t, err)
	})

	t.Run("no fields to update", func(t *testing.T) {
		err := repo.UpdateBlogPostFields(ctx, postID, map[string]interface{}{})
		assert.Error(t, err)
	})
}

func TestDeleteBlogPost(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := repository.NewBlogRepository(pool)

	post := models.BlogPost{
		Title:           "Test Post",
		Slug:            "test-post",
		Excerpt:         "Test excerpt",
		Content:         "Test content",
		FeaturedImageID: uuid.New(),
		AuthorID:        uuid.New(),
	}

	postID, err := repo.SaveBlogPost(ctx, post)
	assert.NoError(t, err)

	t.Run("successful delete", func(t *testing.T) {
		err := repo.DeleteBlogPost(ctx, postID)
		assert.NoError(t, err)
	})

	t.Run("post not found", func(t *testing.T) {
		err := repo.DeleteBlogPost(ctx, postID)
		assert.Error(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		err := repo.DeleteBlogPost(ctx, postID)
		assert.Error(t, err)
	})
}

func TestBlogRepo_GetBlogPostByID(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := repository.NewBlogRepository(pool)
	id := uuid.New()

	testPost := models.BlogPost{
		ID:              id,
		Slug:            "Post1",
		Status:          "published",
		FeaturedImageID: uuid.New(),
	}

	postID, err := repo.SaveBlogPost(ctx, testPost)
	require.NoError(t, err)

	t.Run("get blog_post by id", func(t *testing.T) {
		post, err := repo.GetBlogPostByID(ctx, postID)
		assert.NoError(t, err)
		assert.Equal(t, post.ID, postID)
	})
}

func TestBlogRepo_GetBlogPosts(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := repository.NewBlogRepository(pool)

	// Подготовка тестовых данных
	now := time.Now()
	testPosts := []models.BlogPost{
		{
			Title:  "Published Post 1",
			Slug:   "Post1",
			Status: "published",
			// PublishedAt: now.Add(-24 * time.Hour),
			CreatedAt: now.Add(-48 * time.Hour),
		},
		{
			Title:     "Draft Post 1",
			Slug:      "Post2",
			Status:    "draft",
			CreatedAt: now.Add(-12 * time.Hour),
		},
		{
			Title:     "Draft Post 2",
			Slug:      "Post3",
			Status:    "draft",
			CreatedAt: now.Add(-6 * time.Hour),
		},
		{
			Title:  "Published Post 2",
			Slug:   "Post4",
			Status: "published",
			// PublishedAt: now.Add(-2 * time.Hour),
			CreatedAt: now.Add(-3 * time.Hour),
		},
		{
			Title:     "Archived Post 1",
			Slug:      "Post5",
			Status:    "archived",
			CreatedAt: now.Add(-72 * time.Hour),
		},
	}

	// Создаем тестовые посты в БД
	for i := range testPosts {
		_, err := repo.SaveBlogPost(ctx, testPosts[i])
		require.NoError(t, err)
	}

	t.Run("successful get all posts", func(t *testing.T) {
		posts, total, err := repo.GetBlogPosts(ctx, "all", 1, 10)
		require.NoError(t, err)
		assert.Equal(t, len(testPosts), total)
		assert.Len(t, posts, len(testPosts))
	})

	t.Run("successful get published posts", func(t *testing.T) {
		posts, _, err := repo.GetBlogPosts(ctx, "published", 1, 10)
		require.NoError(t, err)
		assert.Len(t, posts, 2)
		for _, post := range posts {
			assert.Equal(t, "published", post.Status)
		}
	})

	t.Run("successful get draft posts with pagination", func(t *testing.T) {
		// Первая страница - 1 запись
		posts, _, err := repo.GetBlogPosts(ctx, "draft", 1, 1)
		require.NoError(t, err)
		assert.Len(t, posts, 1)
		assert.Equal(t, "Draft Post 2", posts[0].Title)

		// Вторая страница - 1 запись
		posts, _, err = repo.GetBlogPosts(ctx, "draft", 2, 1)
		require.NoError(t, err)
		assert.Len(t, posts, 1)
		assert.Equal(t, "Draft Post 1", posts[0].Title)
	})

	t.Run("successful get archived posts", func(t *testing.T) {
		posts, _, err := repo.GetBlogPosts(ctx, "archived", 1, 10)
		require.NoError(t, err)
		assert.Len(t, posts, 1)
		assert.Equal(t, "Archived Post 1", posts[0].Title)
	})

	t.Run("invalid status filter", func(t *testing.T) {
		_, _, err := repo.GetBlogPosts(ctx, "invalid_status", 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status filter")
	})

	t.Run("automatic page correction", func(t *testing.T) {
		// Страница 0 должна корректироваться на 1
		posts, total, err := repo.GetBlogPosts(ctx, "all", 0, 10)
		require.NoError(t, err)
		assert.Equal(t, len(testPosts), total)
		assert.Len(t, posts, len(testPosts))

		// perPage > 100 должно корректироваться на 10
		posts, _, err = repo.GetBlogPosts(ctx, "all", 1, 101)
		require.NoError(t, err)
		assert.Len(t, posts, 5)
	})

	t.Run("empty result", func(t *testing.T) {
		// Пытаемся получить несуществующий статус
		posts, _, err := repo.GetBlogPosts(ctx, "archived", 2, 10)
		require.NoError(t, err)
		assert.Empty(t, posts)
	})
}

func TestMediaGroupOperations(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := repository.NewBlogRepository(pool)

	postID := uuid.New()
	groupID := uuid.New()

	t.Run("successful add media group", func(t *testing.T) {
		err := repo.AddMediaGroupToPost(ctx, postID, groupID, "gallery")
		assert.NoError(t, err)
	})

	// t.Run("get media groups", func(t *testing.T) {
	// 	groups, err := repo.GetPostMediaGroups(ctx, postID, "")
	// 	assert.NoError(t, err)
	// 	assert.Len(t, groups, 1)
	// 	assert.Equal(t, groupID, groups[0])
	// })
}

func TestGalleryRepo_CreateGallery(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGalleryRepo(db)

	now := time.Now().UTC()
	testCtx := context.Background()

	tests := []struct {
		name     string
		gallery  models.Gallery
		wantErr  bool
		preSetup func() uuid.UUID // Функция для предварительной настройки теста
	}{
		{
			name: "successful creation",
			gallery: models.Gallery{
				Title:           "Test Gallery",
				Slug:            "test-gallery",
				Description:     "Test description",
				Images:          []string{"image1.jpg", "image2.jpg"},
				CoverImageIndex: 0,
				AuthorID:        uuid.New(),
				Status:          "draft",
				Metadata:        map[string]interface{}{"key": "value"},
				Tags:            []string{"tag1", "tag2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preSetup != nil {
				existingID := tt.preSetup()
				if tt.gallery.Slug == "" {
					// Получаем slug существующей галереи
					var existingSlug string
					err := db.QueryRow(testCtx,
						"SELECT slug FROM galleries WHERE id = $1", existingID).Scan(&existingSlug)
					require.NoError(t, err)
					tt.gallery.Slug = existingSlug
				}
			}

			id, err := repo.CreateGallery(testCtx, tt.gallery)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEqual(t, uuid.Nil, id)

			// Проверяем, что данные записались в БД
			var dbGallery models.Gallery
			err = db.QueryRow(testCtx,
				`SELECT id, title, slug, description, images, cover_image_index, 
				 author_id, status, metadata, tags, created_at, updated_at
				 FROM galleries WHERE id = $1`, id).
				Scan(
					&dbGallery.ID,
					&dbGallery.Title,
					&dbGallery.Slug,
					&dbGallery.Description,
					&dbGallery.Images,
					&dbGallery.CoverImageIndex,
					&dbGallery.AuthorID,
					&dbGallery.Status,
					&dbGallery.Metadata,
					&dbGallery.Tags,
					&dbGallery.CreatedAt,
					&dbGallery.UpdatedAt,
				)
			require.NoError(t, err)

			require.Equal(t, tt.gallery.Title, dbGallery.Title)
			require.Equal(t, tt.gallery.Slug, dbGallery.Slug)
			require.Equal(t, tt.gallery.Description, dbGallery.Description)
			require.Equal(t, tt.gallery.Images, dbGallery.Images)
			require.Equal(t, tt.gallery.CoverImageIndex, dbGallery.CoverImageIndex)
			require.Equal(t, tt.gallery.AuthorID, dbGallery.AuthorID)
			require.Equal(t, tt.gallery.Status, dbGallery.Status)
			require.Equal(t, tt.gallery.Metadata, dbGallery.Metadata)
			require.Equal(t, tt.gallery.Tags, dbGallery.Tags)
			require.WithinDuration(t, now, dbGallery.CreatedAt, time.Second)
			require.WithinDuration(t, now, dbGallery.UpdatedAt, time.Second)
		})
	}
}

func TestGalleryRepo_UpdateGallery(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGalleryRepo(db)
	testCtx := context.Background()

	// Создаем тестовую галерею
	gallery := models.Gallery{
		Title:    "Original Title",
		Slug:     "original-slug",
		AuthorID: uuid.New(),
		Images:   []string{"new1.jpg", "new2.jpg"},
	}
	id, err := repo.CreateGallery(testCtx, gallery)
	require.NoError(t, err)

	tests := []struct {
		name    string
		updates models.Gallery
		wantErr bool
	}{
		{
			name: "successful update",
			updates: models.Gallery{
				ID:              id,
				Title:           "Updated Title",
				Slug:            "updated-slug",
				Description:     "Updated description",
				Images:          []string{"new1.jpg", "new2.jpg"},
				CoverImageIndex: 1,
				Status:          "published",
				Metadata:        map[string]interface{}{"new": "data"},
				Tags:            []string{"new", "tags"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateGallery(testCtx, tt.updates)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Проверяем обновленные данные
			var dbGallery models.Gallery
			err = db.QueryRow(testCtx,
				`SELECT title, slug, description, images, cover_image_index, 
				 status, metadata, tags, updated_at
				 FROM galleries WHERE id = $1`, id).
				Scan(
					&dbGallery.Title,
					&dbGallery.Slug,
					&dbGallery.Description,
					&dbGallery.Images,
					&dbGallery.CoverImageIndex,
					&dbGallery.Status,
					&dbGallery.Metadata,
					&dbGallery.Tags,
					&dbGallery.UpdatedAt,
				)
			require.NoError(t, err)

			require.Equal(t, tt.updates.Title, dbGallery.Title)
			require.Equal(t, tt.updates.Slug, dbGallery.Slug)
			require.Equal(t, tt.updates.Description, dbGallery.Description)
			require.Equal(t, tt.updates.Images, dbGallery.Images)
			require.Equal(t, tt.updates.CoverImageIndex, dbGallery.CoverImageIndex)
			require.Equal(t, tt.updates.Status, dbGallery.Status)
			require.Equal(t, tt.updates.Metadata, dbGallery.Metadata)
			require.Equal(t, tt.updates.Tags, dbGallery.Tags)
			require.WithinDuration(t, time.Now().UTC(), dbGallery.UpdatedAt, time.Second)
		})
	}
}

func TestGalleryRepo_DeleteGallery(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGalleryRepo(db)
	testCtx := context.Background()

	// Создаем тестовую галерею
	gallery := models.Gallery{
		Title:    "To be deleted",
		Slug:     "delete-me",
		AuthorID: uuid.New(),
		Images:   []string{"new1.jpg", "new2.jpg"},
	}
	id, err := repo.CreateGallery(testCtx, gallery)
	require.NoError(t, err)

	t.Run("successful deletion", func(t *testing.T) {
		err := repo.DeleteGallery(testCtx, id)
		require.NoError(t, err)

		// Проверяем, что галерея удалена
		var count int
		err = db.QueryRow(testCtx,
			"SELECT COUNT(*) FROM galleries WHERE id = $1", id).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})
}

func TestGalleryRepo_GetGalleryByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGalleryRepo(db)
	testCtx := context.Background()

	// Создаем тестовую галерею
	expected := models.Gallery{
		Title:           "Test Get Gallery",
		Slug:            "test-get-gallery",
		Description:     "Test description for get",
		Images:          []string{"get1.jpg", "get2.jpg"},
		CoverImageIndex: 1,
		AuthorID:        uuid.New(),
		Status:          "published",
		Metadata:        map[string]interface{}{"get": "test"},
		Tags:            []string{"get", "test"},
	}
	id, err := repo.CreateGallery(testCtx, expected)
	require.NoError(t, err)
	expected.ID = id

	t.Run("successful get", func(t *testing.T) {
		result, err := repo.GetGalleryByID(testCtx, id)
		require.NoError(t, err)

		require.Equal(t, expected.ID, result.ID)
		require.Equal(t, expected.Title, result.Title)
		require.Equal(t, expected.Slug, result.Slug)
		require.Equal(t, expected.Description, result.Description)
		require.Equal(t, expected.Images, result.Images)
		require.Equal(t, expected.CoverImageIndex, result.CoverImageIndex)
		require.Equal(t, expected.AuthorID, result.AuthorID)
		require.Equal(t, expected.Status, result.Status)
		require.Equal(t, expected.Metadata, result.Metadata)
		require.Equal(t, expected.Tags, result.Tags)
		require.False(t, result.CreatedAt.IsZero())
		require.False(t, result.UpdatedAt.IsZero())
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetGalleryByID(testCtx, uuid.New())
		require.Error(t, err)
		require.Equal(t, err, err)
	})
}

func TestGalleryRepo_GetGalleries(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGalleryRepo(db)
	testCtx := context.Background()

	// Создаем тестовые галереи с разными статусами
	gallery1 := models.Gallery{
		Title:           "Published Gallery",
		Slug:            "published-gallery",
		Status:          "published",
		AuthorID:        uuid.New(),
		Images:          []string{"img1.jpg"},
		CoverImageIndex: 0,
	}

	gallery2 := models.Gallery{
		Title:           "Draft Gallery",
		Slug:            "draft-gallery",
		Status:          "draft",
		AuthorID:        uuid.New(),
		Images:          []string{"img2.jpg"},
		CoverImageIndex: 0,
	}

	_, err := repo.CreateGallery(testCtx, gallery1)
	require.NoError(t, err)
	_, err = repo.CreateGallery(testCtx, gallery2)
	require.NoError(t, err)

	t.Run("get all galleries", func(t *testing.T) {
		galleries, total, err := repo.GetGalleries(testCtx, "all", 1, 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, total, 2)
		require.GreaterOrEqual(t, len(galleries), 2)
	})

	t.Run("filter by published status", func(t *testing.T) {
		galleries, total, err := repo.GetGalleries(testCtx, "published", 1, 10)
		require.NoError(t, err)

		require.GreaterOrEqual(t, total, 1)
		require.GreaterOrEqual(t, len(galleries), 1)
		require.Equal(t, "published", galleries[0].Status)
	})

	t.Run("filter by draft status", func(t *testing.T) {
		galleries, total, err := repo.GetGalleries(testCtx, "draft", 1, 10)
		require.NoError(t, err)

		require.GreaterOrEqual(t, total, 1)
		require.GreaterOrEqual(t, len(galleries), 1)
		require.Equal(t, "draft", galleries[0].Status)
	})

	t.Run("pagination works", func(t *testing.T) {
		// Первая страница - 1 запись
		galleries, total, err := repo.GetGalleries(testCtx, "all", 1, 1)
		require.NoError(t, err)

		require.GreaterOrEqual(t, total, 2)
		require.Equal(t, 1, len(galleries))

		// Вторая страница - следующая запись
		galleries, _, err = repo.GetGalleries(testCtx, "all", 2, 1)
		require.NoError(t, err)
		require.Equal(t, 1, len(galleries))
	})

	t.Run("invalid status filter", func(t *testing.T) {
		_, _, err := repo.GetGalleries(testCtx, "invalid_status", 1, 10)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid status filter")
	})

	t.Run("empty result", func(t *testing.T) {
		galleries, total, err := repo.GetGalleries(testCtx, "archived", 1, 10)
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Empty(t, galleries)
	})
}

func TestTagRepository_GetGalleriesByTags(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGalleryRepo(db)
	testCtx := context.Background()

	// Создаем тестовые галереи с разными тегами
	gallery1 := models.Gallery{
		Title:           "Gallery with tags",
		Slug:            "gallery-with-tags",
		Status:          "published",
		AuthorID:        uuid.New(),
		Images:          []string{"img1.jpg"},
		CoverImageIndex: 0,
		Tags:            []string{"nature", "landscape"},
	}

	gallery2 := models.Gallery{
		Title:           "Gallery with single tag",
		Slug:            "gallery-with-single-tag",
		Status:          "published",
		AuthorID:        uuid.New(),
		Images:          []string{"img2.jpg"},
		CoverImageIndex: 0,
		Tags:            []string{"art"},
	}

	_, err := repo.CreateGallery(testCtx, gallery1)
	require.NoError(t, err)
	_, err = repo.CreateGallery(testCtx, gallery2)
	require.NoError(t, err)

	t.Run("filter with AND logic", func(t *testing.T) {
		galleries, err := repo.GetGalleriesByTags(testCtx, []string{"nature", "landscape"}, true)
		require.NoError(t, err)
		require.Equal(t, 1, len(galleries))
		require.Equal(t, "gallery-with-tags", galleries[0].Slug)
	})

	t.Run("filter with OR logic", func(t *testing.T) {
		galleries, err := repo.GetGalleriesByTags(testCtx, []string{"nature", "art"}, false)
		require.NoError(t, err)
		require.Equal(t, 2, len(galleries))
	})

	t.Run("no matching tags", func(t *testing.T) {
		galleries, err := repo.GetGalleriesByTags(testCtx, []string{"unknown"}, true)
		require.NoError(t, err)
		require.Empty(t, galleries)
	})

	t.Run("empty tags list", func(t *testing.T) {
		galleries, err := repo.GetGalleriesByTags(testCtx, []string{}, true)
		require.NoError(t, err)
		require.Equal(t, 2, len(galleries))
	})

	t.Run("partial match with AND logic", func(t *testing.T) {
		galleries, err := repo.GetGalleriesByTags(testCtx, []string{"nature", "art"}, true)
		require.NoError(t, err)
		require.Empty(t, galleries)
	})

	t.Run("partial match with OR logic", func(t *testing.T) {
		galleries, err := repo.GetGalleriesByTags(testCtx, []string{"nature", "art"}, false)
		require.NoError(t, err)
		require.Equal(t, 2, len(galleries))
	})
}

func TestGalleryRepo_TagsOperations(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewGalleryRepo(db)
	ctx := context.Background()

	// Создаем тестовые галереи с разными тегами
	gallery1 := models.Gallery{
		Title:  "Nature Gallery",
		Slug:   "nature-gallery",
		Images: []string{"img2.jpg"},
		Tags:   []string{"nature", "landscape"},
	}

	gallery2 := models.Gallery{
		Title:  "Art Gallery",
		Slug:   "art-gallery",
		Images: []string{"img2.jpg"},
		Tags:   []string{"art", "painting"},
	}

	gallery1ID, err := repo.CreateGallery(ctx, gallery1)
	require.NoError(t, err)
	gallery2ID, err := repo.CreateGallery(ctx, gallery2)
	require.NoError(t, err)

	t.Run("AddTags - добавление новых тегов", func(t *testing.T) {
		err := repo.AddTags(ctx, gallery1ID.String(), []string{"sunset", "mountains"})
		require.NoError(t, err)

		tags, err := repo.GetTags(ctx, gallery1ID.String())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"nature", "landscape", "sunset", "mountains"}, tags)
	})

	t.Run("RemoveTags - удаление тегов", func(t *testing.T) {
		// Удаляем тег
		err = repo.RemoveTags(ctx, gallery2ID.String(), []string{"painting"})
		require.NoError(t, err)

		// Проверяем результат
		updatedGallery, err := repo.GetGalleryByID(ctx, gallery2ID)
		require.NoError(t, err)
		require.NotContains(t, updatedGallery.Tags, "painting")
		require.ElementsMatch(t, []string{"art"}, updatedGallery.Tags)
	})

	t.Run("UpdateTags - полное обновление тегов", func(t *testing.T) {
		err := repo.UpdateTags(ctx, gallery1ID.String(), []string{"new", "tags"})
		require.NoError(t, err)

		tags, err := repo.GetTags(ctx, gallery1ID.String())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"new", "tags"}, tags)
	})

	t.Run("HasTags - проверка наличия тегов", func(t *testing.T) {
		// Галерея содержит все указанные теги
		has, err := repo.HasTags(ctx, gallery1ID.String(), []string{"new", "tags"})
		require.NoError(t, err)
		require.True(t, has)

		// Галерея не содержит все указанные теги
		has, err = repo.HasTags(ctx, gallery1ID.String(), []string{"new", "unknown"})
		require.NoError(t, err)
		require.False(t, has)

		// Пустой список тегов всегда возвращает true
		has, err = repo.HasTags(ctx, gallery1ID.String(), []string{})
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("GetTags - получение тегов галереи", func(t *testing.T) {
		tags, err := repo.GetTags(ctx, gallery2ID.String())
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"art"}, tags)

		// Несуществующая галерея
		_, err = repo.GetTags(ctx, uuid.New().String())
		require.Error(t, err)
	})

	t.Run("Edge cases - граничные случаи", func(t *testing.T) {
		// Добавление пустого списка тегов
		err := repo.AddTags(ctx, gallery1ID.String(), []string{})
		require.NoError(t, err)

		// Удаление несуществующих тегов
		err = repo.RemoveTags(ctx, gallery1ID.String(), []string{"unknown"})
		require.NoError(t, err)

		// Обновление пустым списком тегов
		err = repo.UpdateTags(ctx, gallery1ID.String(), []string{})
		require.NoError(t, err)
		tags, err := repo.GetTags(ctx, gallery1ID.String())
		require.NoError(t, err)
		require.Empty(t, tags)
	})
}
