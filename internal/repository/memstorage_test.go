package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_SetGaugeValue_Overwrites(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGaugeValue("Alloc", 1.5)
	value, ok := storage.GetGaugeValue("Alloc")
	require.True(t, ok)
	assert.Equal(t, 1.5, value)

	storage.SetGaugeValue("Alloc", 2.25)
	value, ok = storage.GetGaugeValue("Alloc")
	require.True(t, ok)
	assert.Equal(t, 2.25, value, "gauge должен перезаписываться, а не накапливаться")
}

func TestMemStorage_SetCounterValue_Accumulates(t *testing.T) {
	storage := NewMemStorage()

	storage.SetCounterValue("PollCount", 2)
	storage.SetCounterValue("PollCount", 3)

	value, ok := storage.GetCounterValue("PollCount")
	require.True(t, ok)
	assert.Equal(t, int64(5), value, "counter должен накапливаться")
}

func TestMemStorage_SeparateNamespaces(t *testing.T) {
	storage := NewMemStorage()

	storage.SetGaugeValue("x", 1.5)
	storage.SetCounterValue("x", 7)

	gauge, ok := storage.GetGaugeValue("x")
	require.True(t, ok)
	assert.Equal(t, 1.5, gauge)

	counter, ok := storage.GetCounterValue("x")
	require.True(t, ok)
	assert.Equal(t, int64(7), counter)

	_, ok = storage.GetGaugeValue("unknown")
	assert.False(t, ok)
}
