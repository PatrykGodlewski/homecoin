package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	JWTAccessTTL        time.Duration
	JWTRefreshTTL       time.Duration
	OpenAIAPIKey        string
	OpenAIModel         string
	LogLevel            string
	AutoMigrate         bool
	WorkerURL           string
	WorkerInternalToken string
	TLSBehindProxy      bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://homecoin:homecoin@localhost:5432/homecoin?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:   getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}

	var err error
	cfg.JWTAccessTTL, err = time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("parse JWT_ACCESS_TTL: %w", err)
	}
	cfg.JWTRefreshTTL, err = time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("parse JWT_REFRESH_TTL: %w", err)
	}

	cfg.AutoMigrate, err = strconv.ParseBool(getEnv("AUTO_MIGRATE", "true"))
	if err != nil {
		return nil, fmt.Errorf("parse AUTO_MIGRATE: %w", err)
	}

	cfg.WorkerURL = os.Getenv("WORKER_URL")
	cfg.WorkerInternalToken = os.Getenv("WORKER_INTERNAL_TOKEN")
	cfg.TLSBehindProxy, err = strconv.ParseBool(getEnv("TLS_BEHIND_PROXY", "false"))
	if err != nil {
		return nil, fmt.Errorf("parse TLS_BEHIND_PROXY: %w", err)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
