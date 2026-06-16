package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nazazx/league-simulation/internal/service"
)

// GetStandings returns the current league standings.
// GET /standings
func (h *Handler) GetStandings(c *gin.Context) {
	// LEARNING NOTE: Puan tablosu service/repository üzerinden hesaplanır; handler sadece API cevabı hazırlar.
	standings, err := h.standingSvc.GetStandings(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to get standings", nil)
		return
	}

	respond(c, http.StatusOK, true, "Standings retrieved successfully", gin.H{
		"standings": standings,
	})
}

// GetPredictions estimates the final league table from the current state.
// GET /predictions?simulations=10000
func (h *Handler) GetPredictions(c *gin.Context) {
	simulations := 0
	if value := c.Query("simulations"); value != "" {
		// LEARNING NOTE: Query parametresi URL sonunda gelir: /predictions?simulations=10000 gibi.
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			respond(c, http.StatusBadRequest, false, "Invalid simulations value", nil)
			return
		}
		simulations = parsed
	}

	result, err := h.predictionSvc.Predict(c.Request.Context(), simulations)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrFixturesNotGenerated) {
			status = http.StatusBadRequest
		}
		respond(c, status, false, err.Error(), nil)
		return
	}

	respond(c, http.StatusOK, true, "Predictions calculated successfully", result)
}

// AgentChat answers free-form questions with the read-only AI league agent.
// POST /agent/chat
func (h *Handler) AgentChat(c *gin.Context) {
	// LEARNING NOTE: Chat endpoint kullanıcı mesajını alır ve InsightService üzerinden agent cevabı üretir.
	var req agentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	result, err := h.insightSvc.Chat(c.Request.Context(), req.Message, req.History)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrFixturesNotGenerated) ||
			errors.Is(err, service.ErrMessageRequired) ||
			errors.Is(err, service.ErrNoStandings) {
			status = http.StatusBadRequest
		}
		respond(c, status, false, err.Error(), nil)
		return
	}

	respond(c, http.StatusOK, true, "Agent answered successfully", result)
}
