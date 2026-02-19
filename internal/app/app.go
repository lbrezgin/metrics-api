package app

import (
	"log/slog"
	"net/http"

	"github.com/lbrezgin/telemetry/internal/config"
)

type app struct {
	router http.Handler
	cfg    *config.Server
}

func New(router http.Handler, cfg *config.Server) *app {
	return &app{
		router: router,
		cfg:    cfg,
	}
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
		slog.Group(
			"repo",
			"driver", a.cfg.RepoCfg.Driver,
			"migrations_path", a.cfg.RepoCfg.MigrationsPath,
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
