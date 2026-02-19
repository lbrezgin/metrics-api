package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/lbrezgin/telemetry/internal/config"
	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/repository"
)

type Repo struct {
	db  *sql.DB
	cfg *config.Repo
}

func New(db *sql.DB, cfg *config.Repo) (*Repo, error) {
	switch {
	case db == nil:
		return nil, repository.ErrNilDB
	case cfg == nil:
		return nil, repository.ErrNilConfig
	}
	return &Repo{
		db:  db,
		cfg: cfg,
	}, nil
}

func (r *Repo) RunMigrations() error {
	m, err := migrate.New(r.cfg.MigrationsPath, r.cfg.DSN)
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func (r *Repo) UpsertTx(ctx context.Context, metrics []model.Metrics) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				slog.Info("failed to rollback", "error", rbErr)
			}
		}
	}()

	for _, m := range metrics {
		if vErr := m.Validate(); vErr != nil {
			slog.Info("invalid metric", "error", vErr, "id", m.ID, "mtype", m.MType)
			return vErr
		}

		query, qErr := insertResolveQuery(&m)
		if qErr != nil {
			return qErr
		}

		if _, execErr := tx.ExecContext(ctx, query, m.ID, m.MType, m.Delta, m.Value, m.Hash); execErr != nil {
			slog.Info("failed to execute query", "error", execErr)
			return execErr
		}
	}

	return tx.Commit()
}

func (r *Repo) Ping(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Upsert(ctx context.Context, metric *model.Metrics) error {
	if metric == nil {
		return repository.ErrNilMetric
	}
	query, err := insertResolveQuery(metric)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, query, metric.ID, metric.MType, metric.Delta, metric.Value, metric.Hash)
	if err != nil {
		return fmt.Errorf("failed to upsert metric: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}
	if count == 0 {
		return repository.ErrNoRowsWereAffected
	}
	return nil
}

func (r *Repo) Find(ctx context.Context, id, mtype string) (*model.Metrics, error) {
	query := `SELECT id, mtype, delta, value, hash FROM metrics WHERE id = $1 AND mtype = $2`
	row := r.db.QueryRowContext(ctx, query, id, mtype)
	var metric model.Metrics

	if err := row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value, &metric.Hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrMetricNotFound
		}
		return nil, fmt.Errorf("scan metric %s/%s: %w", id, mtype, err)
	}
	return &metric, nil
}

func (r *Repo) List(ctx context.Context) ([]model.Metrics, error) {
	query := `SELECT * FROM metrics;`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fetch metrics: %w", err)
	}
	defer rows.Close()

	var metrics []model.Metrics
	for rows.Next() {
		var m model.Metrics
		if err := rows.Scan(&m.ID, &m.MType, &m.Delta, &m.Value, &m.Hash); err != nil {
			return nil, fmt.Errorf("scan metric row: %w", err)
		}
		metrics = append(metrics, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating over metrics: %w", err)
	}
	return metrics, nil
}

func (r *Repo) Restore(ctx context.Context, metrics []model.Metrics) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	for _, metric := range metrics {
		query, _ := insertResolveQuery(&metric)
		_, err = tx.ExecContext(
			ctx,
			query,
			metric.ID, metric.MType, metric.Delta, metric.Value, metric.Hash,
		)
		if err != nil {
			if errRollback := tx.Rollback(); errRollback != nil {
				return fmt.Errorf("transaction failed: %w, rollback failed: %w", err, errRollback)
			}
			return fmt.Errorf("failed to make transaction: %w", err)
		}
	}
	return tx.Commit()
}

func insertResolveQuery(metric *model.Metrics) (string, error) {
	if metric == nil {
		return "", repository.ErrNilMetric
	}

	var query string
	switch metric.MType {
	case model.Counter:
		query = `
			INSERT INTO metrics (id, mtype, delta, value, hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT(id, mtype)
			DO UPDATE SET delta = metrics.delta + EXCLUDED.delta;
		`
	case model.Gauge:
		query = `
			INSERT INTO metrics (id, mtype, delta, value, hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT(id, mtype)
			DO UPDATE SET value = EXCLUDED.value;
		`
	default:
		return "", model.ErrInvalidMetric
	}
	return query, nil
}
