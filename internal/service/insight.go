package service

// LEARNING NOTE: Bu dosya AI agent/chat mantığını içerir.
// Request akışı: POST /agent/chat -> handler.AgentChat -> InsightService.Chat -> bu dosya.
// Burada iki mod vardır: OpenAI API key varsa tool-calling agent, yoksa deterministic fallback.

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/nazazx/league-simulation/internal/repository"
)

// insightService implements InsightService with an optional OpenAI tool-calling analyst.
// When no API key is configured, it falls back to a deterministic template engine.
// LEARNING NOTE: Bu servis AI agent/chat use-case'ini taşır; OpenAI varsa tool-calling, yoksa fallback çalışır.
// Struct alanları agent'ın ihtiyaç duyduğu veri kaynaklarını ve OpenAI client ayarlarını tutar.
type insightService struct {
	matchRepo     repository.MatchRepository
	predictionSvc PredictionService
	openAIKey     string
	openAIModel   string
	httpClient    *http.Client
}

// NewInsightService creates a new InsightService.
func NewInsightService(
	matchRepo repository.MatchRepository,
	predictionSvc PredictionService,
	openAIKey string,
	openAIModel string,
) InsightService {
	// LEARNING NOTE: OPENAI_MODEL verilmezse default model seçilir.
	// Constructor interface döndürdüğü için dış katman concrete insightService detayını bilmez.
	if openAIModel == "" {
		openAIModel = "gpt-4.1-mini"
	}

	return &insightService{
		matchRepo:     matchRepo,
		predictionSvc: predictionSvc,
		openAIKey:     openAIKey,
		openAIModel:   openAIModel,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

// Chat answers free-form user questions with the same read-only league tools.
func (s *insightService) Chat(ctx context.Context, message string, history []models.AgentChatMessage) (*models.AgentChatResponse, error) {
	// LEARNING NOTE: Request-response akışında AgentChat handler'ı bu metodu çağırır.
	// Buradaki temel sorumluluk: mesajı doğrula, domain dışıysa yönlendir, veri hazırla, OpenAI veya fallback cevabı üret.
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, ErrMessageRequired
	}

	if !isLeagueAgentQuestion(message) {
		// LEARNING NOTE: Domain guard, konu dışı soruların LLM'e gitmeden sınırlandırılmasını sağlar.
		// Bu hem maliyeti azaltır hem de agent'ın uygulama konusu dışına taşmasını engeller.
		return agentHelpResponse(message), nil
	}

	input, err := s.buildInsightInput(ctx)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(s.openAIKey) == "" {
		// LEARNING NOTE: API key yoksa uygulama yine çalışır; demo ortamında OpenAI zorunlu değildir.
		return buildTemplateChatResponse(message, input), nil
	}

	result, err := s.chatWithOpenAI(ctx, message, history)
	if err != nil {
		log.Printf("OpenAI chat unavailable, using local fallback: %v", err)
		fallback := buildTemplateChatResponse(message, input)
		fallback.Answer += " OpenAI connection was unavailable, so I answered with the local fallback analyst."
		return fallback, nil
	}
	return result, nil
}

// --- Data Assembly ---

type insightInput struct {
	// LEARNING NOTE: Agent'a verilecek bağlam burada toplanır: tablo, prediction, historical stats ve sıradaki hafta.
	// Bu struct DB tablosu değildir; agent cevabı için hazırlanmış internal context modelidir.
	Standings       []models.Standing            `json:"standings"`
	Prediction      *models.PredictionResponse   `json:"prediction"`
	HistoricalStats []models.HistoricalTeamStats `json:"historical_stats"`
	NextWeek        int                          `json:"next_week"`
	NextWeekMatches []models.Match               `json:"next_week_matches"`
}

func (s *insightService) buildInsightInput(ctx context.Context) (*insightInput, error) {
	// LEARNING NOTE: Agent SQL bilmez; ihtiyaç duyduğu veri service/repository çağrılarıyla hazırlanır.
	// Bu fonksiyon "agent'a verilecek veri paketi"ni oluşturur.
	standings, err := s.matchRepo.GetStandings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}
	if len(standings) == 0 {
		return nil, ErrNoStandings
	}

	prediction, err := s.predictionSvc.Predict(ctx, 1000)
	if err != nil {
		return nil, err
	}
	if len(prediction.Predictions) == 0 {
		return nil, ErrNoPredictions
	}

	historicalStats, err := s.matchRepo.GetHistoricalTeamStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical stats: %w", err)
	}

	nextWeek, _ := s.matchRepo.GetNextUnplayedWeek(ctx)
	// LEARNING NOTE: `_` blank identifier'dır. Burada next week hatası kritik görülmediği için hata değeri bilinçli yok sayılır.
	var nextWeekMatches []models.Match
	if nextWeek > 0 {
		nextWeekMatches, _ = s.matchRepo.GetByWeek(ctx, nextWeek)
	}

	return &insightInput{
		Standings:       standings,
		Prediction:      prediction,
		HistoricalStats: historicalStats,
		NextWeek:        nextWeek,
		NextWeekMatches: nextWeekMatches,
	}, nil
}

// --- OpenAI Orchestration ---

func (s *insightService) chatWithOpenAI(ctx context.Context, message string, history []models.AgentChatMessage) (*models.AgentChatResponse, error) {
	// LEARNING NOTE: Instructions modele rolünü, sınırlarını ve hangi konularda cevap vereceğini söyler.
	// Prompt engineering tarafı burasıdır: agent'ın sadece lig/prediction/fikstür konularında kalması istenir.
	instructions := strings.Join([]string{
		"You are a focused AI league analyst inside League Simulation, a custom football league workspace.",
		"Answer in the user's language. If the user writes Turkish, answer in Turkish.",
		"Stay within this app's football league domain only.",
		"Allowed topics: current standings, fixtures, match results, historical sample data, prediction algorithm, title probabilities, team form, and how this project works.",
		"If the user asks an unrelated question, politely say you can only help with this league simulation and list 3 example questions they can ask.",
		"You have read-only tools for league standings, predictions, fixtures, historical matches, and historical team stats.",
		"Call tools before answering questions about teams, standings, predictions, fixtures, or historical form.",
		"Do not invent match results, teams, injuries, transfers, or external facts.",
		"Keep answers concise, practical, and presentation-friendly.",
	}, " ")

	response, err := s.createOpenAIResponse(ctx, openAIResponsesRequest{
		// LEARNING NOTE: İlk OpenAI çağrısında model, user message + history + tool listesi ile karar verir.
		// Cevap direkt metin olabilir veya önce tool çağrısı isteyebilir.
		Model:           s.openAIModel,
		Instructions:    instructions,
		Input:           buildChatPrompt(message, history),
		Tools:           insightTools(),
		MaxOutputTokens: 800,
	})
	if err != nil {
		return nil, err
	}

	toolCalls := []string{}
	for i := 0; i < 4; i++ {
		// LEARNING NOTE: Tool-calling döngüsünde model tool ister, backend çalıştırır, sonucu tekrar modele verir.
		// Döngüye limit koymak sonsuz tool çağrısı riskini engeller.
		calls := response.functionCalls()
		if len(calls) == 0 {
			// LEARNING NOTE: Artık function_call yoksa model final cevabı üretmiş demektir.
			answer := strings.TrimSpace(response.OutputText)
			if answer == "" {
				answer = strings.TrimSpace(response.firstText())
			}
			if answer == "" {
				return nil, fmt.Errorf("openai response did not include final text output")
			}
			return &models.AgentChatResponse{
				Answer:    answer,
				Source:    "openai",
				Model:     s.openAIModel,
				ToolCalls: uniqueStrings(toolCalls),
			}, nil
		}

		outputs := s.executeToolBatch(ctx, calls, &toolCalls)

		response, err = s.createOpenAIResponse(ctx, openAIResponsesRequest{
			Model:              s.openAIModel,
			Instructions:       instructions,
			PreviousResponseID: response.ID,
			Input:              outputs,
			Tools:              insightTools(),
			MaxOutputTokens:    800,
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("openai tool-calling loop did not finish")
}

// executeToolBatch runs all function calls and returns their outputs.
// This eliminates the duplicated tool-execution loop in chatWithOpenAI and generateWithOpenAI (DRY).
func (s *insightService) executeToolBatch(ctx context.Context, calls []openAIFunctionCall, toolCallLog *[]string) []openAIFunctionCallOutput {
	// LEARNING NOTE: Bu helper DRY sağlar; tool çalıştırma kodu tek yerde tutulur.
	// `toolCallLog *[]string` pointer'dır; çağrılan tool isimlerini dışarıdaki slice'a ekleyebilir.
	outputs := make([]openAIFunctionCallOutput, 0, len(calls))
	for _, call := range calls {
		if toolCallLog != nil {
			*toolCallLog = append(*toolCallLog, call.Name)
		}
		output, err := s.executeInsightTool(ctx, call)
		if err != nil {
			output = map[string]interface{}{"error": err.Error()}
		}
		outputs = append(outputs, openAIFunctionCallOutput{
			Type:   "function_call_output",
			CallID: call.CallID,
			Output: mustJSON(output),
		})
	}
	return outputs
}

// --- Template Fallback Engine ---

func buildTemplateChatResponse(message string, input *insightInput) *models.AgentChatResponse {
	// LEARNING NOTE: Fallback cevap LLM kullanmaz; mevcut veriden deterministik kısa özet üretir.
	// Böylece OpenAI bağlantısı yoksa bile demo bozulmadan anlamlı cevap döner.
	leader := input.Standings[0]
	favorite := input.Prediction.Predictions[0]
	historicalLeader := topHistoricalTeam(input.HistoricalStats, input.Standings)

	answer := fmt.Sprintf(
		"Current snapshot: %s leads the table with %d points, while %s has the highest title probability at %.1f%%. ",
		leader.TeamName, leader.Points, favorite.TeamName, favorite.ChampionshipProbability,
	)
	if historicalLeader.TeamID != 0 {
		answer += fmt.Sprintf("%s also has the strongest historical seed at %.2f points per match. ", historicalLeader.TeamName, historicalLeader.PointsPerMatch)
	}
	answer += nextUnplayedWeekFocus(input)

	if looksTurkish(message) {
		answer = fmt.Sprintf(
			"Mevcut tabloya göre %s %d puanla lider. Prediction tarafında en yüksek şampiyonluk olasılığı %s için %.1f%%. ",
			leader.TeamName, leader.Points, favorite.TeamName, favorite.ChampionshipProbability,
		)
		if historicalLeader.TeamID != 0 {
			answer += fmt.Sprintf("Historical seed tarafında en güçlü takım %.2f puan/maç ile %s. ", historicalLeader.PointsPerMatch, historicalLeader.TeamName)
		}
		answer += nextUnplayedWeekFocus(input)
	}

	return &models.AgentChatResponse{
		Answer:    strings.TrimSpace(answer),
		Source:    "template",
		ToolCalls: []string{"get_standings", "get_predictions", "get_historical_stats", "get_matches"},
	}
}

// --- Template Helpers ---

func agentHelpResponse(message string) *models.AgentChatResponse {
	// LEARNING NOTE: Konu dışı mesajlarda agent yardım moduna geçer ve kullanıcıya sorabileceği örnekleri verir.
	answer := "I can only help with this football league simulation. You can ask things like: Who is the title favorite and why? What do the predictions say? Which fixtures are next?"
	if looksTurkish(message) {
		answer = "Ben sadece bu futbol ligi simülasyonu hakkında yardımcı olabilirim. Şunları sorabilirsin: Kim favori ve neden? Prediction ne söylüyor? Sıradaki fikstürler hangileri?"
	}

	return &models.AgentChatResponse{
		Answer: answer,
		Source: "template",
	}
}

func isLeagueAgentQuestion(message string) bool {
	// LEARNING NOTE: Bu basit keyword guard profesyonel bir moderation sistemi değildir; ama case için yeterli domain filtresi sağlar.
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		return false
	}

	keywords := []string{
		"league", "table", "standing", "standings", "fixture", "fixtures", "match", "matches",
		"team", "teams", "prediction", "predict", "probability", "champion", "championship",
		"title", "favorite", "favourite", "rank", "points", "goal", "goals", "form",
		"historical", "history", "season", "week", "simulat", "monte carlo", "algorithm",
		"bosphorus", "anka", "galata", "moda", "rovers", "athletic",
		"lig", "tablo", "puan", "fikstür", "fikstur", "maç", "mac", "takım", "takim",
		"tahmin", "olasılık", "olasilik", "şampiyon", "sampiyon", "favori", "sıra", "sira",
		"gol", "form", "geçmiş", "gecmis", "sezon", "hafta", "simülasyon", "simulasyon",
		"algoritma", "monte carlo", "ne sorabilirim", "neler sorabilirim", "yardım", "yardim",
		"help", "what can i ask",
	}

	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

type historicalTeamView struct {
	TeamID         int
	TeamName       string
	PointsPerMatch float64
}

func topHistoricalTeam(stats []models.HistoricalTeamStats, standings []models.Standing) historicalTeamView {
	names := make(map[int]string, len(standings))
	for _, standing := range standings {
		names[standing.TeamID] = standing.TeamName
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].PointsPerMatch != stats[j].PointsPerMatch {
			return stats[i].PointsPerMatch > stats[j].PointsPerMatch
		}
		return stats[i].HistoricalRating > stats[j].HistoricalRating
	})

	if len(stats) == 0 {
		return historicalTeamView{}
	}
	return historicalTeamView{
		TeamID:         stats[0].TeamID,
		TeamName:       names[stats[0].TeamID],
		PointsPerMatch: stats[0].PointsPerMatch,
	}
}

func nextUnplayedWeekFocus(input *insightInput) string {
	if input.NextWeek == 0 {
		return "The season is complete; focus shifts to final table interpretation."
	}
	if len(input.NextWeekMatches) == 0 {
		return fmt.Sprintf("Week %d is the next simulation checkpoint.", input.NextWeek)
	}
	return fmt.Sprintf("Week %d is next, with %d fixtures that can change the table.", input.NextWeek, len(input.NextWeekMatches))
}

func looksTurkish(text string) bool {
	lowered := strings.ToLower(text)
	turkishSignals := []string{"ş", "ğ", "ı", "ö", "ü", "ç", "kim", "neden", "hangi", "tahmin", "puan", "maç", "takım", "lider"}
	for _, signal := range turkishSignals {
		if strings.Contains(lowered, signal) {
			return true
		}
	}
	return false
}
