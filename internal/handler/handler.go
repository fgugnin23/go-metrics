package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

type Storage interface {
	SetGaugeValue(name string, value float64)
	SetCounterValue(name string, value int64)
	GetGaugeValue(name string) (float64, bool)
	GetCounterValue(name string) (int64, bool)
}

type MetricsHandler struct {
	storage Storage
}

func NewMetricsHandler(storage Storage) MetricsHandler {
	return MetricsHandler{storage: storage}
}

func (handler MetricsHandler) Update(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metric_type")
	metricName := chi.URLParam(r, "metric_name")
	rawMetricValue := chi.URLParam(r, "metric_value")

	if metricType != models.Gauge && metricType != models.Counter {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(rawMetricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handler.storage.SetGaugeValue(metricName, value)
	case models.Counter:
		value, err := strconv.ParseInt(rawMetricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handler.storage.SetCounterValue(metricName, value)
	}
}

func (handler MetricsHandler) Get(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metric_type")
	metricName := chi.URLParam(r, "metric_name")

	if metricType != models.Gauge && metricType != models.Counter {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var body string

	switch metricType {
	case models.Gauge:
		value, ok := handler.storage.GetGaugeValue(metricName)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body = strconv.FormatFloat(value, 'f', -1, 64)
	case models.Counter:
		value, ok := handler.storage.GetCounterValue(metricName)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body = strconv.FormatInt(value, 10)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}
