package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/nazazx/league-simulation/internal/models"
)

func TestPrediction_DefaultSimulations(t *testing.T) {
	teams := testTeams()
	matches := buildPlayedMatches(teams, 4)
	teamRepo := &mockTeamRepo{teams: teams}
	matchRepo := &mockMatchRepo{
		matches:         matches,
		standings:       testStandings(),
		historicalStats: testHistoricalStats(),
	}
	sim := NewSimulator()
	svc := NewPredictionService(teamRepo, matchRepo, sim)

	result, err := svc.Predict(context.Background(), 0) // 0 → default 1000
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Simulations != defaultPredictionSimulations {
		t.Errorf("expected %d simulations, got %d", defaultPredictionSimulations, result.Simulations)
	}
}

func TestPrediction_MaxSimulations(t *testing.T) {
	teams := testTeams()
	matches := buildPlayedMatches(teams, 4)
	teamRepo := &mockTeamRepo{teams: teams}
	matchRepo := &mockMatchRepo{
		matches:         matches,
		standings:       testStandings(),
		historicalStats: testHistoricalStats(),
	}
	sim := NewSimulator()
	svc := NewPredictionService(teamRepo, matchRepo, sim)

	result, err := svc.Predict(context.Background(), 999999)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Simulations != maxPredictionSimulations {
		t.Errorf("expected %d max simulations, got %d", maxPredictionSimulations, result.Simulations)
	}
}

func TestPrediction_ProbabilitiesSumTo100(t *testing.T) {
	teams := testTeams()
	matches := buildPlayedMatches(teams, 4)
	teamRepo := &mockTeamRepo{teams: teams}
	matchRepo := &mockMatchRepo{
		matches:         matches,
		standings:       testStandings(),
		historicalStats: testHistoricalStats(),
	}
	sim := NewSimulator()
	svc := NewPredictionService(teamRepo, matchRepo, sim)

	result, err := svc.Predict(context.Background(), 1000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	totalChampionship := 0.0
	for _, p := range result.Predictions {
		totalChampionship += p.ChampionshipProbability
	}

	// Allow small floating-point rounding error
	if math.Abs(totalChampionship-100.0) > 1.0 {
		t.Errorf("championship probabilities sum to %.2f, expected ~100", totalChampionship)
	}
}

func TestPrediction_NoFixtures(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{matches: nil}
	sim := NewSimulator()
	svc := NewPredictionService(teamRepo, matchRepo, sim)

	_, err := svc.Predict(context.Background(), 100)
	if !errors.Is(err, ErrFixturesNotGenerated) {
		t.Errorf("expected ErrFixturesNotGenerated, got %v", err)
	}
}

func TestPrediction_NoTeams(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: nil}
	matchRepo := &mockMatchRepo{}
	sim := NewSimulator()
	svc := NewPredictionService(teamRepo, matchRepo, sim)

	_, err := svc.Predict(context.Background(), 100)
	if !errors.Is(err, ErrNoTeams) {
		t.Errorf("expected ErrNoTeams, got %v", err)
	}
}

func TestPrediction_AllTeamsPresent(t *testing.T) {
	teams := testTeams()
	matches := buildPlayedMatches(teams, 4)
	teamRepo := &mockTeamRepo{teams: teams}
	matchRepo := &mockMatchRepo{
		matches:         matches,
		standings:       testStandings(),
		historicalStats: testHistoricalStats(),
	}
	sim := NewSimulator()
	svc := NewPredictionService(teamRepo, matchRepo, sim)

	result, err := svc.Predict(context.Background(), 500)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Predictions) != len(teams) {
		t.Errorf("expected %d predictions, got %d", len(teams), len(result.Predictions))
	}

	teamNames := map[string]bool{}
	for _, p := range result.Predictions {
		teamNames[p.TeamName] = true
	}
	for _, team := range teams {
		if !teamNames[team.Name] {
			t.Errorf("missing prediction for team %s", team.Name)
		}
	}
}

// buildPlayedMatches creates a set of played matches for testing predictions.
func buildPlayedMatches(teams []models.Team, playedWeeks int) []models.Match {
	var matches []models.Match
	id := 1
	for week := 1; week <= 12; week++ {
		for i := 0; i < len(teams); i += 2 {
			if i+1 >= len(teams) {
				break
			}
			played := week <= playedWeeks
			m := models.Match{
				BaseModel:  models.BaseModel{ID: id},
				Week:       week,
				HomeTeamID: teams[i].ID,
				AwayTeamID: teams[i+1].ID,
				Played:     played,
			}
			if played {
				hs, as := 2, 1
				m.HomeScore = &hs
				m.AwayScore = &as
			}
			matches = append(matches, m)
			id++
		}
	}
	return matches
}
