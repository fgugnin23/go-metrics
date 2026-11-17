package repository

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

type FileStorage struct{ path string }

func NewFileStorage(path string) *FileStorage { return &FileStorage{path: path} }

func (storage *FileStorage) Load() ([]models.Metrics, error) {
	data, err := os.ReadFile(storage.path)
	if os.IsNotExist(err) {
		return []models.Metrics{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read metrics: %w", err)
	}
	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("decode metrics: %w", err)
	}
	if metrics == nil {
		return nil, fmt.Errorf("metrics must be a JSON array")
	}
	if err := validateSnapshot(metrics); err != nil {
		return nil, err
	}
	return metrics, nil
}

func (storage *FileStorage) Save(metrics []models.Metrics) error {
	if metrics == nil {
		metrics = []models.Metrics{}
	}
	if err := validateSnapshot(metrics); err != nil {
		return err
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("encode metrics: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(storage.path), ".metrics-*.tmp")
	if err != nil {
		return fmt.Errorf("create metrics temporary file: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write metrics: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync metrics: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close metrics: %w", err)
	}
	if err := os.Rename(file.Name(), storage.path); err != nil {
		return fmt.Errorf("replace metrics: %w", err)
	}
	return nil
}

func validateSnapshot(metrics []models.Metrics) error {
	seen := make(map[[2]string]bool, len(metrics))
	for _, metric := range metrics {
		if metric.ID == "" {
			return models.ErrInvalidMetricName
		}
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil || metric.Delta != nil || math.IsNaN(*metric.Value) || math.IsInf(*metric.Value, 0) {
				return fmt.Errorf("invalid gauge %q", metric.ID)
			}
		case models.Counter:
			if metric.Delta == nil || metric.Value != nil {
				return fmt.Errorf("invalid counter %q", metric.ID)
			}
		default:
			return models.ErrInvalidMetricType
		}
		key := [2]string{metric.MType, metric.ID}
		if seen[key] {
			return fmt.Errorf("duplicate metric %s/%s", metric.MType, metric.ID)
		}
		seen[key] = true
	}
	return nil
}
