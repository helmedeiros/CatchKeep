-- +goose Up
CREATE TABLE teams (
    id         TEXT PRIMARY KEY,
    trainer_id TEXT NOT NULL REFERENCES trainers (id) ON DELETE CASCADE,
    name       TEXT NOT NULL
) STRICT;

CREATE INDEX teams_trainer_id ON teams (trainer_id);

CREATE TABLE team_members (
    team_id    TEXT NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    pokemon_id TEXT NOT NULL REFERENCES pokemon (id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    PRIMARY KEY (team_id, pokemon_id)
) STRICT;

-- +goose Down
DROP TABLE team_members;
DROP TABLE teams;
