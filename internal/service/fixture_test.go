package service

import (
	"context"
	"errors"
	"testing"
)

func TestFixtureGenerate_Success(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{exists: false}
	svc := NewFixtureService(teamRepo, matchRepo)

	err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 4 teams → quadruple round-robin → 12 weeks × 2 matches = 24 total
	if len(matchRepo.createdMatches) != 24 {
		t.Errorf("expected 24 matches, got %d", len(matchRepo.createdMatches))
	}
}

func TestFixtureGenerate_CorrectWeeks(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{exists: false}
	svc := NewFixtureService(teamRepo, matchRepo)

	err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	weekCount := map[int]int{}
	for _, m := range matchRepo.createdMatches {
		weekCount[m.Week]++
	}

	// 12 weeks expected, each with 2 matches
	if len(weekCount) != 12 {
		t.Errorf("expected 12 weeks, got %d", len(weekCount))
	}
	for week, count := range weekCount {
		if count != 2 {
			t.Errorf("week %d: expected 2 matches, got %d", week, count)
		}
	}
}

func TestFixtureGenerate_EachPairPlays4Times(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{exists: false}
	svc := NewFixtureService(teamRepo, matchRepo)

	err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	type pair struct{ a, b int }
	counts := map[pair]int{}
	for _, m := range matchRepo.createdMatches {
		a, b := m.HomeTeamID, m.AwayTeamID
		if a > b {
			a, b = b, a
		}
		counts[pair{a, b}]++
	}

	// 4 teams = 6 unique pairs, each should play 4 times
	if len(counts) != 6 {
		t.Errorf("expected 6 unique pairs, got %d", len(counts))
	}
	for p, c := range counts {
		if c != 4 {
			t.Errorf("pair (%d,%d) played %d times, expected 4", p.a, p.b, c)
		}
	}
}

func TestFixtureGenerate_NoSelfMatches(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{exists: false}
	svc := NewFixtureService(teamRepo, matchRepo)

	err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, m := range matchRepo.createdMatches {
		if m.HomeTeamID == m.AwayTeamID {
			t.Errorf("self-match found: team %d vs team %d", m.HomeTeamID, m.AwayTeamID)
		}
	}
}

func TestFixtureGenerate_BalancedHomeAway(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{exists: false}
	svc := NewFixtureService(teamRepo, matchRepo)

	err := svc.Generate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	homeGames := map[int]int{}
	awayGames := map[int]int{}
	for _, m := range matchRepo.createdMatches {
		homeGames[m.HomeTeamID]++
		awayGames[m.AwayTeamID]++
	}

	// Each team should play 6 home and 6 away (12 total)
	for _, team := range testTeams() {
		if homeGames[team.ID] != 6 {
			t.Errorf("team %s: expected 6 home games, got %d", team.Name, homeGames[team.ID])
		}
		if awayGames[team.ID] != 6 {
			t.Errorf("team %s: expected 6 away games, got %d", team.Name, awayGames[team.ID])
		}
	}
}

func TestFixtureGenerate_AlreadyExist(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()}
	matchRepo := &mockMatchRepo{exists: true}
	svc := NewFixtureService(teamRepo, matchRepo)

	err := svc.Generate(context.Background())
	if !errors.Is(err, ErrFixturesAlreadyExist) {
		t.Errorf("expected ErrFixturesAlreadyExist, got %v", err)
	}
}

func TestFixtureGenerate_TooFewTeams(t *testing.T) {
	teamRepo := &mockTeamRepo{teams: testTeams()[:2]}
	matchRepo := &mockMatchRepo{exists: false}
	svc := NewFixtureService(teamRepo, matchRepo)

	err := svc.Generate(context.Background())
	if !errors.Is(err, ErrMinimumTeams) {
		t.Errorf("expected ErrMinimumTeams, got %v", err)
	}
}
