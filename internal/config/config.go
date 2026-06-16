package config

// LEARNING NOTE: Bu dosya environment variable okuma işini merkezi hale getirir.
// Böylece port, database bilgileri ve OpenAI ayarları kodun içine sabit yazılmaz.

import (
	"fmt"
	"os"
)

// Config holds all application configuration values.
// LEARNING NOTE: Config struct, port, database ve OpenAI gibi ortam ayarlarını tek yerde toplar.
// main.go cfg := config.Load() dediğinde bu struct'ı kullanır.
type Config struct {
	ServerPort  string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	OpenAIKey   string
	OpenAIModel string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	// LEARNING NOTE: Environment variable yoksa default değer kullanılır; lokal çalıştırma kolaylaşır.
	// Docker Compose production benzeri ortamda bu değerleri override edebilir.
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "league"),
		DBPassword:  getEnv("DB_PASSWORD", "league123"),
		DBName:      getEnv("DB_NAME", "league"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		OpenAIKey:   getEnv("OPENAI_API_KEY", ""),
		OpenAIModel: getEnv("OPENAI_MODEL", "gpt-4.1-mini"),
	}
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	// LEARNING NOTE: DSN, PostgreSQL'e bağlanmak için gereken connection string'dir.
	// `(c *Config)` receiver sayesinde bu metot Config alanlarına c.DBHost gibi erişir.
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

// getEnv reads an env variable or returns a default value.
func getEnv(key, fallback string) string {
	// LEARNING NOTE: os.LookupEnv iki değer döndürür: değer ve bulundu mu bilgisi.
	// `if val, ok := ...; ok {}` syntax'ı Go'da kısa scoped değişken tanımlamadır.
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
