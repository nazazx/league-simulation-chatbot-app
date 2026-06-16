package service

// LEARNING NOTE: Bu dosya tek bir maçın skorunu üretir.
// MatchService gerçek hafta oynatırken, PredictionService de kalan maçları simüle ederken bu simulator'ı kullanır.

import (
	"math"
	"math/rand"

	"github.com/nazazx/league-simulation/internal/models"
)

// simulator implements MatchSimulator with a rating-based random algorithm.
// LEARNING NOTE: Bu struct state tutmaz; sadece MatchSimulator interface'indeki Simulate metodunu sağlar.
// `struct{}` boş struct demektir; Go'da veri tutmayan ama method taşıyan tipler için kullanılabilir.
type simulator struct{}

// NewSimulator creates a new MatchSimulator.
func NewSimulator() MatchSimulator {
	// LEARNING NOTE: Constructor interface döndürür. Dış katman sadece "skor simüle edebilen şey" ile ilgilenir.
	return &simulator{}
}

// Simulate generates realistic match scores based on team ratings.
//
// Algorithm:
//   - Attack rating increases expected goals
//   - Opponent defense rating decreases expected goals
//   - Form rating nudges expected goals up or down
//   - Home team receives a small fixed advantage
//   - Final score clamped to 0-5
//
// This keeps scores realistic while making seeded ratings meaningful.
func (s *simulator) Simulate(home, away models.Team) (int, int) {
	// LEARNING NOTE: Method receiver `(s *simulator)`, bu fonksiyonun simulator tipine ait olduğunu gösterir.
	// Return tipi `(int, int)` olduğu için Go bu fonksiyondan iki skor değeri döndürebilir.
	homeGoals := generateGoals(home, away, true)
	awayGoals := generateGoals(away, home, false)
	return homeGoals, awayGoals
}

// generateGoals produces a goal count for one team.
func generateGoals(team, opponent models.Team, isHome bool) int {
	// LEARNING NOTE: Skor üretimi tamamen rastgele değil; attack, defense, form ve home advantage etkili olur.
	// Bu fonksiyon bir takımın gol sayısını üretir; iki takım için ayrı ayrı çağrılır.
	attack := ratingOrDefault(team.AttackRating, team.Strength)
	defenseAgainst := ratingOrDefault(opponent.DefenseRating, opponent.Strength)
	form := ratingOrDefault(team.FormRating, team.Strength)

	expected := 1.15
	// LEARNING NOTE: expected, takımın beklenen gol değeridir. Rating 70 referans kabul edilir.
	// Attack yüksekse artar, rakip defense yüksekse düşer, form iyiyse biraz yükselir.
	expected += float64(attack-70) / 28
	expected -= float64(defenseAgainst-70) / 42
	expected += float64(form-70) / 70
	if isHome {
		expected += 0.25
	}

	noise := rand.NormFloat64()*0.85 + rand.Float64()*0.35
	// LEARNING NOTE: Noise rastgelelik ekler. Böylece güçlü takım hep aynı skoru üretmez.
	goals := int(math.Round(expected + noise))

	if goals < 0 {
		goals = 0
	}
	if goals > 5 {
		goals = 5
	}

	return goals
}

func ratingOrDefault(rating int, fallback int) int {
	// LEARNING NOTE: Rating 0 gelirse fallback kullanılır; fallback de yoksa nötr değer olarak 50 döner.
	if rating > 0 {
		return rating
	}
	if fallback > 0 {
		return fallback
	}
	return 50
}
