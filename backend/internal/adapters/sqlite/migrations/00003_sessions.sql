-- +goose Up
CREATE TABLE sessions (
    token_hash   TEXT PRIMARY KEY,
    trainer_id   TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    created_at   TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
) STRICT;

CREATE INDEX sessions_trainer_id ON sessions (trainer_id);

-- +goose Down
DROP TABLE sessions;
