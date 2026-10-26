package handler

import (
	"net/http"
	"strconv"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

type Storage interface {
	SetGaugeValue(name string, value float64)
	SetCounterValue(name string, value int64)
}

type UpdateMetricsHandle struct {
	storage Storage
}

func NewUpdateMetricsHandle(storage Storage) UpdateMetricsHandle {
	return UpdateMetricsHandle{storage: storage}
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
