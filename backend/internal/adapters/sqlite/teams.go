package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Teams is the SQLite ports.Teams.
type Teams struct {
	db *sql.DB
}

var _ ports.Teams = (*Teams)(nil)

// NewTeams returns a team store backed by store.
func NewTeams(store *Store) *Teams {
	return &Teams{db: store.db}
}

// ListTeams implements ports.Teams.
func (r *Teams) ListTeams(ctx context.Context, trainer domain.TrainerID) ([]domain.Team, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, m.pokemon_id
		FROM teams t LEFT JOIN team_members m ON m.team_id = t.id
		WHERE t.trainer_id = ?
		ORDER BY t.name COLLATE NOCASE, t.id, m.position`, string(trainer))
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Team
	for rows.Next() {
		var (
			id, name string
			member   sql.NullString
		)
		if err := rows.Scan(&id, &name, &member); err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}
		if len(out) == 0 || out[len(out)-1].ID != domain.TeamID(id) {
			out = append(out, domain.Team{ID: domain.TeamID(id), Name: name, Members: []domain.PokemonID{}})
		}
		if member.Valid {
			t := &out[len(out)-1]
			t.Members = append(t.Members, domain.PokemonID(member.String))
		}
	}
	return out, rows.Err()
}

// SaveTeam implements ports.Teams in a single transaction.
func (r *Teams) SaveTeam(ctx context.Context, trainer domain.TrainerID, t domain.Team, c domain.Change) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO teams (id, trainer_id, name) VALUES (?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET name = excluded.name WHERE teams.trainer_id = excluded.trainer_id`,
		string(t.ID), string(trainer), t.Name)
	if err != nil {
		return fmt.Errorf("save team: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("save team: %w", err)
	} else if n == 0 {
		return domain.ErrTeamNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM team_members WHERE team_id = ?`, string(t.ID)); err != nil {
		return fmt.Errorf("clear members: %w", err)
	}
	for i, m := range t.Members {
		var owned bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM pokemon WHERE id = ? AND trainer_id = ? AND status = 'active')`,
			string(m), string(trainer)).Scan(&owned)
		if err != nil {
			return fmt.Errorf("check member: %w", err)
		}
		if !owned {
			return domain.ErrPokemonNotFound
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO team_members (team_id, pokemon_id, position) VALUES (?, ?, ?)`,
			string(t.ID), string(m), i); err != nil {
			return fmt.Errorf("add member: %w", err)
		}
	}
	if err := insertChange(ctx, tx, c, teamDetail{TeamID: string(t.ID), Name: t.Name}); err != nil {
		return err
	}
	if err := bumpRevision(ctx, tx, trainer); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// DeleteTeam implements ports.Teams.
func (r *Teams) DeleteTeam(ctx context.Context, trainer domain.TrainerID, id domain.TeamID, c domain.Change) error {
	return inTx(ctx, r.db, trainer, func(tx *sql.Tx) error {
		var name string
		err := tx.QueryRowContext(ctx, `DELETE FROM teams WHERE id = ? AND trainer_id = ? RETURNING name`, string(id), string(trainer)).Scan(&name)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrTeamNotFound
		}
		if err != nil {
			return fmt.Errorf("delete team: %w", err)
		}
		return insertChange(ctx, tx, c, teamDetail{TeamID: string(id), Name: name, Deleted: true})
	})
}

type teamDetail struct {
	TeamID  string `json:"teamId"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted,omitempty"`
}
