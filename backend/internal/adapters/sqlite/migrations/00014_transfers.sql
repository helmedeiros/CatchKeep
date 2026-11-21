-- +goose Up
CREATE TABLE transfer_events (
    id         TEXT PRIMARY KEY,
    trainer_id TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('single', 'bulk', 'undo')),
    pokemon    TEXT NOT NULL,
    request_id TEXT,
    undoes     TEXT REFERENCES transfer_events (id),
    undone_by  TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (trainer_id, request_id)
) STRICT;

CREATE INDEX transfer_events_trainer ON transfer_events (trainer_id, created_at);

-- +goose Down
DROP TABLE transfer_events;
