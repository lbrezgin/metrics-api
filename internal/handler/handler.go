package handler

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/repository"
	"github.com/lbrezgin/telemetry/internal/service"
)

const (
	contentType = "Content-Type"

	contentTypeJSON = "application/json"
)

type svc interface {
	Set(ctx context.Context, id, metricType string, val float64) error
	Increment(ctx context.Context, id, metricType string, delta int64) error
	List(ctx context.Context) ([]model.Metrics, error)
	GetVal(ctx context.Context, id, metricType string) (string, error)
	GetGaugeValue(ctx context.Context, id string) (float64, error)
	GetCounterDelta(ctx context.Context, id string) (int64, error)
	PingContext(ctx context.Context) error
	UpsertMetrics(ctx context.Context, metrics []model.Metrics) error
}

func NewMetricsHandler(svc svc) *metricsHandler {
	return &metricsHandler{
		svc: svc,
	}
}

type metricsHandler struct {
	svc svc
}

func (h *metricsHandler) Ping(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := h.svc.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Debug(err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *metricsHandler) Updates(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if writeContentTypeError(w, r.Header.Get(contentType), contentTypeJSON) {
		return
	}

	var metrics []model.Metrics
	if !tryDecodeJSONRequest(w, r, &metrics) {
		return
	}

	if err := h.svc.UpsertMetrics(r.Context(), metrics); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

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
		if err = h.svc.Set(r.Context(), metricName, model.Gauge, val); err != nil {
			slog.Error(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	case model.Counter:
		val, err := strconv.Atoi(metricValue)
		if err != nil {
			http.Error(w, "bad value given", http.StatusBadRequest)
			return
		}
		nVal := int64(val)
		if err = h.svc.Increment(r.Context(), metricName, model.Counter, nVal); err != nil {
			slog.Error(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, service.ErrUnknownMetricType.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *metricsHandler) UpdateFromBody(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if writeContentTypeError(w, r.Header.Get(contentType), contentTypeJSON) {
		return
	}

	var metric model.Metrics
	if !tryDecodeJSONRequest(w, r, &metric) {
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
		if err := h.svc.Set(r.Context(), metric.ID, metric.MType, *metric.Value); err != nil {
			slog.Error(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	case model.Counter:
		if metric.Delta == nil {
			writeJSONError(w, http.StatusBadRequest, "incorrect data")
			return
		}
		if err := h.svc.Increment(r.Context(), metric.ID, metric.MType, *metric.Delta); err != nil {
			slog.Error(err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	default:
		slog.Info("unknown metric type", "metric_type", metric.MType)
		writeJSONError(w, http.StatusBadRequest, service.ErrUnknownMetricType.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *metricsHandler) ShowFromBody(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set(contentType, contentTypeJSON)

	if writeContentTypeError(w, r.Header.Get(contentType), contentTypeJSON) {
		return
	}

	var metric model.Metrics
	if !tryDecodeJSONRequest(w, r, &metric) {
		return
	}

	switch metric.MType {
	case model.Gauge:
		val, err := h.svc.GetGaugeValue(r.Context(), metric.ID)
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
		delta, err := h.svc.GetCounterDelta(r.Context(), metric.ID)
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
	// tryEncodeJSONRequest writes http.Error in ResponseWriter if any error
	// occurs during encoding.
	_ = tryEncodeJSONRequest(w, metric)
}

func (h *metricsHandler) List(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("internal/static/index.html")
	if err != nil {
		slog.Error("failed to parse template", "err", err, "handler", "List")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	metrics, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("failed to get metrics", "err", err, "handler", "List")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.Execute(w, metrics); err != nil {
		slog.Error("failed to execute template", "err", err, "handler", "List")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *metricsHandler) Show(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricType != model.Counter && metricType != model.Gauge {
		http.Error(w, service.ErrUnknownMetricType.Error(), http.StatusBadRequest)
		return
	}

	val, err := h.svc.GetVal(r.Context(), metricName, metricType)
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
