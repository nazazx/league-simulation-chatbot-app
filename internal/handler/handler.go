package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nazazx/league-simulation/internal/models"
	"github.com/nazazx/league-simulation/internal/repository"
	"github.com/nazazx/league-simulation/internal/service"
	"github.com/jmoiron/sqlx"
)

// Handler holds all HTTP handler dependencies.
// LEARNING NOTE: Handler struct'ı HTTP katmanının ihtiyaç duyduğu repository ve service bağımlılıklarını taşır.
// Individual handler methods are split across domain-specific files:
//   - team_handler.go      → HealthCheck, GetTeams
//   - match_handler.go     → GetMatches, GetMatchesByWeek, UpdateMatchResult, PlayNextWeek, PlayAllRemaining, GetHistoricalMatches, GetHistoricalStats
//   - fixture_handler.go   → GenerateFixtures, Reset
//   - prediction_handler.go → GetStandings, GetPredictions, GetInsights, AgentChat
type Handler struct {
	// LEARNING NOTE: Interface tipleri kullanmak test etmeyi ve implementasyon değiştirmeyi kolaylaştırır.
	db            *sqlx.DB
	teamRepo      repository.TeamRepository
	matchRepo     repository.MatchRepository
	fixtureSvc    service.FixtureService
	matchSvc      service.MatchService
	standingSvc   service.StandingService
	predictionSvc service.PredictionService
	insightSvc    service.InsightService
}

// NewHandler creates a new Handler with all dependencies.
// LEARNING NOTE: Constructor fonksiyonu Handler'ı hazırlar; main.go bağımlılıkları üretip buraya verir.
func NewHandler(
	db *sqlx.DB,
	teamRepo repository.TeamRepository,
	matchRepo repository.MatchRepository,
	fixtureSvc service.FixtureService,
	matchSvc service.MatchService,
	standingSvc service.StandingService,
	predictionSvc service.PredictionService,
	insightSvc service.InsightService,
) *Handler {
	return &Handler{
		db:            db,
		teamRepo:      teamRepo,
		matchRepo:     matchRepo,
		fixtureSvc:    fixtureSvc,
		matchSvc:      matchSvc,
		standingSvc:   standingSvc,
		predictionSvc: predictionSvc,
		insightSvc:    insightSvc,
	}
}

// --- Shared helpers ---

// respond writes a standardised JSON envelope to the client.
func respond(c *gin.Context, status int, success bool, message string, data interface{}) {
	// LEARNING NOTE: Tüm API cevapları aynı success/message/data formatından döner; frontend için tutarlı olur.
	c.JSON(status, models.APIResponse{
		Success: success,
		Message: message,
		Data:    data,
	})
}

// mapServiceError returns the appropriate HTTP status for a known service error.
func mapServiceError(err error, mapping map[error]int) int {
	for target, status := range mapping {
		if err == target {
			return status
		}
	}
	return http.StatusInternalServerError
}

// --- Request DTOs ---

type updateMatchRequest struct {
	// LEARNING NOTE: Struct tag'leri JSON alan adını ve Gin validation kurallarını belirtir.
	HomeScore *int `json:"home_score" binding:"required,min=0"`
	AwayScore *int `json:"away_score" binding:"required,min=0"`
}

type teamRequest struct {
	Name          string `json:"name"`
	Strength      *int   `json:"strength"`
	AttackRating  *int   `json:"attack_rating"`
	DefenseRating *int   `json:"defense_rating"`
	FormRating    *int   `json:"form_rating"`
}

type importTeamsRequest struct {
	Teams   []teamRequest `json:"teams"`
	Replace bool          `json:"replace"`
}

type agentChatRequest struct {
	// LEARNING NOTE: DTO, HTTP request/response için kullanılan taşıma modelidir; database modeli olmak zorunda değildir.
	Message string                    `json:"message" binding:"required"`
	History []models.AgentChatMessage `json:"history"`
}
