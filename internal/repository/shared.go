package repository

import "errors"

var (
	ErrNilMetric            = errors.New("nil metric")
	ErrNilDB                = errors.New("nil db")
	ErrNilConfig            = errors.New("nil repo config")
	ErrNoRowsWereAffected   = errors.New("no rows affected")
	ErrMetricNotFound       = errors.New("metric not found")
	ErrAffectedRowsMismatch = errors.New("affected rows count is different from expected")
)
