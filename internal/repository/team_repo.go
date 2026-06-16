package repository

import (
	"context"
	"database/sql"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/jmoiron/sqlx"
)

// teamRepo implements TeamRepository using PostgreSQL.
// LEARNING NOTE: Concrete repository SQL'i bilir; dış katmanlar ise sadece TeamRepository interface'ini görür.
type teamRepo struct {
	db *sqlx.DB
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(db *sqlx.DB) TeamRepository {
	return &teamRepo{db: db}
}

// GetAll returns all teams ordered by ID.
func (r *teamRepo) GetAll(ctx context.Context) ([]models.Team, error) {
	// LEARNING NOTE: sqlx SelectContext, query sonucundaki satırları Go struct slice'ına map eder.
	var teams []models.Team
	err := r.db.SelectContext(ctx, &teams, "SELECT * FROM teams ORDER BY id")
	return teams, err
}

// Count returns the total number of teams.
func (r *teamRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM teams")
	return count, err
}

// Upsert creates a team or updates the ratings of an existing team with the same name.
func (r *teamRepo) Upsert(ctx context.Context, team models.Team) (*models.Team, error) {
	saved, err := upsertTeam(ctx, r.db, team)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// UpsertMany creates or updates multiple teams in one transaction.
func (r *teamRepo) UpsertMany(ctx context.Context, teams []models.Team) ([]models.Team, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	saved := make([]models.Team, 0, len(teams))
	for _, team := range teams {
		next, err := upsertTeam(ctx, tx, team)
		if err != nil {
			return nil, err
		}
		saved = append(saved, next)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return saved, nil
}

// ReplaceAll clears the current roster and inserts the provided teams in one transaction.
func (r *teamRepo) ReplaceAll(ctx context.Context, teams []models.Team) ([]models.Team, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM teams"); err != nil {
		return nil, err
	}

	saved := make([]models.Team, 0, len(teams))
	for _, team := range teams {
		next, err := upsertTeam(ctx, tx, team)
		if err != nil {
			return nil, err
		}
		saved = append(saved, next)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return saved, nil
}

// Delete removes one team by ID.
func (r *teamRepo) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM teams WHERE id = $1", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type teamGetter interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

func upsertTeam(ctx context.Context, getter teamGetter, team models.Team) (models.Team, error) {
	var saved models.Team
	err := getter.GetContext(ctx, &saved, `
		INSERT INTO teams (name, strength, attack_rating, defense_rating, form_rating)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (name) DO UPDATE SET
			strength = EXCLUDED.strength,
			attack_rating = EXCLUDED.attack_rating,
			defense_rating = EXCLUDED.defense_rating,
			form_rating = EXCLUDED.form_rating,
			updated_at = NOW()
		RETURNING *
	`, team.Name, team.Strength, team.AttackRating, team.DefenseRating, team.FormRating)
	return saved, err
}
