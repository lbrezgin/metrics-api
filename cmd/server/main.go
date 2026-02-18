package main

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/lbrezgin/telemetry/internal/app"
	"github.com/lbrezgin/telemetry/internal/config"
	"github.com/lbrezgin/telemetry/internal/handler"
	"github.com/lbrezgin/telemetry/internal/logger"
	"github.com/lbrezgin/telemetry/internal/persister"
	"github.com/lbrezgin/telemetry/internal/repository"
	"github.com/lbrezgin/telemetry/internal/router"
	"github.com/lbrezgin/telemetry/internal/service"
	"github.com/lbrezgin/telemetry/internal/squeeze"
)

func main() {
	cfg := &config.ServerConfig{
		LogCfg:       &config.LogConfig{},
		PersisterCfg: &config.PersisterConfig{},
	}

	// Load environment variables from .env if present.
	if err := godotenv.Load(); err != nil {
		log.Printf("env file failed to load: %v\n", err)
	}

	if err := config.LoadConfig(cfg); err != nil {
		log.Fatal(err)
	}

	// Apply flag values to empty config fields.
	// Defaults are used when flags are not provided.
	parseFlags(cfg)

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg *config.ServerConfig) error {
	repo := repository.NewMemStorage()
	metricsService := service.NewMetricsService(repo)
	metricsHandler := handler.NewMetricsHandler(metricsService)

	middlewares := []func(http.Handler) http.Handler{
		logger.LoggingMiddleware,
		middleware.StripSlashes,
		squeeze.GzipMiddleware,
	}

	closer, err := logger.Init(cfg.LogCfg)
	if err != nil {
		return err
	}

	if closer != nil {
		defer closer.Close()
	}

	p, err := persister.New(cfg.PersisterCfg, repo)
	if err != nil {
		return err
	}

	if *cfg.PersisterCfg.StoreInterval == 0 {
		middlewares = append(middlewares, persister.PersisterMiddleware(p))
	}

	routeR := router.New(metricsHandler, middlewares...)

	stop := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		close(stop)
	}()

	if err = p.Restore(); err != nil {
		if errors.Is(err, persister.ErrFileDoesNotExist) {
			slog.Info("file for restore doesn't exist, restore skipped.")
		} else {
			return err
		}
	}

	errCh := make(chan error, 1)
	go p.Run(stop, errCh)

	api := app.New(
		routeR,
		cfg,
	)

	go func() {
		if perr := <-errCh; perr != nil {
			slog.Error("persister failed", "error", perr)
			close(stop)
		}
	}()

	if err := api.Start(stop); err != nil {
		return err
	}
	return nil
}
