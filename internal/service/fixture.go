package service

// LEARNING NOTE: Bu dosya fikstür üretme iş kuralını içerir.
// Request-response akışında POST /fixtures/generate endpoint'i önce handler'a gelir,
// handler da buradaki FixtureService.Generate metodunu çağırır.

import (
	"context"
	"fmt"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/nazazx/league-simulation/internal/repository"
)

// fixtureService implements FixtureService.
// LEARNING NOTE: Küçük harfle başlayan type dış paketlere kapalıdır; Go'da buna unexported denir.
// Dışarıya concrete fixtureService yerine FixtureService interface'i gösterilir.
type fixtureService struct {
	// LEARNING NOTE: Bu alanlar dependency'dir. fixtureService takım listesini teamRepo'dan alır, ürettiği maçları matchRepo ile kaydeder.
	teamRepo  repository.TeamRepository
	matchRepo repository.MatchRepository
}

// NewFixtureService creates a new FixtureService.
func NewFixtureService(teamRepo repository.TeamRepository, matchRepo repository.MatchRepository) FixtureService {
	// LEARNING NOTE: Constructor fonksiyonudur. Parametre olarak repository interface'leri alır ve FixtureService interface'i döndürür.
	// Bu syntax'ta `&fixtureService{...}` struct'ın adresini döndürür; Go bunu interface'e uyduğu için kabul eder.
	return &fixtureService{
		teamRepo:  teamRepo,
		matchRepo: matchRepo,
	}
}

// Generate creates a quadruple round-robin fixture for all teams.
//
// Algorithm (Circle Method):
//  1. Fix one team, rotate the rest to generate n-1 rounds (first half).
//  2. Mirror all rounds by swapping home/away for the second half.
//  3. Repeat the double round-robin once more.
//  4. Result: 4*(n-1) weeks, n/2 matches per week, 2*n*(n-1) total matches.
//
// Example for 4 teams: 12 weeks, 2 matches/week, 24 total matches.
func (s *fixtureService) Generate(ctx context.Context) error {
	// Check if fixtures already exist
	// LEARNING NOTE: context.Context request iptal edilirse DB çağrılarının da iptal edilebilmesini sağlar.
	// `(s *fixtureService)` receiver'dır; Generate metodunun fixtureService'e ait olduğunu gösterir.
	exists, err := s.matchRepo.Exists(ctx)
	if err != nil {
		// LEARNING NOTE: `%w` hatayı wrap eder. Böylece üst katmanda errors.Is ile asıl hata hâlâ yakalanabilir.
		return fmt.Errorf("failed to check existing fixtures: %w", err)
	}
	if exists {
		// LEARNING NOTE: Aynı sezon için iki kere fixture üretmeyi engelliyoruz; önce reset isteniyor.
		return ErrFixturesAlreadyExist
	}

	// Get all teams
	teams, err := s.teamRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get teams: %w", err)
	}

	// Validate team count
	n := len(teams)
	// LEARNING NOTE: Bu algoritma en az 4 ve çift sayıda takım bekler. 4 takımda 12 hafta oluşur.
	if n < 4 {
		return fmt.Errorf("%w: got %d", ErrMinimumTeams, n)
	}
	if n%2 != 0 {
		return fmt.Errorf("%w: got %d", ErrEvenTeamsRequired, n)
	}

	// Generate fixtures using circle method
	// LEARNING NOTE: İş kuralı service içinde, kalıcı kayıt işlemi repository içinde tutulur.
	// generateRoundRobin sadece maç listesini üretir; DB'ye yazma işi aşağıdaki CreateMany çağrısındadır.
	matches := generateRoundRobin(teams)

	// Save to database
	if err := s.matchRepo.CreateMany(ctx, matches); err != nil {
		return fmt.Errorf("failed to save fixtures: %w", err)
	}

	return nil
}

// generateRoundRobin creates a quadruple round-robin schedule.
func generateRoundRobin(teams []models.Team) []models.Match {
	// LEARNING NOTE: Bu helper DB'ye dokunmaz; input takım listesinden memory'de fixture üretir.
	// []models.Team bir slice'tır; Go'da dinamik uzunluklu liste gibi düşünebilirsin.
	n := len(teams)
	var firstCycle []models.Match

	// Create a working copy of team IDs (excluding the first one which stays fixed)
	// LEARNING NOTE: make([]models.Team, n-1) belirli uzunlukta slice oluşturur. copy ise takımları bu yeni slice'a kopyalar.
	rotating := make([]models.Team, n-1)
	copy(rotating, teams[1:])

	// First half: n-1 weeks
	for week := 1; week <= n-1; week++ {
		// LEARNING NOTE: Circle method'da ilk takım sabit kalır, diğer takımlar rotate edilerek her hafta farklı eşleşme üretilir.
		// First team vs last in rotating list
		firstCycle = append(firstCycle, models.Match{
			Week:       week,
			HomeTeamID: teams[0].ID,
			AwayTeamID: rotating[len(rotating)-1].ID,
		})

		// Pair remaining teams from outside-in
		for i := 0; i < (n-2)/2; i++ {
			firstCycle = append(firstCycle, models.Match{
				Week:       week,
				HomeTeamID: rotating[i].ID,
				AwayTeamID: rotating[len(rotating)-2-i].ID,
			})
		}

		// Rotate: move last element to front
		// LEARNING NOTE: Bu blok rotating slice'ını döndürür. Böylece sonraki hafta eşleşmeler değişir.
		last := rotating[len(rotating)-1]
		copy(rotating[1:], rotating[:len(rotating)-1])
		rotating[0] = last
	}

	// Second half: mirror first half with swapped home/away
	// LEARNING NOTE: İlk döngü takımların birbirleriyle bir kez oynamasını sağlar. Burada home/away terslenerek ikinci yarı yapılır.
	matches := make([]models.Match, 0, len(firstCycle)*4)
	matches = append(matches, firstCycle...)
	firstHalfCount := len(firstCycle)
	for i := 0; i < firstHalfCount; i++ {
		m := firstCycle[i]
		matches = append(matches, models.Match{
			Week:       m.Week + (n - 1),
			HomeTeamID: m.AwayTeamID, // swap home/away
			AwayTeamID: m.HomeTeamID,
		})
	}

	// Third and fourth quarters repeat the balanced home/away cycle.
	// LEARNING NOTE: Double round-robin bir kez daha tekrar edilir; toplam quadruple round-robin olur.
	doubleRoundRobinCount := len(matches)
	for i := 0; i < doubleRoundRobinCount; i++ {
		m := matches[i]
		matches = append(matches, models.Match{
			Week:       m.Week + 2*(n-1),
			HomeTeamID: m.HomeTeamID,
			AwayTeamID: m.AwayTeamID,
		})
	}

	return matches
}
