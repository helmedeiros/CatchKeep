-- +goose Up
CREATE TABLE import_batches (
    id                   TEXT PRIMARY KEY,
    trainer_id           TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    status               TEXT NOT NULL CHECK (status IN ('staged', 'committed')),
    parsed               TEXT NOT NULL,
    new_count            INTEGER NOT NULL,
    possible_match_count INTEGER NOT NULL,
    created_at           TEXT NOT NULL,
    committed_at         TEXT
) STRICT;

CREATE INDEX import_batches_trainer_id ON import_batches (trainer_id);

CREATE TABLE pokemon (
    id               TEXT PRIMARY KEY,
    trainer_id       TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'transferred', 'removed_by_refresh', 'merged')),
    created_batch_id TEXT REFERENCES import_batches (id)
) STRICT;

CREATE INDEX pokemon_trainer_status ON pokemon (trainer_id, status);

CREATE TABLE observations (
    id          INTEGER PRIMARY KEY,
    pokemon_id  TEXT NOT NULL REFERENCES pokemon (id) ON DELETE CASCADE,
    field       TEXT NOT NULL,
    value       TEXT NOT NULL,
    source      TEXT NOT NULL CHECK (source IN ('csv', 'screenshot', 'manual')),
    source_ref  TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    recorded_at TEXT NOT NULL
) STRICT;

CREATE INDEX observations_pokemon_id ON observations (pokemon_id);

-- +goose Down
DROP TABLE observations;
DROP TABLE pokemon;
DROP TABLE import_batches;
