-- +goose Up
CREATE TABLE trainer_setups (
    trainer_id TEXT PRIMARY KEY REFERENCES trainers (id) ON DELETE CASCADE,
    capacity   INTEGER NOT NULL CHECK (capacity > 0),
    goals      TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE trainer_setups;
