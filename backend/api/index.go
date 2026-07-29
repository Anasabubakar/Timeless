package handler

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/timeless/backend/internal/config"
	"github.com/timeless/backend/internal/database"
	"github.com/timeless/backend/internal/middleware"
	"github.com/timeless/backend/internal/pkg/apierror"
	"github.com/timeless/backend/internal/router"
	"github.com/timeless/backend/internal/worker"
)

var (
	once sync.Once
	app  *fiber.App
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(initApp)

	resp, err := app.Test(r, fiber.TestConfig{
		Timeout:       30 * time.Second,
		FailOnTimeout: false,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func initApp() {
	cfg, err := config.Load()
	if err != nil {
		panic("config: " + err.Error())
	}

	db, err := database.NewPostgres(cfg)
	if err != nil {
		panic("postgres: " + err.Error())
	}

	if err := database.AutoMigrate(db); err != nil {
		panic("migrate: " + err.Error())
	}

	rdb, err := database.NewRedis(cfg)
	if err != nil {
		panic("redis: " + err.Error())
	}

	workerClient, _ := worker.NewClient(cfg)

	app = fiber.New(fiber.Config{
		AppName:      "Timeless API",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
		BodyLimit:    10 * 1024 * 1024,
		// See cmd/api/main.go for why Concurrency is set explicitly;
		// lower here since each serverless invocation has less memory/CPU
		// to work with than the standalone binary.
		Concurrency:  4096,
		ErrorHandler: globalErrorHandler,
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.SecurityHeaders(cfg))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	router.Setup(app, db, rdb, cfg, workerClient)
}

func globalErrorHandler(c fiber.Ctx, err error) error {
	return apierror.WriteError(c, err)
}
