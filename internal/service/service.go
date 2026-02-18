// Package service contains the business logic for working with metrics.
//
// The service layer orchestrates operations on metrics independently from the
// underlying storage implementation. It exposes high-level operations such as
// setting gauge values, incrementing counter metrics and retrieving the list
// of stored metrics. The actual persistence details are hidden behind the
// repository interface.
package service

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/repository"
)

var (
	ErrGaugeHasNilValue   = errors.New("gauge metric has nil value")
	ErrCounterHasNilDelta = errors.New("counter metric has nil delta")
	ErrUnknownMetricType  = errors.New("unknown metric type")
)

// storage defines operations required by the metrics service.
// It allows the service to remain independent from the actual storage backend.
type storage interface {
	Find(string, string) (model.Metrics, error)
	Save(*model.Metrics)
	List() []model.Metrics
}

// metricsService implements business logic for working with metrics on
// top of a storage. It can set and increment metrics. If metric that
// should be updated doesn't exist, it will create new metric.
type metricsService struct {
	repo storage
}

func (s *metricsService) GetGaugeValue(id string) (float64, error) {
	metric, err := s.repo.Find(id, model.Gauge)
	if err != nil {
		return 0, err
	}
	if metric.Value == nil {
		return 0, ErrGaugeHasNilValue
	}
	return *metric.Value, nil
}

func (s *metricsService) GetCounterDelta(id string) (int64, error) {
	metric, err := s.repo.Find(id, model.Counter)
	if err != nil {
		return 0, err
	}
	if metric.Delta == nil {
		return 0, ErrCounterHasNilDelta
	}
	return *metric.Delta, nil
}

// Set assigns a new value to a gauge metric.
// If the metric does not exist yet, it is created automatically.
func (s *metricsService) Set(id, metricType string, val float64) {
	metric, err := s.repo.Find(id, metricType)
	if err != nil {
		metric.ID = id
		metric.MType = metricType
		metric.Value = &val
		s.repo.Save(&metric)
		return
	}
	metric.Value = &val
	s.repo.Save(&metric)
	*metric.Value = 123
}

// Increment increases the value of a counter metric by the given delta.
// If the metric does not exist yet, it is created automatically with
// the initial value equal to delta.
func (s *metricsService) Increment(id, metricType string, delta int64) {
	metric, err := s.repo.Find(id, metricType)
	if err != nil {
		metric.ID = id
		metric.MType = metricType
		metric.Delta = &delta
		s.repo.Save(&metric)
		return
	}
	*metric.Delta += delta
	s.repo.Save(&metric)

}

// GetVal finds metric with given id and type and returns
// its value in a string format.
func (s *metricsService) GetVal(id, metricType string) (string, error) {
	metric, err := s.repo.Find(id, metricType)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) {
			return "", repository.ErrMetricNotFound
		}
		return "", fmt.Errorf("repo failure: %w", err)
	}
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return "", fmt.Errorf("ID: %s: %w", id, ErrGaugeHasNilValue)
		}
		return strconv.FormatFloat(*metric.Value, 'f', -1, 64), nil
	case model.Counter:
		if metric.Delta == nil {
			return "", fmt.Errorf("ID: %s: %w", id, ErrCounterHasNilDelta)
		}
		return strconv.FormatInt(*metric.Delta, 10), nil
	default:
		return "", fmt.Errorf("%q: %w", metricType, ErrUnknownMetricType)
	}
}

// List returns all stored metrics as value copies.
// The returned slice must not be used to mutate the repository state.
func (s *metricsService) List() []model.Metrics {
	return s.repo.List()
}

// NewMetricsService constructs a service instance on top of the provided
// storage implementation.
func NewMetricsService(repo storage) *metricsService {
	return &metricsService{
		repo: repo,
	}
}
