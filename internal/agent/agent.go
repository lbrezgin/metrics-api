// Package agent implements a telemetry agent that periodically collects
// runtime metrics and reports them to the server via HTTP.
//
// The agent is driven by two independent intervals that are stored in AgentConfig:
//   - pollInterval  — how often metrics are collected
//   - reportInterval — how often collected metrics are sent to the server
//
// Metrics are fetched from a collector implementation and transmitted in the
// format expected by the server API.
package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/lbrezgin/telemetry/internal/agent/stats"
	"github.com/lbrezgin/telemetry/internal/config"
	"github.com/lbrezgin/telemetry/internal/model"
)

const (
	contentType = "Content-Type"
	contentEnc  = "Content-Encoding"

	contentTypeJSON = "application/json"
	gzipFormat      = "gzip"
)

var (
	errUnknownMetricType = errors.New("unknown metric type")
)

// collector defines the contract for a metrics source used by the agent.
// The implementation is responsible for gathering system/runtime statistics
// and returning them as a StatsStorage map.
type collector interface {
	CollectMetrics() stats.StatsStorage
}

// agent periodically collects metrics using a collector and sends them
// to the telemetry server using the configured HTTP client.
type agent struct {
	stats  collector
	client *resty.Client
	cfg    *config.AgentConfig
}

// Start starts the agent loop. It listens on the stop channel and
// terminates once a value is received.
//
// Metrics are collected according to pollInterval and reported based on
// reportInterval. Report and poll intervals set in AgentConfig.
func (a *agent) Start(stop <-chan struct{}) error {
	pollTicker := time.NewTicker(intToSec(a.cfg.PollInterval))
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(intToSec(a.cfg.ReportInterval))
	defer reportTicker.Stop()

	metrics := a.stats.CollectMetrics()

	log.Printf(
		"running agent...\nserver address %s\npoll interval %ds\nreport interval %ds\n",
		a.cfg.Addr,
		a.cfg.PollInterval,
		a.cfg.ReportInterval,
	)

	for {
		select {
		case <-stop:
			return nil
		case <-pollTicker.C:
			metrics = a.stats.CollectMetrics()
			log.Println("metrics were successfully collected")
		case <-reportTicker.C:
			err := a.sendMetrics(metrics)
			if err != nil {
				fmt.Printf("sending metrics: %v\n", err)
			} else {
				log.Println("metrics were successfully sent")
			}
		}
	}
}

// sendMetrics iterates through the collected metrics and sends each metric
// to the server using an HTTP POST request. The endpoint URL is constructed
// according to the metric type.
func (a *agent) sendMetrics(metrics stats.StatsStorage) error {
	if metrics == nil {
		return fmt.Errorf("metrics weren't collected, nothing to send")
	}

	for metricID, metricData := range metrics {
		reqURL := fmt.Sprintf("http://%s/update", a.cfg.Addr)

		if metricData.Type != model.Gauge && metricData.Type != model.Counter {
			return errUnknownMetricType
		}

		metricBytes, err := json.Marshal(toMetric(metricID, metricData))
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		var b bytes.Buffer
		writer := gzip.NewWriter(&b)
		if _, err = writer.Write(metricBytes); err != nil {
			return fmt.Errorf("compressing request body: %w", err)
		}
		if err = writer.Close(); err != nil {
			return fmt.Errorf("closing gzip writer: %w", err)
		}

		resp, err := a.client.R().
			SetHeader(contentType, contentTypeJSON).
			SetHeader(contentEnc, gzipFormat).
			SetBody(b.Bytes()).
			Post(reqURL)

		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("status code: %d", resp.StatusCode())
		}
	}
	return nil
}

// NewAgent constructs a new agent instance configured with a metrics
// collector and AgentConfig that contains polling and reporting intervals,
// server address (to where metrics are sent).
func NewAgent(col collector, clt *resty.Client, cfg *config.AgentConfig) *agent {
	return &agent{
		stats:  col,
		client: clt,
		cfg:    cfg,
	}
}

func intToSec(i int) time.Duration {
	return time.Second * time.Duration(i)
}

func toMetric(id string, stat stats.Stat) *model.Metrics {
	m := &model.Metrics{
		ID:    id,
		MType: stat.Type,
	}

	switch stat.Type {
	case model.Gauge:
		m.Value = &stat.FltVal
	case model.Counter:
		m.Delta = &stat.IntVal
	}
	return m
}
