package service

import "errors"

// Sentinel errors for the service layer.
// Handler uses errors.Is() to map these to HTTP status codes.
// LEARNING NOTE: Sentinel error, farklı katmanlarda errors.Is ile güvenli karşılaştırılan sabit hata değeridir.

var (
	// ErrAllMatchesPlayed is returned when there are no unplayed weeks left.
	ErrAllMatchesPlayed = errors.New("all matches have been played")

	// ErrFixturesAlreadyExist is returned when a schedule is generated more than once.
	ErrFixturesAlreadyExist = errors.New("schedule already exists; clear the season first")

	// ErrFixturesNotGenerated is returned when an operation requires a schedule that hasn't been created.
	ErrFixturesNotGenerated = errors.New("schedule has not been built yet")

	// ErrMatchNotFound is returned when a match ID does not exist.
	ErrMatchNotFound = errors.New("match not found")

	// ErrInvalidScore is returned when match scores are negative.
	ErrInvalidScore = errors.New("scores cannot be negative")

	// ErrMinimumTeams is returned when fewer than 4 teams are available.
	ErrMinimumTeams = errors.New("minimum 4 teams required")

	// ErrEvenTeamsRequired is returned when the team count is odd.
	ErrEvenTeamsRequired = errors.New("even number of teams required")

	// ErrUnknownTeam is returned when a match references a team not in the database.
	ErrUnknownTeam = errors.New("match contains an unknown team")

	// ErrNoTeams is returned when the teams table is empty.
	ErrNoTeams = errors.New("no teams found")

	// ErrNoStandings is returned when standings cannot be computed.
	ErrNoStandings = errors.New("no standings found")

	// ErrNoPredictions is returned when predictions produce no results.
	ErrNoPredictions = errors.New("no predictions found")

	// ErrMessageRequired is returned when the chat message is empty.
	ErrMessageRequired = errors.New("message is required")
)
