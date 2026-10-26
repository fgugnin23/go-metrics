package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectRuntimeMetrics_FillsKnownNames(t *testing.T) {
	metrics := []GaugeMetric{
		{name: "Alloc"},
		{name: "NotARuntimeField"},
		{name: "Sys"},
	}

	collectRuntimeMetrics(metrics)

	require.Len(t, metrics, 3, "длина слайса не должна меняться")
	assert.Equal(t, "Alloc", metrics[0].name, "порядок не должен меняться")
	assert.Equal(t, "NotARuntimeField", metrics[1].name)
	assert.Equal(t, "Sys", metrics[2].name)

	assert.Positive(t, metrics[0].value, "Alloc всегда больше нуля")
	assert.Zero(t, metrics[1].value, "неизвестное поле рантайма остаётся нулевым")
	assert.Positive(t, metrics[2].value, "Sys всегда больше нуля")
}

func TestCollectRuntimeMetrics_EmptyInput(t *testing.T) {
	assert.NotPanics(t, func() { collectRuntimeMetrics(nil) })
	assert.NotPanics(t, func() { collectRuntimeMetrics([]GaugeMetric{}) })
}
