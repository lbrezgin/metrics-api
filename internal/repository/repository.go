// Package repository provides an in-memory implementation of a metrics storage.
//
// The storage keeps metrics internally as pointers, but exposes values to callers,
// so that external code cannot mutate internal state without using Save.
package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lbrezgin/telemetry/internal/model"
)

// dbMemStorage represents the in-memory storage container.
type dbMemStorage []*model.Metrics

var (
	// ErrMetricNotFound is returned when a metric with the given id and type does not exist.
	ErrMetricNotFound = errors.New("metric not found")
)

// memStorage is an in-memory implementation of metrics repository.
type memStorage struct {
	db dbMemStorage
	mu sync.RWMutex
}

// Find searches for a metric by id and type and returns its value copy.
// If not found, ErrMetricNotFound is returned.
func (m *memStorage) Find(id, metricType string) (model.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, metric := range m.db {
		if metric.ID == id && metric.MType == metricType {
			return cloneMetric(metric), nil
		}
	}
	return model.Metrics{}, fmt.Errorf(
		"%w: id %s, metric type %s",
		ErrMetricNotFound, id, metricType,
	)
}

// Save stores a metric. If a metric with the same id and type already exists,
// it is replaced. Otherwise, a new metric is appended.
func (m *memStorage) Save(newMetric *model.Metrics) {
	m.mu.Lock()
	defer m.mu.Unlock()

	copy := cloneMetric(newMetric)

	for i, metric := range m.db {
		if metric.ID == newMetric.ID && metric.MType == newMetric.MType {
			m.db[i] = &copy
			return
		}
	}

	m.db = append(m.db, &copy)
}

// List returns a slice of metric value copies.
// Callers cannot mutate the internal storage through the returned slice.
func (m *memStorage) List() []model.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copyDB := make([]model.Metrics, 0, len(m.db))
	for _, metric := range m.db {
		copyDB = append(copyDB, cloneMetric(metric))
	}
	return copyDB
}

func (m *memStorage) Restore(metrics []model.Metrics) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.db = make(dbMemStorage, 0, len(metrics))

	for _, metric := range metrics {
		copy := cloneMetric(&metric)
		m.db = append(m.db, &copy)
	}
}

// cloneMetric is a helper function that creates deep copy of
// metric.
func cloneMetric(origM *model.Metrics) model.Metrics {
	metricCopy := *origM
	if origM.Value != nil {
		v := *origM.Value
		metricCopy.Value = &v
	}
	if origM.Delta != nil {
		d := *origM.Delta
		metricCopy.Delta = &d
	}
	return metricCopy
}

// NewMemStorage creates an empty in-memory storage.
func NewMemStorage() *memStorage {
	return &memStorage{
		db: dbMemStorage{},
	}
}
