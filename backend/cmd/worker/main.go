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
	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/mapping"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
	"github.com/timeless/backend/internal/syncengine"
	"github.com/timeless/backend/internal/worker"
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
			ErrorHandler:   worker.NewDeadLetterHandler(db, logger),
		},
	)

	cipher := security.NewCredentialCipher(cfg.CredentialKey(), cfg.CredentialsEncryptionKeyPrevious...)
	syncRunRepo := repository.NewSyncRunRepository(db)
	integrationRepo := repository.NewIntegrationRepository(db)
	registryCfg := integration.RegistryConfig{NotionClientID: cfg.NotionClientID, NotionClientSecret: cfg.NotionClientSecret}

	// Subscribers register here as features that need to react to
	// cross-entity events (search indexing, notifications, etc.) land —
	// an unconfigured Bus (no SetPublisher, no Subscribe calls) is a
	// documented no-op per eventbus.Bus, so this is safe to wire now
	// ahead of any subscriber existing yet.
	bus := eventbus.NewBus()

	// Notion write-back: entity services (router.Setup) publish
	// CompanyCreated/Updated/Deleted etc.; syncengine.PushService
	// subscribes here and, for every org with an active FieldMapping,
	// translates the current record through mapping.NotionAdapter and
	// pushes it. Registered on every CRUD event of every syncable
	// entity type up front — PushService itself no-ops for orgs with no
	// mapping configured, so this isn't gated on any integration
	// actually being connected.
	fieldMappingRepo := repository.NewFieldMappingRepository(db)
	syncedEntityRepo := repository.NewSyncedEntityRepository(db)
	syncHistoryRepo := repository.NewSyncHistoryRepository(db)
	notionClient := integration.NewNotionClient(cfg.NotionClientID, cfg.NotionClientSecret)
	adapters := map[string]mapping.Adapter{
		"notion": mapping.NewNotionAdapter(notionClient),
	}
	pushSvc := syncengine.NewPushService(db, cipher, fieldMappingRepo, integrationRepo, syncedEntityRepo, syncHistoryRepo, adapters)
	for _, evt := range []string{
		eventbus.CompanyCreated, eventbus.CompanyUpdated, eventbus.CompanyDeleted,
		eventbus.ContactCreated, eventbus.ContactUpdated, eventbus.ContactDeleted,
		eventbus.SponsorCreated, eventbus.SponsorUpdated, eventbus.SponsorDeleted,
	} {
		bus.Subscribe(evt, pushSvc.HandleEvent)
	}

	// Notion inbound: the webhook receiver (handler.NotionWebhookHandler)
	// publishes NotionChanged for any page it can identify;
	// syncengine.PullService reconciles that page's current state against
	// whichever internal entity the sync ledger links it to, applying the
	// change locally or — if the local side changed too — routing it to
	// the conflict queue instead of guessing a winner.
	pullSvc := syncengine.NewPullService(db, cipher, fieldMappingRepo, integrationRepo, syncedEntityRepo, syncHistoryRepo, adapters)
	bus.Subscribe(eventbus.NotionChanged, pullSvc.HandleEvent)

	mux := asynq.NewServeMux()
	handlers := worker.NewHandlers(logger, db, cipher, syncRunRepo, registryCfg, bus)
	worker.RegisterHandlers(mux, handlers)

	stopScheduler := worker.StartPeriodicResync(db, syncRunRepo, cfg, logger)
	defer stopScheduler()

	stopRetention := worker.StartActivityRetention(db, cfg.AuditLogRetentionDays, logger)
	defer stopRetention()

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
