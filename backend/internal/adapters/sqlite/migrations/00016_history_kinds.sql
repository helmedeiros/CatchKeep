-- +goose Up
-- The history log takes every kind of change the spec records (§13).
ALTER TABLE history_events RENAME TO history_events_old;

CREATE TABLE history_events (
    id         TEXT PRIMARY KEY,
    trainer_id TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('single', 'bulk', 'undo', 'import', 'protection', 'team', 'rules', 'setup')),
    pokemon    TEXT NOT NULL,
    detail     TEXT,
    request_id TEXT,
    undoes     TEXT REFERENCES history_events (id),
    undone_by  TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (trainer_id, request_id)
) STRICT;

INSERT INTO history_events SELECT id, trainer_id, kind, pokemon, detail, request_id, undoes, undone_by, created_at
FROM history_events_old ORDER BY rowid;

DROP TABLE history_events_old;

CREATE INDEX history_events_trainer ON history_events (trainer_id, created_at);

-- +goose Down
-- Migrations are forward-only (spec §13); older code reads the old kinds.
DELETE FROM history_events WHERE kind NOT IN ('single', 'bulk', 'undo');
