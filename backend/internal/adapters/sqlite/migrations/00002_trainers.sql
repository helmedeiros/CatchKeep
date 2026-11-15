-- +goose Up
CREATE TABLE trainers (
    id           TEXT PRIMARY KEY,
    display_name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    pin_hash     TEXT NOT NULL,
    created_at   TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE trainers;
