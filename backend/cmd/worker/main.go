package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/timeless/backend/internal/bootstrap"
	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("%v", err)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	rt, err := bootstrap.NewWorkerRuntime(cfg, db, logger)
	if err != nil {
		log.Fatalf("failed to initialize worker runtime: %v", err)
	}
	defer rt.StopBackgroundGoroutines()

	logger.Info("starting worker", "redis", cfg.RedisURL)
	if err := rt.Server.Run(rt.Mux); err != nil {
		logger.Error("worker failed", "error", err)
		os.Exit(1)
	}
}
