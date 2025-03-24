-- +goose Up
CREATE TABLE "public"."permissions" (
    "id" int2 NOT NULL,
    "permission" varchar(255) NOT NULL,
    PRIMARY KEY ("id")
);

CREATE TABLE "public"."users" (
    "id" int2 NOT NULL DEFAULT nextval('users_id_seq1'::regclass),
    "name" varchar NOT NULL,
    "email" varchar NOT NULL,
    "phone" varchar(255) NOT NULL,
    "password" varchar(255) NOT NULL,
    "permission_id" int2 NOT NULL,
    "basket_id" int2,
    "registration_date" date,
    "last_login" date,
    PRIMARY KEY ("id")
);

COMMENT ON COLUMN "public"."users"."permission_id" IS 'id на таблицу permissions';
COMMENT ON COLUMN "public"."users"."basket_id" IS 'id на таблицу basket';

ALTER TABLE "public"."users" ADD CONSTRAINT "email" UNIQUE ("email");
ALTER TABLE "public"."users" ADD CONSTRAINT "phone" UNIQUE ("phone");


INSERT INTO "public"."permissions" ("id", "permission") VALUES (1, 'user');
INSERT INTO "public"."permissions" ("id", "permission") VALUES (2, 'moderator');
INSERT INTO "public"."permissions" ("id", "permission") VALUES (3, 'admin');

-- +goose Down
DROP TABLE users;
DROP TABLE permissions;
