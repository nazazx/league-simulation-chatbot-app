package service

import (
	"testing"

	"github.com/nazazx/league-simulation/internal/models"
)

func TestSimulator_ScoresInRange(t *testing.T) {
	sim := NewSimulator()
	teams := testTeams()

	for i := 0; i < 1000; i++ {
		for _, home := range teams {
			for _, away := range teams {
				if home.ID == away.ID {
					continue
				}
				hs, as := sim.Simulate(home, away)
				if hs < 0 || hs > 5 {
					t.Errorf("home score %d out of range 0-5 (home=%s, away=%s)", hs, home.Name, away.Name)
				}
				if as < 0 || as > 5 {
					t.Errorf("away score %d out of range 0-5 (home=%s, away=%s)", as, home.Name, away.Name)
				}
			}
		}
	}
}

func TestSimulator_StrongerTeamScoresMore(t *testing.T) {
	sim := NewSimulator()

	strong := models.Team{
		BaseModel:     models.BaseModel{ID: 1},
		Name:          "Strong",
		Strength:      95,
		AttackRating:  95,
		DefenseRating: 95,
		FormRating:    95,
	}
	weak := models.Team{
		BaseModel:     models.BaseModel{ID: 2},
		Name:          "Weak",
		Strength:      30,
		AttackRating:  30,
		DefenseRating: 30,
		FormRating:    30,
	}

	totalStrong := 0
	totalWeak := 0
	runs := 5000

	for i := 0; i < runs; i++ {
		hs, as := sim.Simulate(strong, weak)
		totalStrong += hs
		totalWeak += as
	}

	avgStrong := float64(totalStrong) / float64(runs)
	avgWeak := float64(totalWeak) / float64(runs)

	if avgStrong <= avgWeak {
		t.Errorf("expected stronger team to score more on average: strong=%.2f, weak=%.2f", avgStrong, avgWeak)
	}
}

func TestSimulator_HomeAdvantage(t *testing.T) {
	sim := NewSimulator()

	team := models.Team{
		BaseModel:     models.BaseModel{ID: 1},
		Name:          "Equal",
		Strength:      75,
		AttackRating:  75,
		DefenseRating: 75,
		FormRating:    75,
	}

	totalHome := 0
	totalAway := 0
	runs := 5000

	for i := 0; i < runs; i++ {
		hs, as := sim.Simulate(team, team)
		totalHome += hs
		totalAway += as
	}

	avgHome := float64(totalHome) / float64(runs)
	avgAway := float64(totalAway) / float64(runs)

	// Home team should score slightly more on average (home advantage = +0.25)
	if avgHome <= avgAway {
		t.Errorf("expected home advantage: home=%.2f, away=%.2f", avgHome, avgAway)
	}
}

func TestRatingOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		rating   int
		fallback int
		want     int
	}{
		{"positive rating", 80, 50, 80},
		{"zero rating with fallback", 0, 70, 70},
		{"zero rating zero fallback", 0, 0, 50},
		{"negative rating with fallback", -1, 60, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ratingOrDefault(tt.rating, tt.fallback)
			if got != tt.want {
				t.Errorf("ratingOrDefault(%d, %d) = %d, want %d", tt.rating, tt.fallback, got, tt.want)
			}
		})
	}
}
