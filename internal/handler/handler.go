package handler

import (
	"net/http"
	"strconv"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

type Storage interface {
	SetGaugeValue(name string, value float64)
	SetCounterValue(name string, value int64)
	GetGaugeValue(name string) (float64, bool)
	GetCounterValue(name string) (int64, bool)
}

type UpdateMetricsHandle struct {
	storage Storage
}

func NewUpdateMetricsHandle(storage Storage) UpdateMetricsHandle {
	return UpdateMetricsHandle{storage: storage}
}

type GetMetricsHandle struct {
	storage Storage
}

func NewGetMetricsHandle(storage Storage) GetMetricsHandle {
	return GetMetricsHandle{storage: storage}
}

func (handle UpdateMetricsHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricType := r.PathValue("metric_type")
	metricName := r.PathValue("metric_name")
	rawMetricValue := r.PathValue("metric_value")

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
		handle.storage.SetGaugeValue(metricName, value)
	case models.Counter:
		value, err := strconv.ParseInt(rawMetricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handle.storage.SetCounterValue(metricName, value)
	}
}

func (handle GetMetricsHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricType := r.PathValue("metric_type")
	metricName := r.PathValue("metric_name")

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
		value, ok := handle.storage.GetGaugeValue(metricName)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body = strconv.FormatFloat(value, 'f', -1, 64)
	case models.Counter:
		value, ok := handle.storage.GetCounterValue(metricName)
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
