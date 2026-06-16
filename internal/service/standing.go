package service

import (
	"context"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/nazazx/league-simulation/internal/repository"
)

// standingService implements StandingService.
// LEARNING NOTE: Bu ince servis, handler'ın doğrudan repository'ye bağımlı olmasını azaltır.
type standingService struct {
	matchRepo repository.MatchRepository
}

// NewStandingService creates a new StandingService.
func NewStandingService(matchRepo repository.MatchRepository) StandingService {
	return &standingService{matchRepo: matchRepo}
}

// GetStandings returns the current league standings.
func (s *standingService) GetStandings(ctx context.Context) ([]models.Standing, error) {
	// LEARNING NOTE: Puan tablosunun asıl hesaplaması repository içindeki SQL query ile yapılır.
	return s.matchRepo.GetStandings(ctx)
}
