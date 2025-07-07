-- +goose Up
CREATE TABLE galleries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    images TEXT[] NOT NULL DEFAULT '{}',  -- Массив путей/URL изображений
    cover_image_index INT DEFAULT 0,      -- Индекс обложки в массиве images
    author_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB,
    tags VARCHAR(255)[] DEFAULT '{}'      -- Массив тегов
);

-- Индексы для поиска по тегам и массивам
CREATE INDEX idx_galleries_tags ON galleries USING GIN(tags);
CREATE INDEX idx_galleries_images ON galleries USING GIN(images);

-- +goose Down
DROP TABLE galleries;


