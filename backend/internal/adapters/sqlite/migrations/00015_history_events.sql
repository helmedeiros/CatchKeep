-- +goose Up
-- Transfers become the first kinds of one history log (spec §13); later
-- kinds keep what they need in detail.
CREATE TABLE history_events (
    id         TEXT PRIMARY KEY,
    trainer_id TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('single', 'bulk', 'undo')),
    pokemon    TEXT NOT NULL,
    detail     TEXT,
    request_id TEXT,
    undoes     TEXT REFERENCES history_events (id),
    undone_by  TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (trainer_id, request_id)
) STRICT;

INSERT INTO history_events (id, trainer_id, kind, pokemon, request_id, undoes, undone_by, created_at)
SELECT id, trainer_id, kind, pokemon, request_id, undoes, undone_by, created_at FROM transfer_events ORDER BY rowid;

DROP TABLE transfer_events;

CREATE INDEX history_events_trainer ON history_events (trainer_id, created_at);

-- +goose Down
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

INSERT INTO transfer_events (id, trainer_id, kind, pokemon, request_id, undoes, undone_by, created_at)
SELECT id, trainer_id, kind, pokemon, request_id, undoes, undone_by, created_at FROM history_events
WHERE kind IN ('single', 'bulk', 'undo') ORDER BY rowid;

DROP TABLE history_events;

CREATE INDEX transfer_events_trainer ON transfer_events (trainer_id, created_at);
