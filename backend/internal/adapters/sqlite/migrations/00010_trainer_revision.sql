-- +goose Up
ALTER TABLE trainers ADD COLUMN revision INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE trainers DROP COLUMN revision;
