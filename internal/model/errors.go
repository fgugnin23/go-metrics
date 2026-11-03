package models

import "errors"

var (
	ErrInvalidMetricType  = errors.New("invalid metric type")
	ErrInvalidMetricName  = errors.New("metric name is empty")
	ErrInvalidMetricValue = errors.New("metric value is missing")
	ErrMetricNotFound     = errors.New("metric not found")
)
