-- +goose Up

-- Enable pgcrypto extension for UUID generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_id UUID NOT NULL,               -- ID пользователя, загрузившего файл
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- Дата/время загрузки
    media_type VARCHAR(20) NOT NULL,         -- Тип медиа: 'photo', 'video', 'document' и т.д.
    original_filename VARCHAR(255) NOT NULL,  -- Оригинальное имя файла
    storage_path TEXT NOT NULL,              -- Путь к файлу в хранилище
    file_size BIGINT NOT NULL,               -- Размер файла в байтах
    mime_type VARCHAR(100),                 -- MIME-тип файла
    width INT,                              -- Ширина (для фото/видео)
    height INT,                             -- Высота (для фото/видео)
    duration INT,                           -- Длительность в секундах (для видео)
    is_public BOOLEAN DEFAULT FALSE,        -- Публичный ли файл
    metadata JSONB                          -- Дополнительные метаданные
);

-- Индексы для ускорения поиска
CREATE INDEX idx_media_uploader ON media(uploader_id);
CREATE INDEX idx_media_type ON media(media_type);
CREATE INDEX idx_media_created ON media(created_at);

-- Таблица для группировки медиа
CREATE TABLE media_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    owner_id UUID NOT NULL,                  -- Владелец группы (например, пользователь)
    description TEXT                        -- Описание группы
);

-- Связующая таблица
CREATE TABLE media_group_items (
    group_id UUID NOT NULL REFERENCES media_groups(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    position INT NOT NULL DEFAULT 0,         -- Позиция в группе (для сортировки)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, media_id)
);

CREATE INDEX idx_media_group_items_media ON media_group_items(media_id);

-- Добавляем файл
INSERT INTO media (uploader_id, media_type, original_filename, storage_path, file_size)
VALUES ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'photo', 'cat.jpg', '/uploads/2023/cat.jpg', 1024);

INSERT INTO media (uploader_id, media_type, original_filename, storage_path, file_size)
VALUES ('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'photo', 'cat123.jpg', '/uploads/2023/cat123.jpg', 1024);

-- Создаем группу
INSERT INTO media_groups (owner_id, description)
VALUES ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Мои котики');

-- Добавляем файл в группу
-- INSERT INTO media_group_items (group_id, media_id)
-- VALUES ('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33');


-- SELECT m.* 
-- FROM media m
-- JOIN media_group_items mgi ON m.id = mgi.media_id
-- WHERE mgi.group_id = 'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22'
-- ORDER BY mgi.position;

-- +goose Down