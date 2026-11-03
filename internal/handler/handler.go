package handler

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

type MetricsService interface {
	UpdateMetric(metric models.Metrics) error
	GetMetric(metricType, name string) (models.Metrics, error)
	ListMetrics() []models.Metrics
}

type MetricsHandler struct {
	service MetricsService
}

func NewMetricsHandler(service MetricsService) MetricsHandler {
	return MetricsHandler{service: service}
}

func (handler MetricsHandler) MainPage(w http.ResponseWriter, _ *http.Request) {
	var page bytes.Buffer

	if err := renderMetricsPage(&page, newMetricsPageData(handler.service.ListMetrics())); err != nil {
		log.Println("error when rendering main page: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(page.Bytes())
}

func (handler MetricsHandler) Update(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metric_type")
	metricName := chi.URLParam(r, "metric_name")
	rawMetricValue := chi.URLParam(r, "metric_value")

	metric := models.Metrics{ID: metricName, MType: metricType}
	switch metricType {
	case models.Gauge:
		value, err := strconv.ParseFloat(rawMetricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		metric.Value = &value
	case models.Counter:
		value, err := strconv.ParseInt(rawMetricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		metric.Delta = &value
	}

	if err := handler.service.UpdateMetric(metric); err != nil {
		writeServiceError(w, err)
	}
}

func (handler MetricsHandler) Get(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metric_type")
	metricName := chi.URLParam(r, "metric_name")

	metric, err := handler.service.GetMetric(metricType, metricName)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	var body string

	switch metric.MType {
	case models.Gauge:
		body = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	case models.Counter:
		body = strconv.FormatInt(*metric.Delta, 10)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrInvalidMetricType), errors.Is(err, models.ErrInvalidMetricValue):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, models.ErrInvalidMetricName), errors.Is(err, models.ErrMetricNotFound):
		w.WriteHeader(http.StatusNotFound)
	default:
		log.Println("unexpected error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
