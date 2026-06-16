package service

// LEARNING NOTE: Bu dosya prediction mekanizmasının merkezidir.
// GET /api/v1/predictions endpoint'i handler üzerinden buradaki Predict metoduna gelir.
// Teorik olarak model: weighted team rating + Monte Carlo simulation.

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/nazazx/league-simulation/internal/repository"
)

const (
	// LEARNING NOTE: const bloğu sabit değerleri tutar. Bu değerler runtime'da değişmez.
	defaultPredictionSimulations = 1000
	maxPredictionSimulations     = 50000
	predictionAvailableAfterWeek = 4
)

// predictionService implements PredictionService with Monte Carlo simulations.
// LEARNING NOTE: Prediction service gerçek maçları değiştirmez; kalan sezonu memory içinde simüle eder.
// Bu yüzden prediction çalıştırmak DB'deki maç sonuçlarını bozmaz.
type predictionService struct {
	// LEARNING NOTE: Service burada repository ve simulator interface'lerine bağımlıdır.
	// Bu yapı, algoritmayı DB erişiminden ve skor üretiminden ayırır.
	teamRepo  repository.TeamRepository
	matchRepo repository.MatchRepository
	simulator MatchSimulator
}

// NewPredictionService creates a new PredictionService.
func NewPredictionService(
	teamRepo repository.TeamRepository,
	matchRepo repository.MatchRepository,
	simulator MatchSimulator,
) PredictionService {
	return &predictionService{
		teamRepo:  teamRepo,
		matchRepo: matchRepo,
		simulator: simulator,
	}
}

// Predict estimates the final league table by simulating the remaining fixtures
// many times in memory. It never writes predicted match results to the database.
func (s *predictionService) Predict(ctx context.Context, simulations int) (*models.PredictionResponse, error) {
	// LEARNING NOTE: Kullanıcı simulations vermediyse default değer kullanılır, aşırı büyükse limitlenir.
	// Bu hem API kullanımını kolaylaştırır hem de yanlışlıkla çok pahalı hesaplama yapılmasını engeller.
	if simulations <= 0 {
		simulations = defaultPredictionSimulations
	}
	if simulations > maxPredictionSimulations {
		simulations = maxPredictionSimulations
	}

	teams, err := s.teamRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}
	if len(teams) == 0 {
		return nil, ErrNoTeams
	}

	matches, err := s.matchRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get matches: %w", err)
	}
	if len(matches) == 0 {
		return nil, ErrFixturesNotGenerated
	}

	currentStandings, err := s.matchRepo.GetStandings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}

	historicalStats, err := s.matchRepo.GetHistoricalTeamStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical team stats: %w", err)
	}

	baseTable := make(map[int]models.Standing, len(currentStandings))
	currentPoints := make(map[int]int, len(currentStandings))
	currentStandingMap := make(map[int]models.Standing, len(currentStandings))
	// LEARNING NOTE: map[int]Standing, teamID -> Standing eşlemesi kurar.
	// Mevcut tablo her simülasyonun başlangıç noktasıdır; her denemede bunun kopyası alınır.
	for _, standing := range currentStandings {
		baseTable[standing.TeamID] = standing
		currentPoints[standing.TeamID] = standing.Points
		currentStandingMap[standing.TeamID] = standing
	}

	historicalMap := make(map[int]models.HistoricalTeamStats, len(historicalStats))
	// LEARNING NOTE: Historical stats de teamID ile map'e çevrilir; böylece her takımın geçmiş verisine O(1) erişilir.
	for _, stat := range historicalStats {
		historicalMap[stat.TeamID] = stat
	}

	teamMap := make(map[int]models.Team, len(teams))
	for _, team := range teams {
		// LEARNING NOTE: buildPredictionTeam, seed rating + historical data + canlı form değerlerini karıştırır.
		// Sonuçta simülasyonda kullanılacak adjusted takım rating'i oluşur.
		teamMap[team.ID] = buildPredictionTeam(team, historicalMap[team.ID], currentStandingMap[team.ID])
	}

	currentWeek := getCurrentWeek(matches)
	stats := make(map[int]*predictionAggregate, len(teams))
	for _, team := range teams {
		stats[team.ID] = &predictionAggregate{}
	}

	for i := 0; i < simulations; i++ {
		// LEARNING NOTE: Monte Carlo, kalan sezonu birçok kez oynatıp olasılık hesabı yapma yaklaşımıdır.
		// Tek bir gelecek tahmini yerine binlerce olası sezon sonu üretilir.
		table := cloneStandings(baseTable)

		for _, match := range matches {
			if match.Played {
				// LEARNING NOTE: Oynanmış maçlar zaten mevcut tabloda olduğu için tekrar simüle edilmez.
				continue
			}

			home, homeOK := teamMap[match.HomeTeamID]
			away, awayOK := teamMap[match.AwayTeamID]
			if !homeOK || !awayOK {
				return nil, fmt.Errorf("match %d contains an unknown team", match.ID)
			}

			homeScore, awayScore := s.simulator.Simulate(home, away)
			// LEARNING NOTE: applyVirtualResult, simüle edilen skoru sadece geçici tabloya işler.
			applyVirtualResult(table, home, away, homeScore, awayScore)
		}

		finalTable := standingsFromMap(table)
		for rank, standing := range finalTable {
			// LEARNING NOTE: rank zero-based gelir; bu yüzden kullanıcıya anlamlı sıra için rank+1 eklenir.
			aggregate := stats[standing.TeamID]
			aggregate.totalPoints += standing.Points
			aggregate.totalRank += rank + 1
			if rank == 0 {
				aggregate.championships++
			}
			if rank < 2 {
				aggregate.topTwoFinishes++
			}
		}
	}

	predictions := make([]models.TeamPrediction, 0, len(teams))
	for _, team := range teams {
		aggregate := stats[team.ID]
		// LEARNING NOTE: Olasılık formülü basittir: takım kaç simülasyonda şampiyon oldu / toplam simülasyon * 100.
		predictions = append(predictions, models.TeamPrediction{
			TeamID:                  team.ID,
			TeamName:                team.Name,
			CurrentPoints:           currentPoints[team.ID],
			PredictedPoints:         round(float64(aggregate.totalPoints)/float64(simulations), 2),
			AverageRank:             round(float64(aggregate.totalRank)/float64(simulations), 2),
			ChampionshipProbability: round(float64(aggregate.championships)*100/float64(simulations), 2),
			TopTwoProbability:       round(float64(aggregate.topTwoFinishes)*100/float64(simulations), 2),
		})
	}

	sort.Slice(predictions, func(i, j int) bool {
		// LEARNING NOTE: sort.Slice custom sıralama syntax'ıdır. Burada en yüksek şampiyonluk olasılığı önce gelir.
		if predictions[i].ChampionshipProbability != predictions[j].ChampionshipProbability {
			return predictions[i].ChampionshipProbability > predictions[j].ChampionshipProbability
		}
		if predictions[i].PredictedPoints != predictions[j].PredictedPoints {
			return predictions[i].PredictedPoints > predictions[j].PredictedPoints
		}
		return predictions[i].TeamName < predictions[j].TeamName
	})

	return &models.PredictionResponse{
		CurrentWeek:        currentWeek,
		AvailableAfterWeek: predictionAvailableAfterWeek,
		Simulations:        simulations,
		Predictions:        predictions,
	}, nil
}

type predictionAggregate struct {
	// LEARNING NOTE: Aggregate struct, tüm simülasyonlardan gelen toplamları tutar; sonra ortalamaya çevrilir.
	// Örneğin totalPoints / simulations = predicted points.
	totalPoints    int
	totalRank      int
	championships  int
	topTwoFinishes int
}

func cloneStandings(source map[int]models.Standing) map[int]models.Standing {
	// LEARNING NOTE: map reference type olduğu için doğrudan kullanılsa simülasyonlar birbirini etkiler.
	// Bu fonksiyon her simülasyon için bağımsız tablo kopyası üretir.
	clone := make(map[int]models.Standing, len(source))
	for teamID, standing := range source {
		clone[teamID] = standing
	}
	return clone
}

func applyVirtualResult(table map[int]models.Standing, home, away models.Team, homeScore, awayScore int) {
	// LEARNING NOTE: Virtual result sadece geçici tabloyu günceller; database'e yazılmaz.
	// Futbol puan kuralı burada uygulanır: galibiyet 3, beraberlik 1, mağlubiyet 0 puan.
	homeStanding := table[home.ID]
	awayStanding := table[away.ID]

	homeStanding.TeamID = home.ID
	homeStanding.TeamName = home.Name
	awayStanding.TeamID = away.ID
	awayStanding.TeamName = away.Name

	homeStanding.Played++
	awayStanding.Played++
	homeStanding.GoalsFor += homeScore
	homeStanding.GoalsAgainst += awayScore
	awayStanding.GoalsFor += awayScore
	awayStanding.GoalsAgainst += homeScore
	homeStanding.GoalDifference = homeStanding.GoalsFor - homeStanding.GoalsAgainst
	awayStanding.GoalDifference = awayStanding.GoalsFor - awayStanding.GoalsAgainst

	switch {
	// LEARNING NOTE: Go'da switch koşulsuz da yazılabilir. Bu kullanım if/else zincirinin daha okunur halidir.
	case homeScore > awayScore:
		homeStanding.Wins++
		homeStanding.Points += 3
		awayStanding.Losses++
	case homeScore < awayScore:
		awayStanding.Wins++
		awayStanding.Points += 3
		homeStanding.Losses++
	default:
		homeStanding.Draws++
		awayStanding.Draws++
		homeStanding.Points++
		awayStanding.Points++
	}

	table[home.ID] = homeStanding
	table[away.ID] = awayStanding
}

func standingsFromMap(table map[int]models.Standing) []models.Standing {
	// LEARNING NOTE: Simülasyon tablosu map olarak tutulur, fakat sıralama yapabilmek için slice'a çevrilir.
	standings := make([]models.Standing, 0, len(table))
	for _, standing := range table {
		standings = append(standings, standing)
	}

	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Points != standings[j].Points {
			return standings[i].Points > standings[j].Points
		}
		if standings[i].GoalDifference != standings[j].GoalDifference {
			return standings[i].GoalDifference > standings[j].GoalDifference
		}
		if standings[i].GoalsFor != standings[j].GoalsFor {
			return standings[i].GoalsFor > standings[j].GoalsFor
		}
		return standings[i].TeamName < standings[j].TeamName
	})

	return standings
}

func getCurrentWeek(matches []models.Match) int {
	currentWeek := 0
	for _, match := range matches {
		if match.Played && match.Week > currentWeek {
			currentWeek = match.Week
		}
	}
	return currentWeek
}

func round(value float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(value*ratio) / ratio
}

func buildPredictionTeam(team models.Team, historical models.HistoricalTeamStats, standing models.Standing) models.Team {
	// LEARNING NOTE: Prediction rating'i seed rating, historical prior ve current-season form karışımıdır.
	// Formül mantığı: başlangıç gücü + geçmiş veri + bu sezonki canlı performans.
	historicalRating := 50.0
	historicalAttack := 50.0
	historicalDefense := 50.0
	if historical.MatchesPlayed > 0 {
		// LEARNING NOTE: Historical data varsa geçmiş maç başı gol/puan değerlerinden attack-defense-form etkisi çıkarılır.
		historicalRating = clampFloat(historical.HistoricalRating, 35, 95)
		historicalAttack = clampFloat(50+historical.GoalsForPerMatch*14, 35, 95)
		historicalDefense = clampFloat(50+(2.2-historical.GoalsAgainstPerMatch)*14, 35, 95)
	}

	liveForm := float64(ratingOrDefault(team.FormRating, team.Strength))
	liveAttack := float64(ratingOrDefault(team.AttackRating, team.Strength))
	liveDefense := float64(ratingOrDefault(team.DefenseRating, team.Strength))
	if standing.Played > 0 {
		// LEARNING NOTE: Takım bu sezon maç oynadıysa live form hesaplanır.
		// Points, averaj, atılan gol ve yenilen gol birlikte değerlendirilir.
		played := float64(standing.Played)
		pointsPerMatchRating := float64(standing.Points) / played / 3 * 100
		goalDifferenceRating := 50 + (float64(standing.GoalDifference)/played)*10
		liveAttack = 50 + (float64(standing.GoalsFor)/played)*12
		liveDefense = 50 + (2.0-float64(standing.GoalsAgainst)/played)*12
		liveForm = pointsPerMatchRating*0.55 + goalDifferenceRating*0.25 + ((liveAttack+liveDefense)/2)*0.20
	}

	adjusted := team
	// LEARNING NOTE: Aşağıdaki ağırlıklar modelin teorik kısmıdır.
	// Strength = seed %45 + historical %35 + live form %20.
	adjusted.Strength = roundRating(
		float64(ratingOrDefault(team.Strength, 50))*0.45 +
			historicalRating*0.35 +
			liveForm*0.20,
	)
	adjusted.AttackRating = roundRating(
		// LEARNING NOTE: Attack rating'de seed attack en baskın değer; historical attack ve live attack yardımcı sinyaldir.
		float64(ratingOrDefault(team.AttackRating, team.Strength))*0.55 +
			historicalAttack*0.25 +
			liveAttack*0.20,
	)
	adjusted.DefenseRating = roundRating(
		// LEARNING NOTE: Defense rating de aynı mantıkla seed defense, historical defense ve live defense karışımıdır.
		float64(ratingOrDefault(team.DefenseRating, team.Strength))*0.55 +
			historicalDefense*0.25 +
			liveDefense*0.20,
	)
	adjusted.FormRating = roundRating(
		// LEARNING NOTE: Form rating'de canlı sezon performansının ağırlığı daha yüksektir.
		float64(ratingOrDefault(team.FormRating, team.Strength))*0.40 +
			historicalRating*0.25 +
			liveForm*0.35,
	)
	return adjusted
}

func roundRating(value float64) int {
	return int(math.Round(clampFloat(value, 35, 95)))
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
