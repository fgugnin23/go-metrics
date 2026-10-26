package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgugnin23/go-metrics/internal/repository"
)

func TestNewMetricsPageData_SortsByName(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGaugeValue("Sys", 3)
	storage.SetGaugeValue("Alloc", 1)
	storage.SetGaugeValue("HeapSys", 2)
	storage.SetCounterValue("PollCount", 1)
	storage.SetCounterValue("Errors", 2)

	data := newMetricsPageData(storage)

	assert.Equal(t, []string{"Alloc", "HeapSys", "Sys"}, rowNames(data.Gauges))
	assert.Equal(t, []string{"Errors", "PollCount"}, rowNames(data.Counters))
}

func TestNewMetricsPageData_FormatsValues(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGaugeValue("Fractional", 1.5)
	storage.SetGaugeValue("Integral", 100)
	storage.SetGaugeValue("Negative", -0.25)
	storage.SetCounterValue("PollCount", 3)

	data := newMetricsPageData(storage)

	require.Len(t, data.Gauges, 3)
	assert.Equal(t, metricRow{Name: "Fractional", Value: "1.5"}, data.Gauges[0])
	assert.Equal(t, metricRow{Name: "Integral", Value: "100"}, data.Gauges[1])
	assert.Equal(t, metricRow{Name: "Negative", Value: "-0.25"}, data.Gauges[2])

	require.Len(t, data.Counters, 1)
	assert.Equal(t, metricRow{Name: "PollCount", Value: "3"}, data.Counters[0])
}

func TestNewMetricsPageData_EmptyStorage(t *testing.T) {
	data := newMetricsPageData(repository.NewMemStorage())

	assert.Empty(t, data.Gauges)
	assert.Empty(t, data.Counters)
}

func TestRenderMetricsPage(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGaugeValue("Alloc", 1.5)
	storage.SetCounterValue("PollCount", 3)

	var page strings.Builder
	require.NoError(t, renderMetricsPage(&page, newMetricsPageData(storage)))

	body := page.String()

	assert.Contains(t, body, "<h3>Список gauge метрик</h3>")
	assert.Contains(t, body, "<h3>Список counter метрик</h3>")
	assert.Contains(t, body, "<li><b>Alloc: </b>1.5</li>")
	assert.Contains(t, body, "<li><b>PollCount: </b>3</li>")
}

func TestRenderMetricsPage_EmptyStorage(t *testing.T) {
	var page strings.Builder
	require.NoError(t, renderMetricsPage(&page, newMetricsPageData(repository.NewMemStorage())))

	body := page.String()

	assert.Contains(t, body, "<h3>Список gauge метрик</h3>")
	assert.Contains(t, body, "<h3>Список counter метрик</h3>")
	assert.NotContains(t, body, "<li>")
}

func TestRenderMetricsPage_EscapesMetricName(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGaugeValue("<script>alert(1)</script>", 1)

	var page strings.Builder
	require.NoError(t, renderMetricsPage(&page, newMetricsPageData(storage)))

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
