-- +goose Up

-- 1. Только фотографии (media_type = 'photo')
INSERT INTO media (
    id, uploader_id, created_at, media_type, 
    original_filename, storage_path, file_size, 
    mime_type, width, height, is_public, metadata
) VALUES
    -- Фото вулкана (основное изображение поста)
    (
        'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 
        'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 
        '2025-04-28 10:00:00+03', 
        'photo', 
        'volcano.jpg', 
        'uploads/photos/2025/04/volcano_abc123.jpg', 
        2500000, 
        'image/jpeg', 
        1920, 
        1080, 
        TRUE, 
        '{"tags": ["nature", "volcano"], "camera": "Sony A7IV"}'
    ),
    
    -- Фото леса (для галереи)
    (
        'd0f7e17a-f0a5-455c-a585-ddf6332c6466', 
        'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 
        '2025-04-27 15:30:00+03', 
        'photo', 
        'forest.png', 
        'uploads/photos/2025/04/forest_xyz456.png', 
        1800000, 
        'image/png', 
        1200, 
        800, 
        FALSE, 
        '{"alt": "Лесной пейзаж Камчатки", "location": "55.1234, 160.5678"}'
    ),
    
    -- Дополнительное фото для галереи
    (
        'f3c214e6-3d45-4a2a-bf88-5d8a1e2c1b12', 
        'c3eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 
        '2025-04-26 11:20:00+03', 
        'photo', 
        'mountain.jpg', 
        'uploads/photos/2025/04/mountain_def789.jpg', 
        3200000, 
        'image/jpeg', 
        2400, 
        1600, 
        TRUE, 
        '{"description": "Вид на вулкан Ключевская сопка"}'
    );

-- 2. Посты блога (только те, что связаны с фото)
INSERT INTO blog_posts (
    id, title, slug, excerpt, content, 
    featured_image_id, author_id, 
    status, published_at, metadata
) VALUES
    (
        'f8a1e2c1-b12d-4a2a-3d45-5d8a1e2c1b12', 
        'Путешествие по Камчатке', 
        'kamchatka-trip-2025', 
        'Фотоотчёт из экспедиции', 
        'Полный текст с описанием маршрута...', 
        'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',  -- Ссылка на фото вулкана
        'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 
        'published', 
        '2025-04-29 17:45:00+03', 
        '{"tags": ["путешествия", "фото"]}'
    );

-- 3. Группы медиа (только для фото)
INSERT INTO media_groups (id, created_at, owner_id, description) 
VALUES
    (
        'c5be2270-d863-4a05-9165-44843ea166bc', 
        '2025-04-28 10:00:00+03',
        'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', -- Пример ID владельца (замените на реальный)
        'Галерея из экспедиции'
    );
-- 4. Связи фото с группами
INSERT INTO media_group_items (group_id, media_id, position) 
VALUES
    ('c5be2270-d863-4a05-9165-44843ea166bc', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1),
    ('c5be2270-d863-4a05-9165-44843ea166bc', 'd0f7e17a-f0a5-455c-a585-ddf6332c6466', 2),
    ('c5be2270-d863-4a05-9165-44843ea166bc', 'f3c214e6-3d45-4a2a-bf88-5d8a1e2c1b12', 3);

-- 5. Связи постов с группами фото
INSERT INTO post_media_groups (post_id, group_id, relation_type) 
VALUES
    (
        'f8a1e2c1-b12d-4a2a-3d45-5d8a1e2c1b12', 
        'c5be2270-d863-4a05-9165-44843ea166bc', 
        'gallery'
    );

-- +goose Down

-- Получить все медиа для поста "Путешествие по Камчатке"
-- SELECT 
--     bp.title AS post_title,
--     mg.name AS group_name,
--     m.file_path,
--     m.metadata->>'type' AS media_type
-- FROM 
--     blog_posts bp
-- JOIN 
--     post_media_groups pmg ON bp.id = pmg.post_id
-- JOIN 
--     media_groups mg ON pmg.group_id = mg.id
-- JOIN 
--     media_group_items mgi ON mg.id = mgi.group_id
-- JOIN 
--     media m ON mgi.media_id = m.id
-- WHERE 
--     bp.id = 'f8a1e2c1-b12d-4a2a-3d45-5d8a1e2c1b12';