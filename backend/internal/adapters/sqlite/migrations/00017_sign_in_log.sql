-- +goose Up
CREATE TABLE sign_in_log (
    id         INTEGER PRIMARY KEY,
    kind       TEXT NOT NULL CHECK (kind IN ('signed_in', 'wrong_pin', 'paused', 'profile_created', 'pin_reset')),
    trainer_id TEXT REFERENCES trainers (id) ON DELETE CASCADE,
    device     TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

CREATE INDEX sign_in_log_created ON sign_in_log (created_at);

-- +goose Down
DROP TABLE sign_in_log;
