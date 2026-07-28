package main

import (
	"errors"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/hibiken/asynq"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/database"
	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
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
			Logger:         &asynqLogger{logger: logger},
			RetryDelayFunc: rateLimitAwareRetryDelay,
		},
	)

	cipher := security.NewCredentialCipher(cfg.CredentialKey(), cfg.CredentialsEncryptionKeyPrevious...)
	syncRunRepo := repository.NewSyncRunRepository(db)
	registryCfg := integration.RegistryConfig{NotionClientID: cfg.NotionClientID, NotionClientSecret: cfg.NotionClientSecret}

	mux := asynq.NewServeMux()
	handlers := worker.NewHandlers(logger, db, cipher, syncRunRepo, registryCfg)
	worker.RegisterHandlers(mux, handlers)

	stopScheduler := worker.StartPeriodicResync(db, syncRunRepo, cfg, logger)
	defer stopScheduler()

	logger.Info("starting worker", "redis", cfg.RedisURL)
	if err := srv.Run(mux); err != nil {
		logger.Error("worker failed", "error", err)
		os.Exit(1)
	}
}

// rateLimitAwareRetryDelay backs off using the provider's own Retry-After
// when a sync failed because of a rate limit, instead of asynq's generic
// exponential curve — the provider told us exactly how long to wait.
func rateLimitAwareRetryDelay(n int, e error, t *asynq.Task) time.Duration {
	var rl *integration.RateLimitError
	if errors.As(e, &rl) {
		return rl.RetryAfterDuration(30 * time.Second)
	}
	return asynq.DefaultRetryDelayFunc(n, e, t)
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
