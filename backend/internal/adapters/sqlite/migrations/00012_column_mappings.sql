-- +goose Up
ALTER TABLE import_batches ADD COLUMN source_table TEXT;
ALTER TABLE import_batches ADD COLUMN column_mapping TEXT;

CREATE TABLE column_mappings (
    trainer_id TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    signature  TEXT NOT NULL,
    mapping    TEXT NOT NULL,
    PRIMARY KEY (trainer_id, signature)
) STRICT;

-- +goose Down
DROP TABLE column_mappings;
ALTER TABLE import_batches DROP COLUMN column_mapping;
ALTER TABLE import_batches DROP COLUMN source_table;
