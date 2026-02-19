package service

import (
	"context"
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

type Storage interface {
	Find(ctx context.Context, id, mtype string) (*model.Metrics, error)
	Upsert(ctx context.Context, metric *model.Metrics) error
	List(ctx context.Context) ([]model.Metrics, error)
	Ping(ctx context.Context) error
	UpsertTx(ctx context.Context, metric []model.Metrics) error
}

type metricsService struct {
	repo Storage
}

func (s *metricsService) UpsertMetrics(ctx context.Context, metrics []model.Metrics) error {
	if err := s.repo.UpsertTx(ctx, metrics); err != nil {
		return err
	}
	return nil
}

func (s *metricsService) PingContext(ctx context.Context) error {
	if err := s.repo.Ping(ctx); err != nil {
		return err
	}
	return nil
}

func (s *metricsService) GetGaugeValue(ctx context.Context, id string) (float64, error) {
	metric, err := s.repo.Find(ctx, id, model.Gauge)
	if err != nil {
		return 0, err
	}
	if metric.Value == nil {
		return 0, ErrGaugeHasNilValue
	}
	return *metric.Value, nil
}

func (s *metricsService) GetCounterDelta(ctx context.Context, id string) (int64, error) {
	metric, err := s.repo.Find(ctx, id, model.Counter)
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
func (s *metricsService) Set(ctx context.Context, id, metricType string, val float64) error {
	metric := &model.Metrics{
		ID:    id,
		MType: metricType,
		Value: &val,
	}
	if err := s.repo.Upsert(ctx, metric); err != nil {
		return err
	}
	return nil
}

// Increment increases the value of a counter metric by the given delta.
// If the metric does not exist yet, it is created automatically with
// the initial value equal to delta.
func (s *metricsService) Increment(ctx context.Context, id, metricType string, delta int64) error {
	metric, err := s.repo.Find(ctx, id, metricType)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) {
			metric = &model.Metrics{}
			metric.ID = id
			metric.MType = metricType
			metric.Delta = &delta
		} else {
			return err
		}
	} else {
		*metric.Delta += delta
	}
	if err = s.repo.Upsert(ctx, metric); err != nil {
		return err
	}
	return nil
}

// GetVal finds metric with given id and type and returns
// its value in a string format.
func (s *metricsService) GetVal(ctx context.Context, id, metricType string) (string, error) {
	metric, err := s.repo.Find(ctx, id, metricType)
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
func (s *metricsService) List(ctx context.Context) ([]model.Metrics, error) {
	result, err := s.repo.List(ctx)
	if err != nil {
		return []model.Metrics{}, fmt.Errorf("list service: %w", err)
	}
	return result, nil
}

func NewMetricsService(repo Storage) *metricsService {
	return &metricsService{
		repo: repo,
	}
}
