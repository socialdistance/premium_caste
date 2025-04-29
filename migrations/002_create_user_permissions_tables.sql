-- +goose Up
-- Enable pgcrypto extension for UUID generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE "public"."users" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "name" varchar NOT NULL,
    "email" varchar NOT NULL,
    "phone" varchar(255) NOT NULL,
    "password" varchar(255) NOT NULL,
    "is_admin" BOOLEAN DEFAULT FALSE,
    "basket_id" uuid,
    "registration_date" TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
    "last_login" timestamp
);

ALTER TABLE "public"."users" ADD CONSTRAINT "email" UNIQUE ("email");
ALTER TABLE "public"."users" ADD CONSTRAINT "phone" UNIQUE ("phone");

-- +goose Down
DROP TABLE users;
DROP TABLE permissions;
