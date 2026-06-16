package repository

import (
	"context"

	"github.com/nazazx/league-simulation/internal/models"
)

// TeamRepository defines the data access methods for teams.
// LEARNING NOTE: Repository interface'i servis katmanını SQL detaylarından ayırır.
type TeamRepository interface {
	GetAll(ctx context.Context) ([]models.Team, error)
	Count(ctx context.Context) (int, error)
	Upsert(ctx context.Context, team models.Team) (*models.Team, error)
	UpsertMany(ctx context.Context, teams []models.Team) ([]models.Team, error)
	ReplaceAll(ctx context.Context, teams []models.Team) ([]models.Team, error)
	Delete(ctx context.Context, id int) error
}

// MatchUpdate holds a single match result to be applied in a batch transaction.
// LEARNING NOTE: Bu küçük DTO, toplu maç güncellemesinde repository'ye taşınan veriyi temsil eder.
type MatchUpdate struct {
	ID        int
	HomeScore int
	AwayScore int
}

// MatchRepository defines the data access methods for matches.
// LEARNING NOTE: MatchRepository, maçlar ve standings için tüm database operasyonlarının kontratıdır.
type MatchRepository interface {
	GetAll(ctx context.Context) ([]models.Match, error)
	GetByID(ctx context.Context, id int) (*models.Match, error)
	GetByWeek(ctx context.Context, week int) ([]models.Match, error)
	CreateMany(ctx context.Context, matches []models.Match) error
	GetNextUnplayedWeek(ctx context.Context) (int, error)
	UpdateMatchResult(ctx context.Context, id int, homeScore, awayScore int) error
	UpdateMatchResults(ctx context.Context, updates []MatchUpdate) error
	DeleteAll(ctx context.Context) error
	Exists(ctx context.Context) (bool, error)
	GetStandings(ctx context.Context) ([]models.Standing, error)
	GetHistoricalTeamStats(ctx context.Context) ([]models.HistoricalTeamStats, error)
	GetHistoricalMatches(ctx context.Context, search string, season string) ([]models.HistoricalMatchResponse, error)
}
