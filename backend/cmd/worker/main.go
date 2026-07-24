package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/database"
	"github.com/timeless/backend/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to parse redis URL: %v", err)
	}

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			Logger: &asynqLogger{logger: logger},
		},
	)

	mux := asynq.NewServeMux()
	handlers := worker.NewHandlers(logger, db)
	worker.RegisterHandlers(mux, handlers)

	logger.Info("starting worker", "redis", cfg.RedisURL)
	if err := srv.Run(mux); err != nil {
		logger.Error("worker failed", "error", err)
		os.Exit(1)
	}
}

type asynqLogger struct {
	logger *slog.Logger
}

func (l *asynqLogger) Debug(args ...interface{}) { l.logger.Debug("asynq", "msg", args) }
func (l *asynqLogger) Info(args ...interface{})  { l.logger.Info("asynq", "msg", args) }
func (l *asynqLogger) Warn(args ...interface{})  { l.logger.Warn("asynq", "msg", args) }
func (l *asynqLogger) Error(args ...interface{}) { l.logger.Error("asynq", "msg", args) }
func (l *asynqLogger) Fatal(args ...interface{}) {
	l.logger.Error("asynq fatal", "msg", args)
	os.Exit(1)
}
