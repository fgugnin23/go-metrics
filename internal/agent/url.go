package agent

import (
	"strconv"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

func buildMetricURL(metricType, name, value string) string {
	return serverBaseURL + "/update/" + metricType + "/" + name + "/" + value
}

func gaugeURL(metric GaugeMetric) string {
	return buildMetricURL(models.Gauge, metric.name, strconv.FormatFloat(metric.value, 'f', -1, 64))
}

func counterURL(metric CounterMetric) string {
	return buildMetricURL(models.Counter, metric.name, strconv.FormatInt(metric.value, 10))
}
