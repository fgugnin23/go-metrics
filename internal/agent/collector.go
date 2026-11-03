package agent

import (
	"encoding/json"
	"errors"
	"runtime"
)

// collectRuntimeMetrics заполняет значения метрик текущей статистикой рантайма.
func collectRuntimeMetrics(metrics []GaugeMetric) error {
	runtimeMemStats := &runtime.MemStats{}
	runtime.ReadMemStats(runtimeMemStats)
	rawMetrics, marshalErr := json.Marshal(*runtimeMemStats)

	if marshalErr != nil {
		return errors.New("Error when making json from runtime metrics")
	}

	metricsMap := make(map[string]any)
	unmarshalErr := json.Unmarshal(rawMetrics, &metricsMap)

	if unmarshalErr != nil {
		return errors.New("Error when unmarshalling metrics into a map")
	}

	for i := range metrics {
		if metricValue, ok := metricsMap[metrics[i].name].(float64); ok {
			metrics[i].value = metricValue
		}
	}

	return nil
}
