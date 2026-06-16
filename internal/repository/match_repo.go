package repository

// LEARNING NOTE: Bu dosya MatchRepository implementasyonudur.
// Service katmanı "maçları getir", "haftayı güncelle", "puan tablosunu hesapla" gibi metotları çağırır;
// SQL'in nasıl yazıldığı bu dosyada saklanır.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nazazx/league-simulation/internal/models"
	"github.com/jmoiron/sqlx"
)

// matchRepo implements MatchRepository using PostgreSQL.
// LEARNING NOTE: Bu dosya SQL sorgularının bulunduğu katmandır; business logic burada tutulmaz.
// Küçük harfle başladığı için matchRepo sadece repository paketi içinde görünür.
type matchRepo struct {
	// LEARNING NOTE: sqlx.DB, database bağlantı havuzunu temsil eder. Query çalıştırmak için bu alan kullanılır.
	db *sqlx.DB
}

// NewMatchRepository creates a new MatchRepository.
func NewMatchRepository(db *sqlx.DB) MatchRepository {
	// LEARNING NOTE: Dışarıya concrete matchRepo yerine MatchRepository interface'i döndürülür.
	return &matchRepo{db: db}
}

// GetAll returns all matches ordered by week and id.
func (r *matchRepo) GetAll(ctx context.Context) ([]models.Match, error) {
	// LEARNING NOTE: SelectContext birden fazla satır dönen sorgular için kullanılır ve sonucu slice'a doldurur.
	var matches []models.Match
	err := r.db.SelectContext(ctx, &matches,
		"SELECT * FROM matches ORDER BY week, id")
	return matches, err
}

// GetByID returns a single match by ID.
func (r *matchRepo) GetByID(ctx context.Context, id int) (*models.Match, error) {
	// LEARNING NOTE: GetContext tek satır beklenen sorgular için kullanılır. Sonuç yoksa sql.ErrNoRows dönebilir.
	var match models.Match
	err := r.db.GetContext(ctx, &match, "SELECT * FROM matches WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &match, nil
}

// GetByWeek returns all matches for a specific week.
func (r *matchRepo) GetByWeek(ctx context.Context, week int) ([]models.Match, error) {
	var matches []models.Match
	err := r.db.SelectContext(ctx, &matches,
		"SELECT * FROM matches WHERE week = $1 ORDER BY id", week)
	return matches, err
}

// CreateMany inserts multiple matches in a single batch.
func (r *matchRepo) CreateMany(ctx context.Context, matches []models.Match) error {
	// LEARNING NOTE: Batch insert, birçok maçı tek SQL komutuyla ekleyerek daha verimli çalışır.
	// Burada dinamik olarak ($1,$2,$3), ($4,$5,$6) gibi placeholder'lar üretilir.
	if len(matches) == 0 {
		return nil
	}

	query := "INSERT INTO matches (week, home_team_id, away_team_id) VALUES "
	var values []string
	var args []interface{}
	argIdx := 1

	for _, m := range matches {
		// LEARNING NOTE: args slice'ı SQL parametrelerini taşır. Bu yaklaşım string concat ile SQL injection riskini azaltır.
		values = append(values, fmt.Sprintf("($%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2))
		args = append(args, m.Week, m.HomeTeamID, m.AwayTeamID)
		argIdx += 3
	}

	query += strings.Join(values, ", ")
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetNextUnplayedWeek returns the smallest week number that has unplayed matches.
// Returns 0 if all matches are played.
func (r *matchRepo) GetNextUnplayedWeek(ctx context.Context) (int, error) {
	var week int
	err := r.db.GetContext(ctx, &week,
		"SELECT COALESCE(MIN(week), 0) FROM matches WHERE played = FALSE")
	return week, err
}

// UpdateMatchResult sets the score and marks a match as played.
func (r *matchRepo) UpdateMatchResult(ctx context.Context, id int, homeScore, awayScore int) error {
	// LEARNING NOTE: RowsAffected kontrolü, verilen ID ile gerçekten bir maç güncellendi mi anlamak için kullanılır.
	// Eğer affected == 0 ise bu ID'de maç yoktur ve üst katmana not found bilgisi gider.
	result, err := r.db.ExecContext(ctx, `
		UPDATE matches 
		SET home_score = $1, away_score = $2, played = TRUE, played_at = $3, updated_at = $4
		WHERE id = $5`,
		homeScore, awayScore, time.Now(), time.Now(), id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return err
}

// UpdateMatchResults updates multiple match results inside a single transaction.
// If any update fails the entire batch is rolled back — no half-played weeks.
func (r *matchRepo) UpdateMatchResults(ctx context.Context, updates []MatchUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// LEARNING NOTE: Transaction, tüm maç sonuçları birlikte yazılsın ya da hiçbiri yazılmasın garantisi verir.
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	// LEARNING NOTE: defer tx.Rollback() güvenlik ağıdır. Commit başarılı olursa rollback etkisiz kalır;
	// arada hata olursa transaction otomatik geri alınır.

	now := time.Now()
	for _, u := range updates {
		result, err := tx.ExecContext(ctx, `
			UPDATE matches
			SET home_score = $1, away_score = $2, played = TRUE, played_at = $3, updated_at = $4
			WHERE id = $5`,
			u.HomeScore, u.AwayScore, now, now, u.ID)
		if err != nil {
			return fmt.Errorf("failed to update match %d: %w", u.ID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check affected rows for match %d: %w", u.ID, err)
		}
		if affected == 0 {
			return fmt.Errorf("match %d not found during batch update: %w", u.ID, sql.ErrNoRows)
		}
	}

	if err := tx.Commit(); err != nil {
		// LEARNING NOTE: Commit, transaction içindeki tüm update'leri kalıcı hale getirir.
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// DeleteAll removes all matches from the database.
func (r *matchRepo) DeleteAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM matches")
	return err
}

// Exists checks if any matches exist in the database.
func (r *matchRepo) Exists(ctx context.Context) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM matches")
	return count > 0, err
}

// GetStandings calculates the league table from all played matches.
// This is the core standings query — it uses UNION to treat home/away uniformly.
func (r *matchRepo) GetStandings(ctx context.Context) ([]models.Standing, error) {
	// LEARNING NOTE: Puan tablosu ayrı saklanmaz; oynanmış maçlardan SQL aggregation ile hesaplanır.
	// Bunun avantajı: maç sonucu editlenince standings otomatik doğru hesaplanır, ayrı tablo senkronizasyonu gerekmez.
	var standings []models.Standing
	err := r.db.SelectContext(ctx, &standings, `
		SELECT 
			t.id AS team_id,
			t.name AS team_name,
			COALESCE(COUNT(m.goals_for), 0) AS played,
			COALESCE(SUM(CASE WHEN m.goals_for > m.goals_against THEN 1 ELSE 0 END), 0) AS wins,
			COALESCE(SUM(CASE WHEN m.goals_for = m.goals_against THEN 1 ELSE 0 END), 0) AS draws,
			COALESCE(SUM(CASE WHEN m.goals_for < m.goals_against THEN 1 ELSE 0 END), 0) AS losses,
			COALESCE(SUM(m.goals_for), 0) AS goals_for,
			COALESCE(SUM(m.goals_against), 0) AS goals_against,
			COALESCE(SUM(m.goals_for), 0) - COALESCE(SUM(m.goals_against), 0) AS goal_difference,
			COALESCE(SUM(
				CASE 
					WHEN m.goals_for > m.goals_against THEN 3
					WHEN m.goals_for = m.goals_against THEN 1
					ELSE 0
				END
			), 0) AS points
		FROM teams t
		LEFT JOIN (
			-- Home matches
			-- LEARNING NOTE: UNION ALL ile home ve away maçlar aynı formatta tek sanal tabloya çevrilir.
			-- Böylece her takım için goals_for/goals_against hesaplamak kolaylaşır.
			SELECT home_team_id AS team_id, home_score AS goals_for, away_score AS goals_against
			FROM matches WHERE played = TRUE
			UNION ALL
			-- Away matches
			SELECT away_team_id AS team_id, away_score AS goals_for, home_score AS goals_against
			FROM matches WHERE played = TRUE
		) m ON t.id = m.team_id
		GROUP BY t.id, t.name
		ORDER BY points DESC, goal_difference DESC, goals_for DESC, t.name ASC
	`)
	return standings, err
}

// GetHistoricalTeamStats calculates compact team priors from seeded historical matches.
func (r *matchRepo) GetHistoricalTeamStats(ctx context.Context) ([]models.HistoricalTeamStats, error) {
	// LEARNING NOTE: Historical stats, prediction için geçmiş performans prior'ı üretir.
	// Prior, modelin sezon başında "takım hakkında ön bilgi" sahibi olması demektir.
	var stats []models.HistoricalTeamStats
	err := r.db.SelectContext(ctx, &stats, `
		SELECT
			t.id AS team_id,
			COALESCE(COUNT(h.goals_for), 0) AS matches_played,
			COALESCE(AVG(CASE WHEN h.goals_for > h.goals_against THEN 1.0 ELSE 0.0 END), 0) AS win_rate,
			COALESCE(AVG(
				CASE
					WHEN h.goals_for > h.goals_against THEN 3.0
					WHEN h.goals_for = h.goals_against THEN 1.0
					ELSE 0.0
				END
			), 0) AS points_per_match,
			COALESCE(AVG(h.goals_for), 0) AS goals_for_per_match,
			COALESCE(AVG(h.goals_against), 0) AS goals_against_per_match,
			COALESCE(
				50
				+ AVG(
					CASE
						WHEN h.goals_for > h.goals_against THEN 12.0
						WHEN h.goals_for = h.goals_against THEN 4.0
						ELSE -6.0
					END
				)
				+ AVG(h.goals_for - h.goals_against) * 6,
				50
			) AS historical_rating
		FROM teams t
		LEFT JOIN (
			SELECT home_team_id AS team_id, home_score AS goals_for, away_score AS goals_against
			FROM historical_matches
			UNION ALL
			SELECT away_team_id AS team_id, away_score AS goals_for, home_score AS goals_against
			FROM historical_matches
		) h ON t.id = h.team_id
		GROUP BY t.id
		ORDER BY t.id
	`)
	return stats, err
}

// GetHistoricalMatches returns seeded historical matches with optional search filters.
func (r *matchRepo) GetHistoricalMatches(ctx context.Context, search string, season string) ([]models.HistoricalMatchResponse, error) {
	// LEARNING NOTE: Optional filter pattern'i kullanılır; search/season boşsa filtre etkisiz kalır.
	// ILIKE PostgreSQL'de case-insensitive arama yapar; büyük/küçük harf fark etmez.
	var matches []models.HistoricalMatchResponse
	err := r.db.SelectContext(ctx, &matches, `
		SELECT
			h.id,
			h.season,
			h.week,
			home_team.name AS home_team,
			away_team.name AS away_team,
			h.home_score,
			h.away_score
		FROM historical_matches h
		JOIN teams home_team ON home_team.id = h.home_team_id
		JOIN teams away_team ON away_team.id = h.away_team_id
		WHERE
			($1 = '' OR h.season = $1)
			AND (
				$2 = ''
				OR home_team.name ILIKE '%' || $2 || '%'
				OR away_team.name ILIKE '%' || $2 || '%'
				OR h.season ILIKE '%' || $2 || '%'
			)
		ORDER BY h.season, h.week, h.id
	`, season, search)
	return matches, err
}
