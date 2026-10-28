package agent

import (
	"strconv"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

func buildMetricURL(baseURL, metricType, name, value string) string {
	return baseURL + "/update/" + metricType + "/" + name + "/" + value
}

func gaugeURL(baseURL string, metric GaugeMetric) string {
	return buildMetricURL(baseURL, models.Gauge, metric.name, strconv.FormatFloat(metric.value, 'f', -1, 64))
}

func counterURL(baseURL string, metric CounterMetric) string {
	return buildMetricURL(baseURL, models.Counter, metric.name, strconv.FormatInt(metric.value, 10))
}
