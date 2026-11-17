package repository

import models "github.com/fgugnin23/go-metrics/internal/model"

type MemStorage struct {
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gaugeMetrics:   make(map[string]float64),
		counterMetrics: make(map[string]int64),
	}
}

func (storage *MemStorage) SetGaugeValue(name string, value float64) {
	storage.gaugeMetrics[name] = value
}

func (storage *MemStorage) SetCounterValue(name string, value int64) {
	storage.counterMetrics[name] += value
}

func (storage *MemStorage) GetGaugeValue(name string) (float64, bool) {
	value, ok := storage.gaugeMetrics[name]
	return value, ok
}

func (storage *MemStorage) GetCounterValue(name string) (int64, bool) {
	value, ok := storage.counterMetrics[name]
	return value, ok
}

func (storage *MemStorage) GetGaugeMetrics() map[string]float64 {
	return storage.gaugeMetrics
}

func (storage *MemStorage) GetCounterMetrics() map[string]int64 {
	return storage.counterMetrics
}

func (storage *MemStorage) Restore(metrics []models.Metrics) error {
	if err := validateSnapshot(metrics); err != nil {
		return err
	}
	gauges := make(map[string]float64)
	counters := make(map[string]int64)
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			gauges[metric.ID] = *metric.Value
		case models.Counter:
			counters[metric.ID] = *metric.Delta
		}
	}
	storage.gaugeMetrics = gauges
	storage.counterMetrics = counters
	return nil
}
