package service

import (
	"context"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/nazazx/league-simulation/internal/repository"
)

// mockTeamRepo is an in-memory TeamRepository for unit tests.
type mockTeamRepo struct {
	teams []models.Team
	err   error
}

func (m *mockTeamRepo) GetAll(_ context.Context) ([]models.Team, error) {
	return m.teams, m.err
}

func (m *mockTeamRepo) Count(_ context.Context) (int, error) {
	return len(m.teams), m.err
}

func (m *mockTeamRepo) Upsert(_ context.Context, team models.Team) (*models.Team, error) {
	m.teams = append(m.teams, team)
	return &team, m.err
}

func (m *mockTeamRepo) UpsertMany(_ context.Context, teams []models.Team) ([]models.Team, error) {
	m.teams = append(m.teams, teams...)
	return teams, m.err
}

func (m *mockTeamRepo) ReplaceAll(_ context.Context, teams []models.Team) ([]models.Team, error) {
	m.teams = teams
	return teams, m.err
}

func (m *mockTeamRepo) Delete(_ context.Context, id int) error {
	for i, team := range m.teams {
		if team.ID == id {
			m.teams = append(m.teams[:i], m.teams[i+1:]...)
			return m.err
		}
	}
	return m.err
}

// mockMatchRepo is an in-memory MatchRepository for unit tests.
type mockMatchRepo struct {
	matches         []models.Match
	standings       []models.Standing
	historicalStats []models.HistoricalTeamStats
	historicalMatch []models.HistoricalMatchResponse
	nextWeek        int
	exists          bool

	// Track calls for verification
	createdMatches []models.Match
	updatedResults []repository.MatchUpdate
	deleteCalled   bool

	err error
}

func (m *mockMatchRepo) GetAll(_ context.Context) ([]models.Match, error) {
	return m.matches, m.err
}

func (m *mockMatchRepo) GetByID(_ context.Context, id int) (*models.Match, error) {
	for _, match := range m.matches {
		if match.ID == id {
			return &match, nil
		}
	}
	return nil, m.err
}

func (m *mockMatchRepo) GetByWeek(_ context.Context, week int) ([]models.Match, error) {
	var result []models.Match
	for _, match := range m.matches {
		if match.Week == week {
			result = append(result, match)
		}
	}
	return result, m.err
}

func (m *mockMatchRepo) CreateMany(_ context.Context, matches []models.Match) error {
	m.createdMatches = matches
	m.matches = append(m.matches, matches...)
	m.exists = true
	return m.err
}

func (m *mockMatchRepo) GetNextUnplayedWeek(_ context.Context) (int, error) {
	return m.nextWeek, m.err
}

func (m *mockMatchRepo) UpdateMatchResult(_ context.Context, id int, homeScore, awayScore int) error {
	for i := range m.matches {
		if m.matches[i].ID == id {
			m.matches[i].HomeScore = &homeScore
			m.matches[i].AwayScore = &awayScore
			m.matches[i].Played = true
			return nil
		}
	}
	return m.err
}

func (m *mockMatchRepo) UpdateMatchResults(_ context.Context, updates []repository.MatchUpdate) error {
	m.updatedResults = updates
	for _, u := range updates {
		for i := range m.matches {
			if m.matches[i].ID == u.ID {
				m.matches[i].HomeScore = &u.HomeScore
				m.matches[i].AwayScore = &u.AwayScore
				m.matches[i].Played = true
			}
		}
	}
	return m.err
}

func (m *mockMatchRepo) DeleteAll(_ context.Context) error {
	m.deleteCalled = true
	m.matches = nil
	m.exists = false
	return m.err
}

func (m *mockMatchRepo) Exists(_ context.Context) (bool, error) {
	return m.exists, m.err
}

func (m *mockMatchRepo) GetStandings(_ context.Context) ([]models.Standing, error) {
	return m.standings, m.err
}

func (m *mockMatchRepo) GetHistoricalTeamStats(_ context.Context) ([]models.HistoricalTeamStats, error) {
	return m.historicalStats, m.err
}

func (m *mockMatchRepo) GetHistoricalMatches(_ context.Context, _ string, _ string) ([]models.HistoricalMatchResponse, error) {
	return m.historicalMatch, m.err
}

// mockSimulator returns fixed scores for predictable tests.
type mockSimulator struct {
	homeScore int
	awayScore int
}

func (m *mockSimulator) Simulate(_, _ models.Team) (int, int) {
	return m.homeScore, m.awayScore
}

// --- Test helpers ---

func testTeams() []models.Team {
	return []models.Team{
		{BaseModel: models.BaseModel{ID: 1}, Name: "Bosphorus United", Strength: 74, AttackRating: 78, DefenseRating: 70, FormRating: 76},
		{BaseModel: models.BaseModel{ID: 2}, Name: "Anka FC", Strength: 69, AttackRating: 72, DefenseRating: 67, FormRating: 71},
		{BaseModel: models.BaseModel{ID: 3}, Name: "Galata Rovers", Strength: 81, AttackRating: 84, DefenseRating: 79, FormRating: 80},
		{BaseModel: models.BaseModel{ID: 4}, Name: "Moda Athletic", Strength: 66, AttackRating: 68, DefenseRating: 65, FormRating: 69},
	}
}

func testStandings() []models.Standing {
	return []models.Standing{
		{TeamID: 1, TeamName: "Bosphorus United", Played: 2, Wins: 2, Draws: 0, Losses: 0, GoalsFor: 5, GoalsAgainst: 1, GoalDifference: 4, Points: 6},
		{TeamID: 2, TeamName: "Anka FC", Played: 2, Wins: 1, Draws: 1, Losses: 0, GoalsFor: 3, GoalsAgainst: 2, GoalDifference: 1, Points: 4},
		{TeamID: 3, TeamName: "Galata Rovers", Played: 2, Wins: 1, Draws: 0, Losses: 1, GoalsFor: 3, GoalsAgainst: 3, GoalDifference: 0, Points: 3},
		{TeamID: 4, TeamName: "Moda Athletic", Played: 2, Wins: 0, Draws: 1, Losses: 1, GoalsFor: 1, GoalsAgainst: 6, GoalDifference: -5, Points: 1},
	}
}

func testHistoricalStats() []models.HistoricalTeamStats {
	return []models.HistoricalTeamStats{
		{TeamID: 1, MatchesPlayed: 12, WinRate: 0.67, PointsPerMatch: 2.25, GoalsForPerMatch: 2.5, GoalsAgainstPerMatch: 1.08, HistoricalRating: 74.5},
		{TeamID: 2, MatchesPlayed: 12, WinRate: 0.58, PointsPerMatch: 2.0, GoalsForPerMatch: 2.0, GoalsAgainstPerMatch: 1.25, HistoricalRating: 68.0},
		{TeamID: 3, MatchesPlayed: 12, WinRate: 0.42, PointsPerMatch: 1.5, GoalsForPerMatch: 1.75, GoalsAgainstPerMatch: 1.67, HistoricalRating: 58.0},
		{TeamID: 4, MatchesPlayed: 12, WinRate: 0.17, PointsPerMatch: 0.75, GoalsForPerMatch: 0.83, GoalsAgainstPerMatch: 2.33, HistoricalRating: 38.0},
	}
}
