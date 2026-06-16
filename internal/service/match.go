package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/nazazx/league-simulation/internal/repository"
)

// matchService implements MatchService.
// LEARNING NOTE: Match service, haftayı oynatma, sezonu bitirme ve maç sonucunu güncelleme iş kurallarını taşır.
type matchService struct {
	teamRepo  repository.TeamRepository
	matchRepo repository.MatchRepository
	simulator MatchSimulator
}

// NewMatchService creates a new MatchService.
func NewMatchService(
	teamRepo repository.TeamRepository,
	matchRepo repository.MatchRepository,
	simulator MatchSimulator,
) MatchService {
	return &matchService{
		teamRepo:  teamRepo,
		matchRepo: matchRepo,
		simulator: simulator,
	}
}

// PlayNextWeek simulates all matches in the next unplayed week.
// Returns the played matches with scores and the updated standings.
func (s *matchService) PlayNextWeek(ctx context.Context) (*models.PlayWeekResult, error) {
	// Find the next unplayed week
	week, err := s.matchRepo.GetNextUnplayedWeek(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get next week: %w", err)
	}
	if week == 0 {
		return nil, ErrAllMatchesPlayed
	}

	return s.playWeek(ctx, week)
}

// PlayAllRemaining simulates every unplayed week and returns grouped results.
func (s *matchService) PlayAllRemaining(ctx context.Context) (*models.PlayAllResult, error) {
	var weeks []models.PlayWeekResult

	// LEARNING NOTE: Döngü, oynanmamış hafta kalmayana kadar devam eder; week 0 gelince biter.
	for {
		week, err := s.matchRepo.GetNextUnplayedWeek(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get next week: %w", err)
		}
		if week == 0 {
			break
		}

		result, err := s.playWeek(ctx, week)
		if err != nil {
			return nil, err
		}
		weeks = append(weeks, *result)
	}

	if len(weeks) == 0 {
		return nil, ErrAllMatchesPlayed
	}

	standings, err := s.matchRepo.GetStandings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}

	return &models.PlayAllResult{
		Weeks:     weeks,
		Standings: standings,
	}, nil
}

// UpdateMatchResult manually edits a match result and returns recalculated standings.
func (s *matchService) UpdateMatchResult(ctx context.Context, matchID int, homeScore int, awayScore int) (*models.EditMatchResult, error) {
	// LEARNING NOTE: Validation service katmanında yapılır; aynı kural farklı endpoint'lerde de korunur.
	if homeScore < 0 || awayScore < 0 {
		return nil, ErrInvalidScore
	}

	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("failed to get match: %w", err)
	}

	teams, err := s.getTeamMap(ctx)
	if err != nil {
		return nil, err
	}

	home, homeOK := teams[match.HomeTeamID]
	away, awayOK := teams[match.AwayTeamID]
	if !homeOK || !awayOK {
		return nil, fmt.Errorf("%w: match %d", ErrUnknownTeam, match.ID)
	}

	if err := s.matchRepo.UpdateMatchResult(ctx, matchID, homeScore, awayScore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("failed to update match %d: %w", matchID, err)
	}

	standings, err := s.matchRepo.GetStandings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}

	return &models.EditMatchResult{
		Match: models.MatchResponse{
			ID:        match.ID,
			Week:      match.Week,
			HomeTeam:  home.Name,
			AwayTeam:  away.Name,
			HomeScore: &homeScore,
			AwayScore: &awayScore,
			Played:    true,
		},
		Standings: standings,
	}, nil
}

func (s *matchService) playWeek(ctx context.Context, week int) (*models.PlayWeekResult, error) {
	// LEARNING NOTE: Küçük harfle başlayan metot private helper'dır; sadece service paketi içinden kullanılır.
	teams, err := s.getTeamMap(ctx)
	if err != nil {
		return nil, err
	}

	// Get matches for this week
	weekMatches, err := s.matchRepo.GetByWeek(ctx, week)
	if err != nil {
		return nil, fmt.Errorf("failed to get week %d matches: %w", week, err)
	}

	// Simulate all unplayed matches first, collecting results in memory
	// LEARNING NOTE: Sonuçlar önce memory'de hazırlanır, sonra repository tek transaction ile DB'ye yazar.
	var updates []repository.MatchUpdate
	var matchResponses []models.MatchResponse
	for _, m := range weekMatches {
		if m.Played {
			continue
		}

		home, homeOK := teams[m.HomeTeamID]
		away, awayOK := teams[m.AwayTeamID]
		if !homeOK || !awayOK {
			return nil, fmt.Errorf("%w: match %d", ErrUnknownTeam, m.ID)
		}

		homeScore, awayScore := s.simulator.Simulate(home, away)

		updates = append(updates, repository.MatchUpdate{
			ID:        m.ID,
			HomeScore: homeScore,
			AwayScore: awayScore,
		})

		matchResponses = append(matchResponses, models.MatchResponse{
			ID:        m.ID,
			Week:      m.Week,
			HomeTeam:  home.Name,
			AwayTeam:  away.Name,
			HomeScore: &homeScore,
			AwayScore: &awayScore,
			Played:    true,
		})
	}

	// Commit all match results in a single transaction
	if err := s.matchRepo.UpdateMatchResults(ctx, updates); err != nil {
		return nil, fmt.Errorf("failed to save week %d results: %w", week, err)
	}

	standings, err := s.matchRepo.GetStandings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}

	return &models.PlayWeekResult{
		Week:      week,
		Matches:   matchResponses,
		Standings: standings,
	}, nil
}

func (s *matchService) getTeamMap(ctx context.Context) (map[int]models.Team, error) {
	// LEARNING NOTE: map[int]Team, takım ID'sinden takıma hızlı erişim sağlar.
	teams, err := s.teamRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}
	teamMap := make(map[int]models.Team)
	for _, t := range teams {
		teamMap[t.ID] = t
	}
	return teamMap, nil
}
