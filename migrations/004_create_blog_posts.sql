-- +goose Up

-- Таблица постов блога
CREATE TABLE blog_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,                  -- Заголовок поста
    slug VARCHAR(255) UNIQUE NOT NULL,           -- URL-дружественный идентификатор
    excerpt TEXT,                                -- Краткое описание
    content TEXT NOT NULL,                       -- Основной текст поста
    featured_image_id UUID REFERENCES media(id),  -- Главное изображение поста
    author_id UUID NOT NULL,                     -- Автор поста (UUID пользователя)
    status VARCHAR(20) NOT NULL DEFAULT 'draft',  -- Статус: draft/published/archived
    published_at TIMESTAMPTZ,                    -- Дата публикации
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB                               -- Дополнительные метаданные
);

-- Таблица связи постов с медиа-группами
CREATE TABLE post_media_groups (
    post_id UUID NOT NULL REFERENCES blog_posts(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES media_groups(id) ON DELETE CASCADE,
    relation_type VARCHAR(30) NOT NULL DEFAULT 'content', -- Тип связи: content/gallery/attachment
    PRIMARY KEY (post_id, group_id)
);

-- Индексы для блога
CREATE INDEX idx_blog_posts_author ON blog_posts(author_id);
CREATE INDEX idx_blog_posts_status ON blog_posts(status);
CREATE INDEX idx_blog_posts_published ON blog_posts(published_at);
CREATE INDEX idx_post_media_groups_post ON post_media_groups(post_id);
CREATE INDEX idx_post_media_groups_group ON post_media_groups(group_id);

-- +goose Down
DROP TRIGGER IF EXISTS trigger_blog_post_timestamp ON blog_posts;
DROP FUNCTION IF EXISTS update_blog_post_timestamp;
DROP TABLE IF EXISTS post_media_groups;
DROP TABLE IF EXISTS blog_posts;

