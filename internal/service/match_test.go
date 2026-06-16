package service

import (
	"context"
	"errors"
	"testing"

	"github.com/nazazx/league-simulation/internal/models"
)

func TestMatchService_PlayNextWeek_AllPlayed(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{nextWeek: 0} // 0 means all played
	sim := &mockSimulator{homeScore: 2, awayScore: 1}
	svc := NewMatchService(teamRepo, matchRepo, sim)

	_, err := svc.PlayNextWeek(context.Background())
	if !errors.Is(err, ErrAllMatchesPlayed) {
		t.Errorf("expected ErrAllMatchesPlayed, got %v", err)
	}
}

func TestMatchService_PlayNextWeek_Success(t *testing.T) {
	teams := testTeams()
	matches := []models.Match{
		{BaseModel: models.BaseModel{ID: 1}, Week: 1, HomeTeamID: 1, AwayTeamID: 4, Played: false},
		{BaseModel: models.BaseModel{ID: 2}, Week: 1, HomeTeamID: 2, AwayTeamID: 3, Played: false},
	}
	teamRepo := &mockTeamRepo{teams: teams}
	matchRepo := &mockMatchRepo{
		matches:   matches,
		nextWeek:  1,
		standings: testStandings(),
	}
	sim := &mockSimulator{homeScore: 2, awayScore: 1}
	svc := NewMatchService(teamRepo, matchRepo, sim)

	result, err := svc.PlayNextWeek(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Week != 1 {
		t.Errorf("expected week 1, got %d", result.Week)
	}
	if len(result.Matches) != 2 {
		t.Errorf("expected 2 match results, got %d", len(result.Matches))
	}

	// Verify batch transaction was used
	if len(matchRepo.updatedResults) != 2 {
		t.Errorf("expected 2 batch updates, got %d", len(matchRepo.updatedResults))
	}

	// Verify scores from mock simulator
	for _, mr := range result.Matches {
		if *mr.HomeScore != 2 || *mr.AwayScore != 1 {
			t.Errorf("expected score 2-1, got %d-%d", *mr.HomeScore, *mr.AwayScore)
		}
	}
}

func TestMatchService_PlayAllRemaining_AllPlayed(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{nextWeek: 0}
	sim := &mockSimulator{homeScore: 1, awayScore: 0}
	svc := NewMatchService(teamRepo, matchRepo, sim)

	_, err := svc.PlayAllRemaining(context.Background())
	if !errors.Is(err, ErrAllMatchesPlayed) {
		t.Errorf("expected ErrAllMatchesPlayed, got %v", err)
	}
}

func TestMatchService_UpdateMatchResult_NegativeScore(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{}
	sim := &mockSimulator{}
	svc := NewMatchService(teamRepo, matchRepo, sim)

	_, err := svc.UpdateMatchResult(context.Background(), 1, -1, 0)
	if !errors.Is(err, ErrInvalidScore) {
		t.Errorf("expected ErrInvalidScore, got %v", err)
	}

	_, err = svc.UpdateMatchResult(context.Background(), 1, 0, -5)
	if !errors.Is(err, ErrInvalidScore) {
		t.Errorf("expected ErrInvalidScore, got %v", err)
	}
}

func TestMatchService_UpdateMatchResult_Success(t *testing.T) {
	teams := testTeams()
	hs, as := 3, 1
	matches := []models.Match{
		{BaseModel: models.BaseModel{ID: 10}, Week: 1, HomeTeamID: 1, AwayTeamID: 2, HomeScore: &hs, AwayScore: &as, Played: true},
	}
	teamRepo := &mockTeamRepo{teams: teams}
	matchRepo := &mockMatchRepo{
		matches:   matches,
		standings: testStandings(),
	}
	sim := &mockSimulator{}
	svc := NewMatchService(teamRepo, matchRepo, sim)

	result, err := svc.UpdateMatchResult(context.Background(), 10, 0, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Match.ID != 10 {
		t.Errorf("expected match ID 10, got %d", result.Match.ID)
	}
	if *result.Match.HomeScore != 0 || *result.Match.AwayScore != 0 {
		t.Errorf("expected score 0-0, got %d-%d", *result.Match.HomeScore, *result.Match.AwayScore)
	}
}
