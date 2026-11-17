package repository

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	models "github.com/fgugnin23/go-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorageRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	file := NewFileStorage(path)
	zero, value := 0.0, 1.25
	zeroCount, maxCount, minCount := int64(0), int64(math.MaxInt64), int64(math.MinInt64)
	want := []models.Metrics{
		{ID: "shared", MType: models.Gauge, Value: &value},
		{ID: "zero", MType: models.Gauge, Value: &zero},
		{ID: "zero", MType: models.Counter, Delta: &zeroCount},
		{ID: "shared", MType: models.Counter, Delta: &maxCount},
		{ID: "min", MType: models.Counter, Delta: &minCount},
	}
	require.NoError(t, file.Save(want))
	got, err := file.Load()
	require.NoError(t, err)
	assert.Equal(t, want, got)
	storage := NewMemStorage()
	storage.SetCounterValue("shared", 10)
	storage.SetGaugeValue("old", 10)
	require.NoError(t, storage.Restore(got))
	require.NoError(t, storage.Restore(got))
	assert.Equal(t, map[string]float64{"shared": 1.25, "zero": 0}, storage.GetGaugeMetrics())
	assert.Equal(t, map[string]int64{"shared": math.MaxInt64, "min": math.MinInt64, "zero": 0}, storage.GetCounterMetrics())
	for _, empty := range [][]models.Metrics{nil, {}} {
		require.NoError(t, file.Save(empty))
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data))
		got, err = file.Load()
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	}
}

func TestFileStorageLoadErrors(t *testing.T) {
	for _, data := range []string{
		"", "{", "{}", "null", "[] []", "[null]",
		`[{"id":"x","type":"unknown","value":1}]`,
		`[{"id":"","type":"gauge","value":1}]`,
		`[{"id":"x","type":"gauge"}]`,
		`[{"id":"x","type":"counter","delta":null}]`,
		`[{"id":"x","type":"gauge","value":1,"delta":1}]`,
		`[{"id":"x","type":"counter","value":1,"delta":1}]`,
		`[{"id":"x","type":"counter","delta":1.5}]`,
		`[{"id":"x","type":"counter","delta":9223372036854775808}]`,
		`[{"id":"x","type":"gauge","value":1e999}]`,
		`[{"id":"x","type":"counter","delta":1},{"id":"x","type":"counter","delta":2}]`,
		`[{"id":"x","type":"gauge","value":1},{"id":"x","type":"gauge","value":2}]`,
	} {
		t.Run(data, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metrics.json")
			require.NoError(t, os.WriteFile(path, []byte(data), 0600))
			got, err := NewFileStorage(path).Load()
			require.Error(t, err)
			assert.Nil(t, got)
		})
	}
	t.Run("missing", func(t *testing.T) {
		got, err := NewFileStorage(filepath.Join(t.TempDir(), "missing.json")).Load()
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("directory", func(t *testing.T) {
		_, err := NewFileStorage(t.TempDir()).Load()
		require.Error(t, err)
	})
	t.Run("unreadable", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "metrics.json")
		require.NoError(t, os.WriteFile(path, []byte("[]"), 0000))
		t.Cleanup(func() { _ = os.Chmod(path, 0600) })
		if os.Geteuid() == 0 {
			t.Skip("root can read files without read permission")
		}
		_, err := NewFileStorage(path).Load()
		require.Error(t, err)
	})
}

func TestFailedSavePreservesFileAndCleansTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")
	file := NewFileStorage(path)
	require.NoError(t, file.Save(nil))
	badValue := math.Inf(1)
	require.Error(t, file.Save([]models.Metrics{{ID: "x", MType: models.Gauge, Value: &badValue}}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data))
	target := filepath.Join(dir, "directory")
	require.NoError(t, os.Mkdir(target, 0700))
	require.Error(t, NewFileStorage(target).Save(nil))
	matches, err := filepath.Glob(filepath.Join(dir, ".metrics-*.tmp"))
	require.NoError(t, err)
	assert.Empty(t, matches)
	require.Error(t, NewFileStorage(filepath.Join(dir, "missing", "metrics.json")).Save(nil))
}

func TestRestoreDoesNotPartiallyApplyInvalidSnapshot(t *testing.T) {
	storage := NewMemStorage()
	storage.SetCounterValue("existing", 42)
	value := 1.0
	err := storage.Restore([]models.Metrics{{ID: "valid", MType: models.Gauge, Value: &value}, {ID: "invalid", MType: models.Counter}})
	require.Error(t, err)
	assert.Empty(t, storage.GetGaugeMetrics())
	assert.Equal(t, map[string]int64{"existing": 42}, storage.GetCounterMetrics())
}
