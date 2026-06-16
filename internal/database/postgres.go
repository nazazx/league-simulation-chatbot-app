package database

// LEARNING NOTE: Bu dosya PostgreSQL bağlantısını kurar.
// main.go database.NewPostgresDB çağırır; başarılı olursa repository'ler aynı DB bağlantısını kullanır.

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	// LEARNING NOTE: Alt çizgili import, paketin sadece init side-effect'i için yüklendiği anlamına gelir.
	// lib/pq PostgreSQL driver'ını database/sql sistemine register eder.
	_ "github.com/lib/pq"
)

// NewPostgresDB creates a new PostgreSQL connection with retry logic.
func NewPostgresDB(dsn string) (*sqlx.DB, error) {
	// LEARNING NOTE: Bu fonksiyon database bağlantısını kurar ve Docker DB geç açılırsa tekrar dener.
	// Return tipi (*sqlx.DB, error): başarılıysa DB pointer'ı, hata varsa error döner.
	var db *sqlx.DB
	var err error

	// Retry up to 10 times (useful when waiting for Docker postgres to start)
	for i := 0; i < 10; i++ {
		// LEARNING NOTE: Docker Compose'da API bazen PostgreSQL hazır olmadan başlayabilir; retry bunun için var.
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			break
		}
		log.Printf("Waiting for database... attempt %d/10: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after 10 attempts: %w", err)
	}

	// Connection pool settings
	// LEARNING NOTE: Connection pool, her request'te yeni bağlantı açmak yerine mevcut bağlantıları tekrar kullanır.
	// Bu ayarlar aynı anda kaç bağlantı açılabileceğini ve bağlantıların ne kadar yaşayacağını kontrol eder.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Connected to PostgreSQL successfully")
	return db, nil
}
