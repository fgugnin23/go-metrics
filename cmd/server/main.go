package main

import (
	"net/http"
	"strconv"
)

type MemStorage struct {
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
}

func (storage MemStorage) setGaugeValue(name string, value float64) {
	storage.gaugeMetrics[name] = value
	println(storage.gaugeMetrics[name])
}

func (storage MemStorage) setCounterValue(name string, value int64) {
	storage.counterMetrics[name] += value
	println(storage.counterMetrics[name])
}

func (storage MemStorage) getGaugeValue(name string) float64 {
	return storage.gaugeMetrics[name]
}

func (storage MemStorage) getCounterValue(name string) int64 {
	return storage.counterMetrics[name]
}

type updateMetricsHandle struct {
	storage MemStorage
}

type getMetricsHandle struct {
	storage MemStorage
}

func (handle updateMetricsHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricType := r.PathValue("metric_type")
	metricName := r.PathValue("metric_name")
	rawMetricValue := r.PathValue("metric_value")

	if metricType != "gauge" && metricType != "counter" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var gaugeMetricValue float64
	var counterMetricValue int64
	var err error

	switch metricType {
	case "gauge":
		gaugeMetricValue, err = strconv.ParseFloat(rawMetricValue, 64)
	case "counter":
		counterMetricValue, err = strconv.ParseInt(rawMetricValue, 10, 64)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch metricType {
	case "gauge":
		handle.storage.setGaugeValue(metricName, gaugeMetricValue)
	case "counter":
		handle.storage.setCounterValue(metricName, counterMetricValue)
	}
}

func (handle getMetricsHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricType := r.PathValue("metric_type")
	metricName := r.PathValue("metric_name")

	if metricType != "gauge" && metricType != "counter" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// switch metricType {
	// case "gauge":
	// 	handle.storage.setGaugeValue(metricName, gaugeMetricValue)
	// case "counter":
	// 	handle.storage.setCounterValue(metricName, counterMetricValue)
	// }
}

func main() {
	storage := MemStorage{
		gaugeMetrics:   make(map[string]float64),
		counterMetrics: make(map[string]int64),
	}

	http.Handle(
		"POST /update/{metric_type}/{metric_name}/{metric_value}",
		updateMetricsHandle{storage},
	)
	http.Handle(
		"GET /value/{metric_type}/{metric_name}",
		getMetricsHandle{storage},
	)
	println("started server on 8080")
	err := http.ListenAndServe(`:8080`, nil)
	if err != nil {
		panic(err)
	}
}
