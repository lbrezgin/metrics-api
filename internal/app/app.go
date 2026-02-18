// Package app wires HTTP router with configuration and runs the HTTP server.
//
// It exports a constructor New and a Start method, which starts the server and
// blocks until it stops.
package app

import (
	"log/slog"
	"net/http"

	"github.com/lbrezgin/telemetry/internal/config"
)

type app struct {
	router http.Handler
	cfg    *config.ServerConfig
}

func (a *app) Start(stop <-chan struct{}) error {
	srv := &http.Server{
		Addr:    a.cfg.Addr,
		Handler: a.router,
	}

	slog.Info(
		"running server...",
		"address", a.cfg.Addr,
		slog.Group(
			"log",
			"level", a.cfg.LogCfg.Level,
			"type", a.cfg.LogCfg.Type,
			"output", a.cfg.LogCfg.Output,
		),
		slog.Group(
			"persister",
			"store_interval", *a.cfg.PersisterCfg.StoreInterval,
			"file_storage_path", a.cfg.PersisterCfg.FileStoragePath,
			"restore", *a.cfg.PersisterCfg.Restore,
		),
	)

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-stop:
		slog.Info("stopping server...")
		_ = srv.Close()
		return nil

	case err := <-errCh:
		return err
	}
}

func New(router http.Handler, cfg *config.ServerConfig) *app {
	return &app{
		router: router,
		cfg:    cfg,
	}
}
