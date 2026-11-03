package service

import models "github.com/fgugnin23/go-metrics/internal/model"

type Storage interface {
	SetGaugeValue(name string, value float64)
	SetCounterValue(name string, value int64)
	GetGaugeValue(name string) (float64, bool)
	GetCounterValue(name string) (int64, bool)
	GetGaugeMetrics() map[string]float64
	GetCounterMetrics() map[string]int64
}

type MetricsService struct {
	storage Storage
}

func NewMetricsService(storage Storage) *MetricsService {
	return &MetricsService{storage: storage}
}

func (service *MetricsService) UpdateMetric(metric models.Metrics) error {
	if err := validateMetric(metric.MType, metric.ID); err != nil {
		return err
	}

	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return models.ErrInvalidMetricValue
		}
		service.storage.SetGaugeValue(metric.ID, *metric.Value)
	case models.Counter:
		if metric.Delta == nil {
			return models.ErrInvalidMetricValue
		}
		service.storage.SetCounterValue(metric.ID, *metric.Delta)
	}

	return nil
}

func (service *MetricsService) GetMetric(metricType, name string) (models.Metrics, error) {
	if err := validateMetric(metricType, name); err != nil {
		return models.Metrics{}, err
	}

	metric := models.Metrics{ID: name, MType: metricType}
	switch metricType {
	case models.Gauge:
		value, ok := service.storage.GetGaugeValue(name)
		if !ok {
			return models.Metrics{}, models.ErrMetricNotFound
		}
		metric.Value = &value
	case models.Counter:
		value, ok := service.storage.GetCounterValue(name)
		if !ok {
			return models.Metrics{}, models.ErrMetricNotFound
		}
		metric.Delta = &value
	}

	return metric, nil
}

func (service *MetricsService) ListMetrics() []models.Metrics {
	gauges := service.storage.GetGaugeMetrics()
	counters := service.storage.GetCounterMetrics()
	metrics := make([]models.Metrics, 0, len(gauges)+len(counters))

	for name, value := range gauges {
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Gauge, Value: &value})
	}
	for name, value := range counters {
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Counter, Delta: &value})
	}

	return metrics
}

func validateMetric(metricType, name string) error {
	if metricType != models.Gauge && metricType != models.Counter {
		return models.ErrInvalidMetricType
	}
	if name == "" {
		return models.ErrInvalidMetricName
	}

	return nil
}
