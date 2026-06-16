package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nazazx/league-simulation/internal/models"
)

// insightTools returns the read-only tool definitions exposed to the OpenAI agent.
// All tools are read-only by design — no mutation tools are registered.
func insightTools() []openAITool {
	// LEARNING NOTE: Agent'a reset/play/update gibi yazma tool'ları verilmez; sadece read-only veri tool'ları vardır.
	emptyObject := map[string]interface{}{
		"type":                 "object",
		"properties":           map[string]interface{}{},
		"additionalProperties": false,
	}

	return []openAITool{
		{
			Type:        "function",
			Name:        "get_standings",
			Description: "Get the current league table sorted by points and standard football tie-breakers.",
			Parameters:  emptyObject,
			Strict:      true,
		},
		{
			Type:        "function",
			Name:        "get_predictions",
			Description: "Get Monte Carlo title predictions. simulations is optional and defaults to 1000.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"simulations": map[string]interface{}{
						"type":        "integer",
						"description": "Number of Monte Carlo simulations, between 1 and 50000.",
					},
				},
				"additionalProperties": false,
			},
			Strict: false,
		},
		{
			Type:        "function",
			Name:        "get_historical_stats",
			Description: "Get compact team historical statistics used as prediction priors.",
			Parameters:  emptyObject,
			Strict:      true,
		},
		{
			Type:        "function",
			Name:        "get_matches",
			Description: "Get current-season fixtures and results. week is optional.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"week": map[string]interface{}{
						"type":        "integer",
						"description": "Optional week number to filter current-season matches.",
					},
				},
				"additionalProperties": false,
			},
			Strict: false,
		},
		{
			Type:        "function",
			Name:        "get_historical_matches",
			Description: "Get seeded historical sample matches. search and season are optional.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Optional team or season search text.",
					},
					"season": map[string]interface{}{
						"type":        "string",
						"description": "Optional season filter, for example 2023-2024.",
					},
				},
				"additionalProperties": false,
			},
			Strict: false,
		},
	}
}

// executeInsightTool dispatches a function call to the appropriate read-only data source.
func (s *insightService) executeInsightTool(ctx context.Context, call openAIFunctionCall) (interface{}, error) {
	// LEARNING NOTE: Dispatcher pattern: tool adına göre doğru repository/service fonksiyonu çağrılır.
	args := map[string]interface{}{}
	if strings.TrimSpace(call.Arguments) != "" {
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return nil, fmt.Errorf("invalid tool arguments for %s: %w", call.Name, err)
		}
	}

	switch call.Name {
	case "get_standings":
		return s.matchRepo.GetStandings(ctx)
	case "get_predictions":
		return s.predictionSvc.Predict(ctx, intArg(args, "simulations", 1000))
	case "get_historical_stats":
		return s.matchRepo.GetHistoricalTeamStats(ctx)
	case "get_matches":
		week := intArg(args, "week", 0)
		if week > 0 {
			return s.matchRepo.GetByWeek(ctx, week)
		}
		return s.matchRepo.GetAll(ctx)
	case "get_historical_matches":
		return s.matchRepo.GetHistoricalMatches(ctx, stringArg(args, "search"), stringArg(args, "season"))
	default:
		return nil, fmt.Errorf("unknown tool %s", call.Name)
	}
}

// --- Chat prompt builder ---

// buildChatPrompt formats the conversation history and current question into an OpenAI input string.
func buildChatPrompt(message string, history []models.AgentChatMessage) string {
	// LEARNING NOTE: LLM stateless çalışır; konuşma geçmişi her istekte prompt'a tekrar eklenir.
	var b strings.Builder
	b.WriteString("Recent conversation:\n")
	for _, item := range compactChatHistory(history, 8) {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		content := strings.TrimSpace(item.Content)
		if role == "" || content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			continue
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	b.WriteString("\nCurrent user question: ")
	b.WriteString(message)
	return b.String()
}

// compactChatHistory returns the last N messages from the history.
func compactChatHistory(history []models.AgentChatMessage, limit int) []models.AgentChatMessage {
	if limit <= 0 || len(history) <= limit {
		return history
	}
	return history[len(history)-limit:]
}

// --- Tool argument helpers ---

func intArg(args map[string]interface{}, key string, fallback int) int {
	// LEARNING NOTE: JSON sayıları Go'da çoğunlukla float64 parse edilir; type switch bu yüzden var.
	value, ok := args[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return fallback
	}
}

func stringArg(args map[string]interface{}, key string) string {
	value, ok := args[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

// uniqueStrings deduplicates a string slice while preserving order.
func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
