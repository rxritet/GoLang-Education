-- Инициализация схемы базы данных.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS books (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    author     TEXT NOT NULL,
    year       INT  NOT NULL CHECK (year > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
