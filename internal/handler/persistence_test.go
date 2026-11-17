package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	models "github.com/fgugnin23/go-metrics/internal/model"
	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatesSaveBeforeResponse(t *testing.T) {
	for _, jsonRoute := range []bool{false, true} {
		name := "path"
		if jsonRoute {
			name = "json"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metrics.json")
			file := repository.NewFileStorage(path)
			metrics := service.NewPersistentMetricsService(repository.NewMemStorage(), file, true)
			router := NewRouter(metrics, zerolog.Nop())
			update := func(kind, value string) *httptest.ResponseRecorder {
				url := "/update/" + kind + "/metric/" + value
				body := ""
				if jsonRoute {
					url = "/update/"
					field := "value"
					if kind == models.Counter {
						field = "delta"
					}
					body = `{"id":"metric","type":"` + kind + `","` + field + `":` + value + `}`
				}
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, url, strings.NewReader(body)))
				return response
			}
			for _, kind := range []string{models.Counter, models.Gauge} {
				for _, value := range []string{"2", "3", "0"} {
					response := update(kind, value)
					require.Equal(t, http.StatusOK, response.Code)
					saved, err := file.Load()
					require.NoError(t, err)
					assert.ElementsMatch(t, metrics.ListMetrics(), saved)
					if jsonRoute {
						assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
					} else {
						assert.Empty(t, response.Body.String())
					}
				}
			}
			before, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, update(models.Counter, "bad").Code)
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			// Prevent file replacement and check that the in-memory update survives HTTP 500.
			require.NoError(t, os.Remove(path))
			require.NoError(t, os.Mkdir(path, 0700))
			assert.Equal(t, http.StatusInternalServerError, update(models.Counter, "7").Code)
			metric, err := metrics.GetMetric(models.Counter, "metric")
			require.NoError(t, err)
			assert.Equal(t, int64(12), *metric.Delta)
		})
	}
}
