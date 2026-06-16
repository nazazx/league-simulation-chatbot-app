package models

// LEARNING NOTE: Bu dosya uygulamanın veri yapılarını tutar.
// Go'da struct, TypeScript interface veya Java class gibi alanlardan oluşan veri tipidir.
// Handler, service ve repository katmanları aynı modelleri kullanarak veri taşır.

import "time"

// --- Base Model (struct composition) ---

// BaseModel contains common fields shared by all database entities.
// LEARNING NOTE: Struct composition, ortak alanları tekrar yazmadan başka struct'lara gömmeyi sağlar.
// Team ve Match içine BaseModel yazınca ID/CreatedAt/UpdatedAt alanları o struct'lara dahil olur.
type BaseModel struct {
	// LEARNING NOTE: `json` API alan adını, `db` ise sqlx'in database kolon eşlemesini belirtir.
	// Örneğin ID Go'da büyük harfle export edilir ama JSON'da "id" olarak döner.
	ID        int       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// --- Domain Models ---

// Team represents a football team in the league.
type Team struct {
	// LEARNING NOTE: BaseModel burada gömülü alan olarak kullanılır; Team.ID şeklinde erişilebilir.
	BaseModel
	Name          string `json:"name" db:"name"`
	Strength      int    `json:"strength" db:"strength"`
	AttackRating  int    `json:"attack_rating" db:"attack_rating"`
	DefenseRating int    `json:"defense_rating" db:"defense_rating"`
	FormRating    int    `json:"form_rating" db:"form_rating"`
}

// Match represents a single match between two teams.
type Match struct {
	BaseModel
	Week       int `json:"week" db:"week"`
	HomeTeamID int `json:"home_team_id" db:"home_team_id"`
	AwayTeamID int `json:"away_team_id" db:"away_team_id"`
	// LEARNING NOTE: *int pointer sayesinde oynanmamış maçta skor nil olabilir; 0-0 ile karışmaz.
	// Go'da pointer, değerin kendisi yerine bellekteki adresini tutar; nil "değer yok" anlamına gelir.
	HomeScore *int       `json:"home_score" db:"home_score"`
	AwayScore *int       `json:"away_score" db:"away_score"`
	Played    bool       `json:"played" db:"played"`
	PlayedAt  *time.Time `json:"played_at" db:"played_at"`
}

// Standing represents a team's position in the league table.
// Calculated dynamically from played matches — not stored in DB.
// LEARNING NOTE: Standing ayrı tabloda saklanmaz; oynanan maçlardan dinamik olarak hesaplanan read model'dir.
// Bu model repository'deki GetStandings SQL sonucuyla doldurulur ve API'ye döner.
type Standing struct {
	TeamID         int    `json:"team_id" db:"team_id"`
	TeamName       string `json:"team_name" db:"team_name"`
	Played         int    `json:"played" db:"played"`
	Wins           int    `json:"wins" db:"wins"`
	Draws          int    `json:"draws" db:"draws"`
	Losses         int    `json:"losses" db:"losses"`
	GoalsFor       int    `json:"goals_for" db:"goals_for"`
	GoalsAgainst   int    `json:"goals_against" db:"goals_against"`
	GoalDifference int    `json:"goal_difference" db:"goal_difference"`
	Points         int    `json:"points" db:"points"`
}

// HistoricalTeamStats summarizes seeded historical matches for prediction priors.
type HistoricalTeamStats struct {
	// LEARNING NOTE: float64 ondalıklı sayı tipidir; win rate ve points per match gibi metrikler tam sayı değildir.
	TeamID               int     `json:"team_id" db:"team_id"`
	MatchesPlayed        int     `json:"matches_played" db:"matches_played"`
	WinRate              float64 `json:"win_rate" db:"win_rate"`
	PointsPerMatch       float64 `json:"points_per_match" db:"points_per_match"`
	GoalsForPerMatch     float64 `json:"goals_for_per_match" db:"goals_for_per_match"`
	GoalsAgainstPerMatch float64 `json:"goals_against_per_match" db:"goals_against_per_match"`
	HistoricalRating     float64 `json:"historical_rating" db:"historical_rating"`
}

// --- Response DTOs ---
// LEARNING NOTE: DTO'lar API'ye özel veri şekilleridir; her zaman DB tablosuyla birebir aynı olmaz.
// Örneğin MatchResponse takım ID yerine takım adını döndürebilir; frontend için daha okunur olur.

// HistoricalMatchResponse is the API representation of a seeded historical match.
type HistoricalMatchResponse struct {
	ID        int    `json:"id" db:"id"`
	Season    string `json:"season" db:"season"`
	Week      int    `json:"week" db:"week"`
	HomeTeam  string `json:"home_team" db:"home_team"`
	AwayTeam  string `json:"away_team" db:"away_team"`
	HomeScore int    `json:"home_score" db:"home_score"`
	AwayScore int    `json:"away_score" db:"away_score"`
}

// MatchResponse is the API representation of a match (includes team names).
type MatchResponse struct {
	ID        int    `json:"id"`
	Week      int    `json:"week"`
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeScore *int   `json:"home_score"`
	AwayScore *int   `json:"away_score"`
	Played    bool   `json:"played"`
}

// PlayWeekResult is returned after playing a week.
type PlayWeekResult struct {
	// LEARNING NOTE: []MatchResponse bir slice'tır; JSON'da array olarak döner.
	Week      int             `json:"week"`
	Matches   []MatchResponse `json:"matches"`
	Standings []Standing      `json:"standings"`
}

// PlayAllResult is returned after playing every remaining week.
type PlayAllResult struct {
	Weeks     []PlayWeekResult `json:"weeks"`
	Standings []Standing       `json:"standings"`
}

// EditMatchResult is returned after manually editing one match score.
type EditMatchResult struct {
	Match     MatchResponse `json:"match"`
	Standings []Standing    `json:"standings"`
}

// PredictionResponse contains aggregated Monte Carlo prediction results.
type PredictionResponse struct {
	// LEARNING NOTE: PredictionResponse, Monte Carlo simülasyonlarının aggregate edilmiş sonucudur.
	// Tek tek simüle edilen maçları değil, özet olasılıkları taşır.
	CurrentWeek        int              `json:"current_week"`
	AvailableAfterWeek int              `json:"available_after_week"`
	Simulations        int              `json:"simulations"`
	Predictions        []TeamPrediction `json:"predictions"`
}

// TeamPrediction represents one team's final table forecast.
type TeamPrediction struct {
	TeamID                  int     `json:"team_id"`
	TeamName                string  `json:"team_name"`
	CurrentPoints           int     `json:"current_points"`
	PredictedPoints         float64 `json:"predicted_points"`
	AverageRank             float64 `json:"average_rank"`
	ChampionshipProbability float64 `json:"championship_probability"`
	TopTwoProbability       float64 `json:"top_two_probability"`
}

// AgentChatMessage represents one message sent to or returned from the league agent.
type AgentChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AgentChatResponse is returned by the chatbot-style league analyst.
type AgentChatResponse struct {
	// LEARNING NOTE: `omitempty`, alan boşsa JSON response'ta hiç gönderilmemesini sağlar.
	Answer    string   `json:"answer"`
	Source    string   `json:"source"`
	Model     string   `json:"model,omitempty"`
	ToolCalls []string `json:"tool_calls,omitempty"`
}

// APIResponse is the standard JSON envelope for all responses.
// LEARNING NOTE: Envelope, tüm response'ların success/message/data dış yapısıyla tutarlı dönmesini sağlar.
type APIResponse struct {
	// LEARNING NOTE: Data interface{} olduğu için farklı endpoint'ler farklı tiplerde veri koyabilir.
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
