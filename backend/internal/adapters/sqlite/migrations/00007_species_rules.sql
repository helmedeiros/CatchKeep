-- +goose Up
ALTER TABLE trainer_setups ADD COLUMN species TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE trainer_setups DROP COLUMN species;
