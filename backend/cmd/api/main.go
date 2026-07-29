package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/database"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/pkg/apierror"
	"github.com/timeless/backend/internal/router"
	"github.com/timeless/backend/internal/worker"
)

func main() {
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

	// Always migrate on boot so fresh Render/Neon databases get schema.
	// GORM AutoMigrate is additive and safe for repeated runs.
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("failed to run auto-migration: %v", err)
	}

	rdb, err := database.NewRedis(cfg)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	workerClient, err := worker.NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to create worker client: %v", err)
	}
	defer workerClient.Close()

	app := fiber.New(fiber.Config{
		AppName:      "Timeless API",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit:    10 * 1024 * 1024,
		// Concurrency bounds simultaneous in-flight connections. fasthttp's
		// default (256*1024) is effectively unlimited for a single small
		// instance — a slow-loris-style flood could open connections far
		// past what the process/DB pool can actually service before any
		// per-route rate limit even gets a chance to run. This is a coarse
		// backstop underneath RedisRateLimiter, not a replacement for it.
		Concurrency:  16384,
		ErrorHandler: globalErrorHandler,
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.SecurityHeaders(cfg))
	app.Use(logger.New(middleware.RequestLogger()))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "Idempotency-Key"},
		ExposeHeaders:    []string{"X-API-Version", "Idempotency-Replayed"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	router.Setup(app, db, rdb, cfg, workerClient)

	go func() {
		addr := ":" + cfg.Port
		log.Printf("Timeless API starting on %s (env: %s)", addr, cfg.Environment)
		if err := app.Listen(addr, fiber.ListenConfig{
			DisableStartupMessage: cfg.Environment == "production",
		}); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	_ = rdb.Close()
	log.Println("server exited cleanly")
}

func globalErrorHandler(c fiber.Ctx, err error) error {
	return apierror.WriteError(c, err)
}
