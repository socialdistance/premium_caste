-- +goose Up
CREATE TABLE IF NOT EXISTS "public"."users" (
    "id" serial NOT NULL,
    "last_name" text,
    "first_name" text,
    "sur_name" text,
    "place_of_work" text,
    "position" text,
    "login" text,
    "tmp_passwd" text,
    "date_of_birthday" text,
    "snils" text,
    "complete" bool,
    "file_uuid" uuid,
    PRIMARY KEY ("id")
);
-- CREATE INDEX IF NOT EXISTS idx_last_name ON creates(LOWER(last_name));
CREATE INDEX IF NOT EXISTS idx_last_name ON creates(last_name);

CREATE TABLE IF NOT EXISTS "public"."recoverys" (
    "id" serial NOT NULL,
    "last_name" text,
    "first_name" text,
    "sur_name" text,
    "place_of_work" text,
    "position" text,
    "login" text,
    "tmp_passwd" text,
    "complete" bool,
    "file_uuid" uuid,
    PRIMARY KEY ("id")
);
-- CREATE INDEX IF NOT EXISTS idx_last_name ON recoverys(LOWER(last_name));
CREATE INDEX IF NOT EXISTS idx_last_name ON recoverys(last_name);

CREATE TABLE IF NOT EXISTS "public"."files" (
    "uuid" uuid NOT NULL,
    "file_path" text,
    "type" text,
    "extention" text,
    "created_at" timestamptz,
    PRIMARY KEY ("uuid")
);

-- +goose Down
DROP TABLE creates;
DROP TABLE recoverys;
DROP TABLE files;
