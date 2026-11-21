-- +goose Up
ALTER TABLE trainer_setups ADD COLUMN in_game_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE trainer_setups ADD COLUMN storage_limit INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE trainer_setups DROP COLUMN storage_limit;
ALTER TABLE trainer_setups DROP COLUMN in_game_count;
