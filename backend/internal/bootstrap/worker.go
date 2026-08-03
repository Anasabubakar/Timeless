// Package bootstrap wires the background-job runtime (event subscribers,
// asynq task handlers, periodic resync, stale-run recovery, activity
// retention) shared by cmd/worker (a dedicated worker process) and
// cmd/api (which can embed the same runtime in-process — see
// config.Config.EmbedWorker). Kept as its own top-level package, not
// inside internal/worker itself, because this wiring needs to construct
// internal/service and internal/syncengine types, and internal/service
// already imports internal/worker (for *worker.Client) — putting this
// here instead of in internal/worker avoids that import cycle.
package bootstrap

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/eventbus"
	"github.com/timeless/backend/internal/integration"
	"github.com/timeless/backend/internal/mapping"
	"github.com/timeless/backend/internal/repository"
	"github.com/timeless/backend/internal/security"
	"github.com/timeless/backend/internal/service"
	"github.com/timeless/backend/internal/syncengine"
	"github.com/timeless/backend/internal/worker"
)

// WorkerRuntime bundles everything needed to run the background-job
// consumer: the asynq server/mux pair (call Server.Run(Mux) — blocking —
// or Server.Start(Mux) — non-blocking, for embedding alongside an HTTP
// server) plus the periodic background goroutines, stopped together via
// Stop.
type WorkerRuntime struct {
	Server *asynq.Server
	Mux    *asynq.ServeMux
	Bus    *eventbus.Bus

	stopScheduler func()
	stopRetention func()
}

// StopBackgroundGoroutines stops the periodic resync and retention
// goroutines. It does NOT touch the asynq server itself — Server.Run
// (blocking, used by cmd/worker) already manages its own shutdown via OS
// signals, and Server.Start (non-blocking, used when embedding this
// runtime in another process) requires the embedder to call
// Server.Shutdown() explicitly when it decides to stop, on its own
// timeline. Calling asynq's Shutdown a second time here would be
// redundant with the Run case and premature for the Start case.
func (w *WorkerRuntime) StopBackgroundGoroutines() {
	w.stopScheduler()
	w.stopRetention()
}

// NewWorkerRuntime constructs the full background-job runtime: every
// eventbus subscriber (Notion push/pull, Zapier ingest), every asynq
// task handler, and the periodic resync/retention goroutines. Identical
// regardless of whether the caller runs Server.Run (cmd/worker, blocking,
// this process's only job) or Server.Start in a goroutine alongside an
// HTTP server (cmd/api with EmbedWorker enabled).
func NewWorkerRuntime(cfg *config.Config, db *gorm.DB, logger *slog.Logger) (*WorkerRuntime, error) {
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return nil, err
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

	bus := eventbus.NewBus()

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

	pullSvc := syncengine.NewPullService(db, cipher, fieldMappingRepo, integrationRepo, syncedEntityRepo, syncHistoryRepo, adapters)
	bus.Subscribe(eventbus.NotionChanged, pullSvc.HandleEvent)

	contactRepo := repository.NewContactRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	contactSvc := service.NewContactService(contactRepo).SetBus(bus)
	zapierIngestSvc := syncengine.NewZapierIngestService(contactRepo, companyRepo, contactSvc)
	bus.Subscribe(eventbus.ZapierWebhookReceived, zapierIngestSvc.HandleEvent)

	mux := asynq.NewServeMux()
	handlers := worker.NewHandlers(logger, db, cipher, syncRunRepo, registryCfg, bus)
	worker.RegisterHandlers(mux, handlers)

	stopScheduler := worker.StartPeriodicResync(db, syncRunRepo, cfg, logger)
	stopRetention := worker.StartActivityRetention(db, cfg.AuditLogRetentionDays, logger)

	return &WorkerRuntime{
		Server:        srv,
		Mux:           mux,
		Bus:           bus,
		stopScheduler: stopScheduler,
		stopRetention: stopRetention,
	}, nil
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
