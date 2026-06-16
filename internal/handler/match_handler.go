package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nazazx/league-simulation/internal/service"
)

// GetMatches returns all matches ordered by week.
// GET /matches
func (h *Handler) GetMatches(c *gin.Context) {
	// LEARNING NOTE: GET endpoint genelde veri okur; burada maç listesi döner, state değişmez.
	matches, err := h.matchRepo.GetAll(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to get matches", nil)
		return
	}

	respond(c, http.StatusOK, true, "Matches retrieved successfully", gin.H{
		"matches": matches,
	})
}

// GetMatchesByWeek returns matches for a specific week.
// GET /matches/week/:week
func (h *Handler) GetMatchesByWeek(c *gin.Context) {
	weekStr := c.Param("week")
	// LEARNING NOTE: URL parametreleri string gelir; sayı olarak kullanmak için strconv.Atoi ile çevrilir.
	week, err := strconv.Atoi(weekStr)
	if err != nil || week < 1 {
		respond(c, http.StatusBadRequest, false, "Invalid week number", nil)
		return
	}

	matches, err := h.matchRepo.GetByWeek(c.Request.Context(), week)
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to get matches", nil)
		return
	}

	respond(c, http.StatusOK, true, "Matches retrieved successfully", gin.H{
		"week":    week,
		"matches": matches,
	})
}

// GetHistoricalMatches returns seeded historical sample matches.
// GET /historical-matches?season=2023-2024&search=arsenal
func (h *Handler) GetHistoricalMatches(c *gin.Context) {
	matches, err := h.matchRepo.GetHistoricalMatches(
		c.Request.Context(),
		strings.TrimSpace(c.Query("search")),
		strings.TrimSpace(c.Query("season")),
	)
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to get historical matches", nil)
		return
	}

	respond(c, http.StatusOK, true, "Historical matches retrieved successfully", gin.H{
		"matches": matches,
	})
}

// GetHistoricalStats returns historical team stats used by prediction.
// GET /historical-stats
func (h *Handler) GetHistoricalStats(c *gin.Context) {
	stats, err := h.matchRepo.GetHistoricalTeamStats(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to get historical stats", nil)
		return
	}

	respond(c, http.StatusOK, true, "Historical stats retrieved successfully", gin.H{
		"stats": stats,
	})
}

// UpdateMatchResult manually edits a match score and recalculates standings.
// PUT /matches/:id
func (h *Handler) UpdateMatchResult(c *gin.Context) {
	matchID, err := strconv.Atoi(c.Param("id"))
	if err != nil || matchID < 1 {
		respond(c, http.StatusBadRequest, false, "Invalid match id", nil)
		return
	}

	var req updateMatchRequest
	// LEARNING NOTE: ShouldBindJSON, request body'deki JSON'u Go struct'ına çevirir ve validation yapar.
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	result, err := h.matchSvc.UpdateMatchResult(c.Request.Context(), matchID, *req.HomeScore, *req.AwayScore)
	if err != nil {
		// LEARNING NOTE: Service katmanındaki domain error'lar burada HTTP status code'a çevrilir.
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrMatchNotFound) || errors.Is(err, service.ErrInvalidScore) {
			status = http.StatusBadRequest
		}
		respond(c, status, false, err.Error(), nil)
		return
	}

	respond(c, http.StatusOK, true, "Match result updated successfully", result)
}

// PlayNextWeek plays the next unplayed week.
// POST /play/week
func (h *Handler) PlayNextWeek(c *gin.Context) {
	// LEARNING NOTE: POST endpoint state değiştirebilir; burada sıradaki hafta oynatılıp DB güncellenir.
	result, err := h.matchSvc.PlayNextWeek(c.Request.Context())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrAllMatchesPlayed) {
			status = http.StatusBadRequest
		}
		respond(c, status, false, err.Error(), nil)
		return
	}

	respond(c, http.StatusOK, true,
		"Week "+strconv.Itoa(result.Week)+" played successfully", result)
}

// PlayAllRemaining plays every remaining week.
// POST /play/all
func (h *Handler) PlayAllRemaining(c *gin.Context) {
	result, err := h.matchSvc.PlayAllRemaining(c.Request.Context())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrAllMatchesPlayed) {
			status = http.StatusBadRequest
		}
		respond(c, status, false, err.Error(), nil)
		return
	}

	respond(c, http.StatusOK, true, "All remaining weeks played successfully", result)
}
