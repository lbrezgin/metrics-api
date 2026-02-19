package memstorage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/repository"
)

type dbMemStorage []*model.Metrics

type MemStorage struct {
	db dbMemStorage
	mu sync.RWMutex
}

func New() *MemStorage {
	return &MemStorage{
		db: dbMemStorage{},
	}
}

func (m *MemStorage) UpsertTx(ctx context.Context, metrics []model.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range metrics {
		met := &metrics[i]

		if err := met.Validate(); err != nil {
			slog.Info("invalid metric", "error", err, "id", met.ID, "mtype", met.MType)
			return err
		}

		if met.MType == model.Counter {
			for _, existing := range m.db {
				if existing.ID == met.ID && existing.MType == met.MType {
					if met.Delta != nil && existing.Delta != nil {
						*met.Delta += *existing.Delta
					}
					break
				}
			}
		}

		cloned := cloneMetric(met)

		updated := false
		for j, existing := range m.db {
			if existing.ID == met.ID && existing.MType == met.MType {
				m.db[j] = cloned
				updated = true
				break
			}
		}
		if !updated {
			m.db = append(m.db, cloned)
		}
	}

	return nil
}

func (m *MemStorage) Ping(_ context.Context) error {
	return nil
}

func (m *MemStorage) Upsert(_ context.Context, metric *model.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if metric == nil {
		return repository.ErrNilMetric
	}

	cloned := cloneMetric(metric)

	for i, existing := range m.db {
		if existing.ID == metric.ID && existing.MType == metric.MType {
			m.db[i] = cloned
			return nil
		}
	}
	m.db = append(m.db, cloned)
	return nil
}

func (m *MemStorage) Find(_ context.Context, id, mtype string) (*model.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, metric := range m.db {
		if metric == nil {
			return nil, repository.ErrNilMetric
		}
		if metric.ID == id && metric.MType == mtype {
			return cloneMetric(metric), nil
		}
	}
	return nil, fmt.Errorf(
		"%w: id %s, metric type %s",
		repository.ErrMetricNotFound, id, mtype,
	)
}

func (m *MemStorage) List(_ context.Context) ([]model.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copyDB := make([]model.Metrics, 0, len(m.db))
	for _, metric := range m.db {
		if metric != nil {
			copyDB = append(copyDB, *cloneMetric(metric))
		}
	}
	return copyDB, nil
}

func (m *MemStorage) Restore(_ context.Context, metrics []model.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.db = make(dbMemStorage, 0, len(metrics))

	for _, metric := range metrics {
		cloned := cloneMetric(&metric)
		m.db = append(m.db, cloned)
	}
	return nil
}

// Create deep copy of metric. Can return nil.
func cloneMetric(origM *model.Metrics) *model.Metrics {
	if origM == nil {
		return nil
	}
	metricCopy := *origM
	if origM.Value != nil {
		v := *origM.Value
		metricCopy.Value = &v
	}
	if origM.Delta != nil {
		d := *origM.Delta
		metricCopy.Delta = &d
	}
	return &metricCopy
}
