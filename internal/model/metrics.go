package model

import (
	"errors"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

var (
	ErrInvalidMetric = errors.New("invalid metric")
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func (m *Metrics) Validate() error {
	if m == nil {
		return ErrInvalidMetric
	}

	if m.ID == "" {
		return ErrInvalidMetric
	}
	switch m.MType {
	case Counter:
		if m.Delta == nil || m.Value != nil {
			return ErrInvalidMetric
		}
	case Gauge:
		if m.Value == nil || m.Delta != nil {
			return ErrInvalidMetric
		}
	default:
		return ErrInvalidMetric
	}
	return nil
}
