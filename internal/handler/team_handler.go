package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nazazx/league-simulation/internal/models"
)

// HealthCheck returns API and database status.
// GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	// LEARNING NOTE: Health endpoint API ve database ayakta mı hızlıca kontrol etmek için kullanılır.
	dbStatus := "up"
	if err := h.db.Ping(); err != nil {
		dbStatus = "down"
	}

	respond(c, http.StatusOK, true, "API is running", gin.H{
		"api":      "up",
		"database": dbStatus,
	})
}

// GetTeams returns all teams.
// GET /teams
func (h *Handler) GetTeams(c *gin.Context) {
	// LEARNING NOTE: Handler burada request'i alır, repository'den veriyi ister ve JSON response döner.
	teams, err := h.teamRepo.GetAll(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to get teams", nil)
		return
	}

	respond(c, http.StatusOK, true, "Teams retrieved successfully", gin.H{
		"teams": teams,
	})
}

// SaveTeam creates a team or updates an existing team with the same name.
// POST /teams
func (h *Handler) SaveTeam(c *gin.Context) {
	if !h.ensureRosterEditable(c) {
		return
	}

	var req teamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, false, "Invalid team payload", nil)
		return
	}

	team, err := normalizeTeam(req)
	if err != nil {
		respond(c, http.StatusBadRequest, false, err.Error(), nil)
		return
	}

	saved, err := h.teamRepo.Upsert(c.Request.Context(), team)
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to save team", nil)
		return
	}

	respond(c, http.StatusOK, true, "Team saved successfully", gin.H{
		"team": saved,
	})
}

// ImportTeams creates, updates, or replaces teams in bulk.
// POST /teams/import
func (h *Handler) ImportTeams(c *gin.Context) {
	if !h.ensureRosterEditable(c) {
		return
	}

	var req importTeamsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, http.StatusBadRequest, false, "Invalid import payload", nil)
		return
	}
	if len(req.Teams) == 0 {
		respond(c, http.StatusBadRequest, false, "Import requires at least one team", nil)
		return
	}
	if len(req.Teams) > 100 {
		respond(c, http.StatusBadRequest, false, "Import is limited to 100 teams", nil)
		return
	}

	teams := make([]models.Team, 0, len(req.Teams))
	seen := make(map[string]struct{}, len(req.Teams))
	for _, item := range req.Teams {
		team, err := normalizeTeam(item)
		if err != nil {
			respond(c, http.StatusBadRequest, false, err.Error(), nil)
			return
		}

		key := strings.ToLower(team.Name)
		if _, ok := seen[key]; ok {
			respond(c, http.StatusBadRequest, false, "Duplicate team in import: "+team.Name, nil)
			return
		}
		seen[key] = struct{}{}
		teams = append(teams, team)
	}

	var (
		saved []models.Team
		err   error
	)
	if req.Replace {
		saved, err = h.teamRepo.ReplaceAll(c.Request.Context(), teams)
	} else {
		saved, err = h.teamRepo.UpsertMany(c.Request.Context(), teams)
	}
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to import teams", nil)
		return
	}

	respond(c, http.StatusOK, true, "Teams imported successfully", gin.H{
		"teams": saved,
	})
}

// DeleteTeam removes a team before fixtures are generated.
// DELETE /teams/:id
func (h *Handler) DeleteTeam(c *gin.Context) {
	if !h.ensureRosterEditable(c) {
		return
	}

	teamID, err := strconv.Atoi(c.Param("id"))
	if err != nil || teamID < 1 {
		respond(c, http.StatusBadRequest, false, "Invalid team id", nil)
		return
	}

	if err := h.teamRepo.Delete(c.Request.Context(), teamID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respond(c, http.StatusNotFound, false, "Team not found", nil)
			return
		}
		respond(c, http.StatusInternalServerError, false, "Failed to delete team", nil)
		return
	}

	respond(c, http.StatusOK, true, "Team deleted successfully", nil)
}

func (h *Handler) ensureRosterEditable(c *gin.Context) bool {
	exists, err := h.matchRepo.Exists(c.Request.Context())
	if err != nil {
		respond(c, http.StatusInternalServerError, false, "Failed to inspect fixture state", nil)
		return false
	}
	if exists {
		respond(c, http.StatusConflict, false, "Roster is locked while fixtures exist. Reset the league before editing teams.", nil)
		return false
	}
	return true
}

func normalizeTeam(req teamRequest) (models.Team, error) {
	name := strings.Join(strings.Fields(req.Name), " ")
	if name == "" {
		return models.Team{}, fmt.Errorf("team name is required")
	}
	if len(name) > 100 {
		return models.Team{}, fmt.Errorf("team name must be 100 characters or less")
	}

	strength, err := ratingValue("strength", req.Strength, 50)
	if err != nil {
		return models.Team{}, err
	}
	attack, err := ratingValue("attack_rating", req.AttackRating, strength)
	if err != nil {
		return models.Team{}, err
	}
	defense, err := ratingValue("defense_rating", req.DefenseRating, strength)
	if err != nil {
		return models.Team{}, err
	}
	form, err := ratingValue("form_rating", req.FormRating, strength)
	if err != nil {
		return models.Team{}, err
	}

	return models.Team{
		Name:          name,
		Strength:      strength,
		AttackRating:  attack,
		DefenseRating: defense,
		FormRating:    form,
	}, nil
}

func ratingValue(field string, value *int, fallback int) (int, error) {
	rating := fallback
	if value != nil {
		rating = *value
	}
	if rating < 1 || rating > 100 {
		return 0, fmt.Errorf("%s must be between 1 and 100", field)
	}
	return rating, nil
}
