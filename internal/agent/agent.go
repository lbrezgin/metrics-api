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

	batchSize = 50
)

type collector interface {
	CollectMetrics() stats.StatsStorage
}

type agent struct {
	stats  collector
	client *resty.Client
	cfg    *config.Agent
}

func NewAgent(col collector, clt *resty.Client, cfg *config.Agent) *agent {
	return &agent{
		stats:  col,
		client: clt,
		cfg:    cfg,
	}
}

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

func (a *agent) sendMetrics(metrics stats.StatsStorage) error {
	if metrics == nil {
		return fmt.Errorf("metrics weren't collected, nothing to send")
	}

	reqURL := fmt.Sprintf("http://%s/updates", a.cfg.Addr)
	metricsToSend := make([]*model.Metrics, 0, batchSize)

	for metricID, metricData := range metrics {
		m := toMetric(metricID, metricData)
		if err := m.Validate(); err != nil {
			log.Printf("skip invalid metric: %v", err)
			continue
		}

		metricsToSend = append(metricsToSend, m)
		if len(metricsToSend) == batchSize {
			if err := a.sentReqInGzip(metricsToSend, reqURL); err != nil {
				return err
			}
			metricsToSend = metricsToSend[:0]
		}
	}

	if len(metricsToSend) > 0 {
		if err := a.sentReqInGzip(metricsToSend, reqURL); err != nil {
			return err
		}
	}
	return nil
}

func (a *agent) sentReqInGzip(metrics []*model.Metrics, reqURL string) error {
	metricsBytes, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to encode metrics: %w", err)
	}

	var b bytes.Buffer
	zw := gzip.NewWriter(&b)

	if _, err = zw.Write(metricsBytes); err != nil {
		return fmt.Errorf("failed to compress metrics: %w", err)
	}
	if err = zw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
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
	return nil
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
