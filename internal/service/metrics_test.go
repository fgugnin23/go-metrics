package service

import (
	"testing"

	models "github.com/fgugnin23/go-metrics/internal/model"
	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsService_UpdateAndGet(t *testing.T) {
	t.Parallel()

	service := NewMetricsService(repository.NewMemStorage())
	require.NoError(t, service.UpdateMetric(gauge("metric", 1.5)))
	require.NoError(t, service.UpdateMetric(gauge("metric", 0)))
	require.NoError(t, service.UpdateMetric(counter("metric", 2)))
	require.NoError(t, service.UpdateMetric(counter("metric", 3)))
	require.NoError(t, service.UpdateMetric(counter("metric", 0)))

	gaugeMetric, err := service.GetMetric(models.Gauge, "metric")
	require.NoError(t, err)
	assert.Equal(t, gauge("metric", 0), gaugeMetric, "gauge перезаписывается, ноль является допустимым значением")

	counterMetric, err := service.GetMetric(models.Counter, "metric")
	require.NoError(t, err)
	assert.Equal(t, counter("metric", 5), counterMetric, "counter накапливается независимо от gauge с тем же именем")
}

func TestMetricsService_UpdateRejectsInvalidMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		metric models.Metrics
		err    error
	}{
		{name: "unknown type", metric: models.Metrics{ID: "metric", MType: "unknown"}, err: models.ErrInvalidMetricType},
		{name: "empty gauge name", metric: gauge("", 1), err: models.ErrInvalidMetricName},
		{name: "empty counter name", metric: counter("", 1), err: models.ErrInvalidMetricName},
		{name: "missing gauge value", metric: models.Metrics{ID: "metric", MType: models.Gauge}, err: models.ErrInvalidMetricValue},
		{name: "missing counter delta", metric: models.Metrics{ID: "metric", MType: models.Counter}, err: models.ErrInvalidMetricValue},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			storage := repository.NewMemStorage()
			storage.SetGaugeValue("metric", 10)
			storage.SetCounterValue("metric", 20)
			service := NewMetricsService(storage)

			assert.ErrorIs(t, service.UpdateMetric(test.metric), test.err)
			assert.Equal(t, map[string]float64{"metric": 10}, storage.GetGaugeMetrics())
			assert.Equal(t, map[string]int64{"metric": 20}, storage.GetCounterMetrics())
		})
	}
}

func TestMetricsService_GetErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		metricType string
		metricName string
		err        error
	}{
		{name: "unknown type", metricType: "unknown", metricName: "metric", err: models.ErrInvalidMetricType},
		{name: "empty gauge name", metricType: models.Gauge, err: models.ErrInvalidMetricName},
		{name: "empty counter name", metricType: models.Counter, err: models.ErrInvalidMetricName},
		{name: "missing gauge", metricType: models.Gauge, metricName: "missing", err: models.ErrMetricNotFound},
		{name: "missing counter", metricType: models.Counter, metricName: "missing", err: models.ErrMetricNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			service := NewMetricsService(repository.NewMemStorage())
			metric, err := service.GetMetric(test.metricType, test.metricName)

			assert.ErrorIs(t, err, test.err)
			assert.Empty(t, metric)
		})
	}
}

func TestMetricsService_ListMetrics(t *testing.T) {
	t.Parallel()

	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)
	assert.Empty(t, service.ListMetrics())

	want := []models.Metrics{
		gauge("Alloc", 1.5),
		gauge("Sys", 10),
		counter("PollCount", 3),
		counter("Errors", 0),
	}
	for _, metric := range want {
		require.NoError(t, service.UpdateMetric(metric))
	}

	metrics := service.ListMetrics()
	assert.ElementsMatch(t, want, metrics)

	for _, metric := range metrics {
		if metric.Value != nil {
			*metric.Value = -1
		}
		if metric.Delta != nil {
			*metric.Delta = -1
		}
	}
	assert.ElementsMatch(t, want, service.ListMetrics(), "возвращённые значения не должны давать доступ к изменению хранилища")
}

func gauge(name string, value float64) models.Metrics {
	return models.Metrics{ID: name, MType: models.Gauge, Value: &value}
}

func counter(name string, value int64) models.Metrics {
	return models.Metrics{ID: name, MType: models.Counter, Delta: &value}
}
