-- +goose Up
ALTER TABLE trainer_setups ADD COLUMN categories TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE trainer_setups DROP COLUMN categories;
