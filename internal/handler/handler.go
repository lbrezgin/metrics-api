// Package handler contains HTTP handlers that expose the metrics service
// over HTTP. The handlers are responsible for translating incoming HTTP
// requests into service calls and serializing the service responses back
// to HTTP clients.
package handler

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/repository"
	"github.com/lbrezgin/telemetry/internal/service"
)

const (
	contentType = "Content-Type"

	contentTypeJSON = "application/json"
)

// svc defines the business operations required by the HTTP layer.
// It is satisfied by the metrics service implementation and allows the
// handler package to remain independent from the service internals.
type svc interface {
	Set(id, metricType string, val float64)
	Increment(id, metricType string, delta int64)
	List() []model.Metrics
	GetVal(id, metricType string) (string, error)
	GetGaugeValue(id string) (float64, error)
	GetCounterDelta(id string) (int64, error)
}

// metricsHandler implements HTTP handlers for metrics operations.
// It delegates the actual business logic to the injected service.
type metricsHandler struct {
	svc svc
}

// Update handles HTTP requests for updating a single metric.
// Depending on the metric type, it either sets a gauge value or
// increments a counter. Supported metric types are defined in the
// model package. The metric data is taken from the URL path.
//
// On success, Update responds with HTTP 200 OK.
func (h *metricsHandler) Update(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	metricType := r.PathValue("type")
	metricValue := r.PathValue("value")
	metricName := r.PathValue("name")

	switch metricType {
	case model.Gauge:
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "bad value given", http.StatusBadRequest)
			return
		}

		h.svc.Set(metricName, model.Gauge, val)
	case model.Counter:
		val, err := strconv.Atoi(metricValue)
		if err != nil {
			http.Error(w, "bad value given", http.StatusBadRequest)
			return
		}

		nVal := int64(val)
		h.svc.Increment(metricName, model.Counter, nVal)

	default:
		http.Error(w, service.ErrUnknownMetricType.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *metricsHandler) UpdateFromBody(w http.ResponseWriter, r *http.Request) {
	if writeContentTypeError(w, r.Header.Get(contentType), contentTypeJSON) {
		return
	}

	var metric model.Metrics
	if !successfulDecoding(w, r, &metric) {
		return
	}

	if metric.ID == "" || metric.MType == "" {
		writeJSONError(w, http.StatusBadRequest, "incorrect data")
		return
	}

	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			writeJSONError(w, http.StatusBadRequest, "incorrect data")
			return
		}
		h.svc.Set(metric.ID, metric.MType, *metric.Value)
	case model.Counter:
		if metric.Delta == nil {
			writeJSONError(w, http.StatusBadRequest, "incorrect data")
			return
		}
		h.svc.Increment(metric.ID, metric.MType, *metric.Delta)
	default:
		slog.Info("unknown metric type", "metric_type", metric.MType)
		writeJSONError(w, http.StatusBadRequest, service.ErrUnknownMetricType.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *metricsHandler) ShowFromBody(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(contentType, contentTypeJSON)

	if writeContentTypeError(w, r.Header.Get(contentType), contentTypeJSON) {
		return
	}

	var metric model.Metrics
	if !successfulDecoding(w, r, &metric) {
		return
	}

	switch metric.MType {
	case model.Gauge:
		val, err := h.svc.GetGaugeValue(metric.ID)
		if err != nil {
			if errors.Is(err, repository.ErrMetricNotFound) {
				writeJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metric.Value = &val
	case model.Counter:
		delta, err := h.svc.GetCounterDelta(metric.ID)
		if err != nil {
			if errors.Is(err, repository.ErrMetricNotFound) {
				writeJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		metric.Delta = &delta
	default:
		writeJSONError(w, http.StatusBadRequest, service.ErrUnknownMetricType.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	// successfulEncoding writes http.Error in ResponseWriter if any error
	// occurs during encoding.
	_ = successfulEncoding(w, metric)
}

// List returns all stored metrics in a HTML format.
// It always responds with HTTP 200 OK.
func (h *metricsHandler) List(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("internal/static/index.html")
	if err != nil {
		http.Error(w, "template not found: "+err.Error(), http.StatusInternalServerError)
		return
	}

	metrics := h.svc.List()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, metrics); err != nil {
		http.Error(w, "execute error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Show returns accumulated metric's value in a text format with
// HTTP 200 OK.
func (h *metricsHandler) Show(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricType != model.Counter && metricType != model.Gauge {
		http.Error(w, service.ErrUnknownMetricType.Error(), http.StatusBadRequest)
		return
	}

	val, err := h.svc.GetVal(metricName, metricType)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) {
			http.Error(w, repository.ErrMetricNotFound.Error(), http.StatusNotFound)
			return
		}

		log.Printf("internal error: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(val))
}

// NewMetricsHandler constructs a metricsHandler instance bound to the
// provided service implementation.
func NewMetricsHandler(svc svc) *metricsHandler {
	return &metricsHandler{
		svc: svc,
	}
}
