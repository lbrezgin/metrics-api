// Package testutil provides helper functions for tests, such as
// creating metrics and converting values to pointers.
package testutil

import "github.com/lbrezgin/telemetry/internal/model"

// F64ptr is a helper function to get pointer of float64.
func F64ptr(v float64) *float64 {
	return &v
}

// I64ptr is a helper function to get pointer of int64.
func I64ptr(d int64) *int64 {
	return &d
}

// GaugeMetric is a helper function for creating metric of gauge type.
func GaugeMetric(id string, v float64) *model.Metrics {
	return &model.Metrics{
		ID:    id,
		MType: model.Gauge,
		Value: F64ptr(v),
	}
}

// CounterMetric is a helper function for creating metric of counter type.
func CounterMetric(id string, d int64) *model.Metrics {
	return &model.Metrics{
		ID:    id,
		MType: model.Counter,
		Delta: I64ptr(d),
	}
}
