package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/lbrezgin/telemetry/internal/app"
	"github.com/lbrezgin/telemetry/internal/persister"
	"github.com/lbrezgin/telemetry/internal/repository/memstorage"
	"github.com/lbrezgin/telemetry/internal/router"
	"github.com/lbrezgin/telemetry/internal/squeeze"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/lbrezgin/telemetry/internal/handler"
	"github.com/lbrezgin/telemetry/internal/service"

	"github.com/lbrezgin/telemetry/internal/config"
	"github.com/lbrezgin/telemetry/internal/logger"
	"github.com/lbrezgin/telemetry/internal/repository/postgres"
)

type storage interface {
	service.Storage
	persister.StateStore
}

func main() {
	cfg := &config.Server{
		RepoCfg:      &config.Repo{},
		LogCfg:       &config.Log{},
		PersisterCfg: &config.Persister{},
	}
	// Load environment variables from .env if present.
	if err := godotenv.Load(); err != nil {
		log.Printf("load env file: %v\n", err)
	}
	if err := config.Load(cfg); err != nil {
		log.Fatal(err)
	}
	// Apply flag values to empty config fields.
	// Defaults are used when flags are not provided.
	parseFlags(cfg)

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg *config.Server) error {
	// Logger:
	// Initialize and configure slog.
	closer, err := logger.Init(cfg.LogCfg)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	// Database:
	// Open database connection.
	db, err := sql.Open(cfg.RepoCfg.Driver, cfg.RepoCfg.DSN)
	if err != nil {
		return fmt.Errorf("connecting to a database: %w", err)
	}
	defer db.Close()

	// Repository:
	// The in-memory storage "memstorage" is used If DSN not provided.
	var repo storage
	if len(cfg.RepoCfg.DSN) == 0 {
		repo = memstorage.New()
		slog.Info("using in-memory storage")
	} else {
		var pgRepo *postgres.Repo
		pgRepo, err = postgres.New(db, cfg.RepoCfg)
		if err != nil {
			return err
		}
		// Run migrations.
		if err = pgRepo.RunMigrations(); err != nil {
			return err
		}
		slog.Info("migrations successfully ran")
		repo = pgRepo
		slog.Info("using postgres storage")
	}

	// Services, Handlers, Middlewares.
	metricsService := service.NewMetricsService(repo)
	metricsHandler := handler.NewMetricsHandler(metricsService)
	middlewares := []func(http.Handler) http.Handler{
		logger.LoggingMiddleware,
		middleware.StripSlashes,
		squeeze.GzipMiddleware,
	}

	// Create persister and add middleware if the persister snapshot mode is synchronous.
	p, err := persister.New(cfg.PersisterCfg, repo)
	if err != nil {
		return err
	}
	if *cfg.PersisterCfg.StoreInterval == 0 {
		middlewares = append(middlewares, persister.PersisterMiddleware(p))
	}

	// Router:
	// Create the router.
	routeR := router.New(metricsHandler, middlewares...)

	// Persister communication mechanism:
	// Channels to allow persister and application stop in error or os.Interrupt.
	stop := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	errCh := make(chan error, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		close(stop)
	}()
	go func() {
		if perr := <-errCh; perr != nil {
			slog.Error("persister failed", "error", perr)
			close(stop)
		}
	}()

	// Restore metrics from the file. Skip if file does not exist.
	if err = p.Restore(); err != nil {
		if errors.Is(err, persister.ErrFileDoesNotExist) {
			slog.Info("file for restore doesn't exist, restore skipped.")
		} else {
			return err
		}
	}

	// Run persister in a separate goroutine.
	go p.Run(stop, errCh)

	// Server:
	// Create and start the server.
	api := app.New(routeR, cfg)
	if err = api.Start(stop); err != nil {
		return err
	}
	return nil
}
