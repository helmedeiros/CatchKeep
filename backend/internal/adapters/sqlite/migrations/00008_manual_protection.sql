-- +goose Up
ALTER TABLE pokemon ADD COLUMN manual_protected INTEGER NOT NULL DEFAULT 0 CHECK (manual_protected IN (0, 1));

-- +goose Down
ALTER TABLE pokemon DROP COLUMN manual_protected;
