package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nazazx/league-simulation/internal/service"
)

// GenerateFixtures generates the league fixtures.
// POST /fixtures/generate
func (h *Handler) GenerateFixtures(c *gin.Context) {
	// LEARNING NOTE: Handler algoritmayı bilmez; fikstür üretme kuralını FixtureService'e devreder.
	err := h.fixtureSvc.Generate(c.Request.Context())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrFixturesAlreadyExist) {
			status = http.StatusConflict
		} else if errors.Is(err, service.ErrMinimumTeams) || errors.Is(err, service.ErrEvenTeamsRequired) {
			status = http.StatusBadRequest
		}
		respond(c, status, false, err.Error(), nil)
		return
	}

	respond(c, http.StatusCreated, true, "Schedule built successfully", nil)
}

// Reset clears all current-season matches.
// POST /reset
func (h *Handler) Reset(c *gin.Context) {
	// LEARNING NOTE: Reset current-season maçlarını siler; historical data geçmiş veri gibi sabit kalır.
	err := h.matchRepo.DeleteAll(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to reset", nil)
		return
	}

	respond(c, http.StatusOK, true,
		"Season cleared successfully. Build a new schedule to start again.", nil)
}
