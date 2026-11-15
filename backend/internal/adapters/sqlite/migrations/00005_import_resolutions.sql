-- +goose Up
ALTER TABLE import_batches ADD COLUMN resolutions TEXT NOT NULL DEFAULT '{}';
ALTER TABLE import_batches ADD COLUMN updated_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE import_batches DROP COLUMN updated_count;
ALTER TABLE import_batches DROP COLUMN resolutions;
