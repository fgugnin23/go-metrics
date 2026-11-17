package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

func (handler MetricsHandler) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metric, err := decodeMetric(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := handler.service.UpdateMetric(metric); err != nil {
		handler.writeServiceError(w, err)
		return
	}

	metric, err = handler.service.GetMetric(metric.MType, metric.ID)
	if err != nil {
		handler.writeServiceError(w, err)
		return
	}

	handler.writeMetricJSON(w, metric)
}

func (handler MetricsHandler) GetJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metric, err := decodeMetric(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metric, err = handler.service.GetMetric(metric.MType, metric.ID)
	if err != nil {
		handler.writeServiceError(w, err)
		return
	}

	handler.writeMetricJSON(w, metric)
}

func decodeMetric(body io.Reader) (models.Metrics, error) {
	var metric models.Metrics
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&metric); err != nil {
		return models.Metrics{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return models.Metrics{}, errors.New("request body must contain a single JSON object")
	}
	return metric, nil
}

func (handler MetricsHandler) writeMetricJSON(w http.ResponseWriter, metric models.Metrics) {
	body, err := json.Marshal(metric)
	if err != nil {
		handler.logger.Info().Err(err).Msg("error when encoding metric")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
