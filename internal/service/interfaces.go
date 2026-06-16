package service

import (
	"context"

	"github.com/nazazx/league-simulation/internal/models"
)

// FixtureService handles fixture generation.
// LEARNING NOTE: Interface, bir servisin dışarıya hangi davranışları sunduğunu tanımlar; implementasyon detayını gizler.
type FixtureService interface {
	Generate(ctx context.Context) error
}

// MatchService handles match simulation and week progression.
// LEARNING NOTE: Handler bu kontrata bağımlıdır; alttaki gerçek struct değişirse handler değişmek zorunda kalmaz.
type MatchService interface {
	PlayNextWeek(ctx context.Context) (*models.PlayWeekResult, error)
	PlayAllRemaining(ctx context.Context) (*models.PlayAllResult, error)
	UpdateMatchResult(ctx context.Context, matchID int, homeScore int, awayScore int) (*models.EditMatchResult, error)
}

// StandingService handles league table calculation.
type StandingService interface {
	GetStandings(ctx context.Context) ([]models.Standing, error)
}

// PredictionService estimates the final table without mutating real matches.
type PredictionService interface {
	Predict(ctx context.Context, simulations int) (*models.PredictionResponse, error)
}

// InsightService provides AI-powered league analysis via chat.
type InsightService interface {
	Chat(ctx context.Context, message string, history []models.AgentChatMessage) (*models.AgentChatResponse, error)
}

// MatchSimulator simulates the score of a single match.
// LEARNING NOTE: Küçük interface'ler okunabilirliği artırır; bu interface sadece skor simülasyonu sorumluluğunu taşır.
type MatchSimulator interface {
	Simulate(home, away models.Team) (homeScore, awayScore int)
}
