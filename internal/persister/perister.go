package persister

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lbrezgin/telemetry/internal/config"
	"github.com/lbrezgin/telemetry/internal/model"
)

var (
	ErrCantBeNil        = errors.New("can't be nill")
	ErrFileDoesNotExist = errors.New("file doesn't exist")
)

type StateStore interface {
	List(ctx context.Context) ([]model.Metrics, error)
	Restore(ctx context.Context, metrics []model.Metrics) error
}

type persister struct {
	cfg   *config.Persister
	store StateStore
}

func (p *persister) Run(stop <-chan struct{}, errCh chan<- error) {
	if *p.cfg.StoreInterval == 0 {
		slog.Info("synchronous snapshot mode")
		return
	}

	t := time.NewTicker(intToSec(*p.cfg.StoreInterval))
	defer t.Stop()

	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if err := p.snapshot(); err != nil {
				if errCh != nil {
					errCh <- err
				}
				return
			}
			slog.Info("snapshot created")
		}
	}
}

func (p *persister) Restore() error {
	if !*p.cfg.Restore {
		slog.Info("no restore")
		return nil
	}

	file, err := os.Open(p.cfg.FileStoragePath)
	if err != nil {
		return ErrFileDoesNotExist
	}
	defer file.Close()

	var metrics []model.Metrics
	decoder := json.NewDecoder(file)

	if err = decoder.Decode(&metrics); err != nil {
		return err
	}

	if err = p.store.Restore(context.TODO(), metrics); err != nil {
		return err
	}
	slog.Info("data successfully restored")
	return nil
}

func (p *persister) snapshot() error {
	file, err := os.OpenFile(p.cfg.FileStoragePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	metrics, err := p.store.List(context.TODO())
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(metrics); err != nil {
		return err
	}
	return nil
}

func intToSec(i int) time.Duration {
	return time.Second * time.Duration(i)
}

func validateConfig(cfg *config.Persister) error {
	if cfg == nil {
		return ErrCantBeNil
	}
	if cfg.Restore == nil || cfg.StoreInterval == nil {
		return ErrCantBeNil
	}
	return nil

}

func New(cfg *config.Persister, store StateStore) (*persister, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &persister{
		cfg:   cfg,
		store: store,
	}, nil
}

func PersisterMiddleware(p *persister) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)

			if r.Method == http.MethodPost &&
				(r.URL.Path == "/update" || strings.HasPrefix(r.URL.Path, "/update/")) {
				if err := p.snapshot(); err != nil {
					slog.Error(
						"failed to create snapshot",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
					)
				} else {
					slog.Info("snapshot created")
				}
			}
		})
	}
}
