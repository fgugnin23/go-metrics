package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

func TestNewMetricsPageData_SortsByName(t *testing.T) {
	data := newMetricsPageData([]models.Metrics{
		testGauge("Sys", 3),
		testGauge("Alloc", 1),
		testGauge("HeapSys", 2),
		testCounter("PollCount", 1),
		testCounter("Errors", 2),
	})

	assert.Equal(t, []string{"Alloc", "HeapSys", "Sys"}, rowNames(data.Gauges))
	assert.Equal(t, []string{"Errors", "PollCount"}, rowNames(data.Counters))
}

func TestNewMetricsPageData_FormatsValues(t *testing.T) {
	data := newMetricsPageData([]models.Metrics{
		testGauge("Fractional", 1.5),
		testGauge("Integral", 100),
		testGauge("Negative", -0.25),
		testCounter("PollCount", 3),
	})

	require.Len(t, data.Gauges, 3)
	assert.Equal(t, metricRow{Name: "Fractional", Value: "1.5"}, data.Gauges[0])
	assert.Equal(t, metricRow{Name: "Integral", Value: "100"}, data.Gauges[1])
	assert.Equal(t, metricRow{Name: "Negative", Value: "-0.25"}, data.Gauges[2])

	require.Len(t, data.Counters, 1)
	assert.Equal(t, metricRow{Name: "PollCount", Value: "3"}, data.Counters[0])
}

func TestNewMetricsPageData_EmptyStorage(t *testing.T) {
	data := newMetricsPageData(nil)

	assert.Empty(t, data.Gauges)
	assert.Empty(t, data.Counters)
}

func TestRenderMetricsPage(t *testing.T) {
	metrics := []models.Metrics{testGauge("Alloc", 1.5), testCounter("PollCount", 3)}

	var page strings.Builder
	require.NoError(t, renderMetricsPage(&page, newMetricsPageData(metrics)))

	body := page.String()

	assert.Contains(t, body, "<h3>Список gauge метрик</h3>")
	assert.Contains(t, body, "<h3>Список counter метрик</h3>")
	assert.Contains(t, body, "<li><b>Alloc: </b>1.5</li>")
	assert.Contains(t, body, "<li><b>PollCount: </b>3</li>")
}

func TestRenderMetricsPage_EmptyStorage(t *testing.T) {
	var page strings.Builder
	require.NoError(t, renderMetricsPage(&page, newMetricsPageData(nil)))

	body := page.String()

	assert.Contains(t, body, "<h3>Список gauge метрик</h3>")
	assert.Contains(t, body, "<h3>Список counter метрик</h3>")
	assert.NotContains(t, body, "<li>")
}

func TestRenderMetricsPage_EscapesMetricName(t *testing.T) {
	metrics := []models.Metrics{testGauge("<script>alert(1)</script>", 1)}

	var page strings.Builder
	require.NoError(t, renderMetricsPage(&page, newMetricsPageData(metrics)))

	body := page.String()

	assert.NotContains(t, body, "<script>alert", "имя метрики не должно попадать в разметку как есть")
	assert.Contains(t, body, "&lt;script&gt;")
}

func rowNames(rows []metricRow) []string {
	names := make([]string, 0, len(rows))

	for _, row := range rows {
		names = append(names, row.Name)
	}

	return names
}

func testGauge(name string, value float64) models.Metrics {
	return models.Metrics{ID: name, MType: models.Gauge, Value: &value}
}

func testCounter(name string, value int64) models.Metrics {
	return models.Metrics{ID: name, MType: models.Counter, Delta: &value}
}
