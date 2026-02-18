// Package stats collects and exposes runtime metrics obtained from
// runtime.MemStats. The package wraps runtime.ReadMemStats and returns
// the collected metrics as a key/value map. It also has two custom metrics
// PollCount and RandomValue that are not from runtime package.
package stats

import (
	"maps"
	"math/rand"
	"runtime"

	"github.com/lbrezgin/telemetry/internal/model"
)

// Supported metric names. These constants are used as keys in
// StatsStorage and correspond to the fields of runtime.MemStats.
const (
	alloc         = "Alloc"
	buckHashSys   = "BuckHashSys"
	frees         = "Frees"
	gCCPUFraction = "GCCPUFraction"
	gCSys         = "GCSys"
	heapAlloc     = "HeapAlloc"
	heapIdle      = "HeapIdle"
	heapInuse     = "HeapInuse"
	heapObjects   = "HeapObjects"
	heapReleased  = "HeapReleased"
	heapSys       = "HeapSys"
	lastGC        = "LastGC"
	lookups       = "Lookups"
	mCacheInuse   = "MCacheInuse"
	mCacheSys     = "MCacheSys"
	mSpanInuse    = "MSpanInuse"
	mSpanSys      = "MSpanSys"
	mallocs       = "Mallocs"
	nextGC        = "NextGC"
	numForcedGC   = "NumForcedGC"
	numGC         = "NumGC"
	otherSys      = "OtherSys"
	pauseTotalNs  = "PauseTotalNs"
	stackInuse    = "StackInuse"
	stackSys      = "StackSys"
	sys           = "Sys"
	totalAlloc    = "TotalAlloc"
)

// Custom metrics that are not from runtime.MemStats.
const (
	// Metric of type Counter that will increase for 1
	// every time metrics are collected from runtime package.
	pollCount = "PollCount"

	// Random value of type Gauge.
	randomValue = "RandomValue"
)

// StatsStorage represents a collection of runtime metrics,
// keyed by their name.
type StatsStorage map[string]Stat

// Stat describes a single runtime metric.
//
// Depending on the metric nature, the value may be stored either
// as IntVal or FltVal. Type specifies whether the metric behaves
// as a Gauge or a Counter.
type Stat struct {
	Type   string
	IntVal int64
	FltVal float64
}

// runtimeStats encapsulates a runtime.MemStats instance and the
// internal storage used to hold collected metrics.
type runtimeStats struct {
	storage StatsStorage
	source  *runtime.MemStats
}

// NewRuntimeStats creates and initializes a runtimeStats collector.
func NewRuntimeStats() *runtimeStats {
	return &runtimeStats{
		storage: make(StatsStorage),
		source:  &runtime.MemStats{},
	}
}

// CollectMetrics reads the current runtime.MemStats values and updates
// the internal metrics storage. Each call overwrites previously stored
// values and returns the updated StatsStorage map.
func (s *runtimeStats) CollectMetrics() StatsStorage {
	runtime.ReadMemStats(s.source)

	s.storage[alloc] = Stat{Type: model.Gauge, FltVal: float64(s.source.Alloc)}
	s.storage[buckHashSys] = Stat{Type: model.Gauge, FltVal: float64(s.source.BuckHashSys)}
	s.storage[frees] = Stat{Type: model.Gauge, FltVal: float64(s.source.Frees)}
	s.storage[gCCPUFraction] = Stat{Type: model.Gauge, FltVal: s.source.GCCPUFraction}
	s.storage[gCSys] = Stat{Type: model.Gauge, FltVal: float64(s.source.GCSys)}
	s.storage[heapAlloc] = Stat{Type: model.Gauge, FltVal: float64(s.source.HeapAlloc)}
	s.storage[heapIdle] = Stat{Type: model.Gauge, FltVal: float64(s.source.HeapIdle)}
	s.storage[heapInuse] = Stat{Type: model.Gauge, FltVal: float64(s.source.HeapInuse)}
	s.storage[heapObjects] = Stat{Type: model.Gauge, FltVal: float64(s.source.HeapObjects)}
	s.storage[heapReleased] = Stat{Type: model.Gauge, FltVal: float64(s.source.HeapReleased)}
	s.storage[heapSys] = Stat{Type: model.Gauge, FltVal: float64(s.source.HeapSys)}
	s.storage[lastGC] = Stat{Type: model.Gauge, FltVal: float64(s.source.LastGC)}
	s.storage[lookups] = Stat{Type: model.Gauge, FltVal: float64(s.source.Lookups)}
	s.storage[mCacheInuse] = Stat{Type: model.Gauge, FltVal: float64(s.source.MCacheInuse)}
	s.storage[mCacheSys] = Stat{Type: model.Gauge, FltVal: float64(s.source.MCacheSys)}
	s.storage[mSpanInuse] = Stat{Type: model.Gauge, FltVal: float64(s.source.MSpanInuse)}
	s.storage[mSpanSys] = Stat{Type: model.Gauge, FltVal: float64(s.source.MSpanSys)}
	s.storage[mallocs] = Stat{Type: model.Gauge, FltVal: float64(s.source.Mallocs)}
	s.storage[nextGC] = Stat{Type: model.Gauge, FltVal: float64(s.source.NextGC)}
	s.storage[numForcedGC] = Stat{Type: model.Gauge, FltVal: float64(s.source.NumForcedGC)}
	s.storage[numGC] = Stat{Type: model.Gauge, FltVal: float64(s.source.NumGC)}
	s.storage[otherSys] = Stat{Type: model.Gauge, FltVal: float64(s.source.OtherSys)}
	s.storage[pauseTotalNs] = Stat{Type: model.Gauge, FltVal: float64(s.source.PauseTotalNs)}
	s.storage[stackInuse] = Stat{Type: model.Gauge, FltVal: float64(s.source.StackInuse)}
	s.storage[stackSys] = Stat{Type: model.Gauge, FltVal: float64(s.source.StackSys)}
	s.storage[sys] = Stat{Type: model.Gauge, FltVal: float64(s.source.Sys)}
	s.storage[totalAlloc] = Stat{Type: model.Gauge, FltVal: float64(s.source.TotalAlloc)}
	s.storage[pollCount] = Stat{Type: model.Counter, IntVal: 1}
	s.storage[randomValue] = Stat{Type: model.Gauge, FltVal: (rand.Float64() * 100)}

	// Return copy of the StatsStorage to protect internal state
	// from external changing.
	return maps.Clone(s.storage)
}
