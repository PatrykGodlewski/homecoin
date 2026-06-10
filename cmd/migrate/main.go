package main

import (
	"os"

	"github.com/godlew/homecoin/internal/infrastructure/config"
	"github.com/godlew/homecoin/internal/infrastructure/logger"
	"github.com/godlew/homecoin/internal/infrastructure/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.LogLevel)
	if err := postgres.RunMigrations(cfg.DatabaseURL, log); err != nil {
		log.Error("database migration failed", "error", err)
		os.Exit(1)
	}
}
