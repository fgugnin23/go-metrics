package handler

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/fgugnin23/go-metrics/internal/model"
	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
)

func postJSON(t *testing.T, router http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	return recorder
}

func TestJSONMetrics_UpdateAndGet(t *testing.T) {
	steps := []struct {
		name string
		body string
		want string
	}{
		{"create gauge", `{"id":"LastGC","type":"gauge","value":1744184459}`, `{"id":"LastGC","type":"gauge","value":1744184459}`},
		{"overwrite gauge", `{"id":"LastGC","type":"gauge","value":-1.25}`, `{"id":"LastGC","type":"gauge","value":-1.25}`},
		{"zero gauge", `{"id":"LastGC","type":"gauge","value":0}`, `{"id":"LastGC","type":"gauge","value":0}`},
		{"create counter", `{"id":"PollCount","type":"counter","delta":2}`, `{"id":"PollCount","type":"counter","delta":2}`},
		{"accumulate counter", `{"id":"PollCount","type":"counter","delta":3}`, `{"id":"PollCount","type":"counter","delta":5}`},
		{"zero delta", `{"id":"PollCount","type":"counter","delta":0}`, `{"id":"PollCount","type":"counter","delta":5}`},
		{"zero counter", `{"id":"PollCount","type":"counter","delta":-5}`, `{"id":"PollCount","type":"counter","delta":0}`},
		{"same name different type", `{"id":"LastGC","type":"counter","delta":7}`, `{"id":"LastGC","type":"counter","delta":7}`},
		{"large counter", `{"id":"Large","type":"counter","delta":9223372036854775807}`, `{"id":"Large","type":"counter","delta":9223372036854775807}`},
		{"exponent and irrelevant field", `{"id":"Small","type":"gauge","value":1e-7,"delta":42}`, `{"id":"Small","type":"gauge","value":0.0000001}`},
		{"whitespace", " \n{\"id\":\"Spaces\",\"type\":\"gauge\",\"value\":1}\n\t", `{"id":"Spaces","type":"gauge","value":1}`},
	}

	router := NewRouter(service.NewMetricsService(repository.NewMemStorage()), zerolog.Nop())
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			response := postJSON(t, router, "/update", step.body)
			require.Equal(t, http.StatusOK, response.Code)
			assert.JSONEq(t, step.want, response.Body.String())

			var request models.Metrics
			require.NoError(t, json.Unmarshal([]byte(step.body), &request))
			lookup, err := json.Marshal(models.Metrics{ID: request.ID, MType: request.MType})
			require.NoError(t, err)
			response = postJSON(t, router, "/value", string(lookup))
			require.Equal(t, http.StatusOK, response.Code)
			assert.JSONEq(t, step.want, response.Body.String())
		})
	}
}

func TestJSONMetrics_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{"empty body", "", http.StatusBadRequest},
		{"malformed JSON", `{"id":"Alloc"`, http.StatusBadRequest},
		{"array", `[{"id":"Alloc","type":"gauge","value":1}]`, http.StatusBadRequest},
		{"null", `null`, http.StatusBadRequest},
		{"empty object", `{}`, http.StatusBadRequest},
		{"multiple objects", `{"id":"Alloc","type":"gauge","value":1} {}`, http.StatusBadRequest},
		{"trailing garbage", `{"id":"Alloc","type":"gauge","value":1} invalid`, http.StatusBadRequest},
		{"string value", `{"id":"Alloc","type":"gauge","value":"1"}`, http.StatusBadRequest},
		{"fractional counter", `{"id":"PollCount","type":"counter","delta":1.5}`, http.StatusBadRequest},
		{"counter overflow", `{"id":"PollCount","type":"counter","delta":9223372036854775808}`, http.StatusBadRequest},
		{"gauge overflow", `{"id":"Alloc","type":"gauge","value":1e400}`, http.StatusBadRequest},
		{"numeric name", `{"id":1,"type":"gauge","value":1}`, http.StatusBadRequest},
		{"missing type", `{"id":"Alloc","value":1}`, http.StatusBadRequest},
		{"unknown type", `{"id":"Alloc","type":"unknown","value":1}`, http.StatusBadRequest},
		{"missing name", `{"type":"gauge","value":1}`, http.StatusNotFound},
	}

	for _, path := range []string{"/update", "/value"} {
		t.Run(path, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					storage := repository.NewMemStorage()
					storage.SetGaugeValue("Alloc", 10)
					storage.SetCounterValue("PollCount", 20)
					router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())

					response := postJSON(t, router, path, test.body)

					assert.Equal(t, test.status, response.Code)
					assert.Equal(t, map[string]float64{"Alloc": 10}, storage.GetGaugeMetrics())
					assert.Equal(t, map[string]int64{"PollCount": 20}, storage.GetCounterMetrics())
				})
			}
		})
	}
}

func TestUpdateJSON_RequiresValue(t *testing.T) {
	for _, body := range []string{
		`{"id":"Alloc","type":"gauge"}`,
		`{"id":"Alloc","type":"gauge","value":null}`,
		`{"id":"Alloc","type":"gauge","delta":1}`,
		`{"id":"PollCount","type":"counter"}`,
		`{"id":"PollCount","type":"counter","delta":null}`,
		`{"id":"PollCount","type":"counter","value":1}`,
	} {
		t.Run(body, func(t *testing.T) {
			storage := repository.NewMemStorage()
			router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())
			assert.Equal(t, http.StatusBadRequest, postJSON(t, router, "/update", body).Code)
			assert.Empty(t, storage.GetGaugeMetrics())
			assert.Empty(t, storage.GetCounterMetrics())
		})
	}
}

func TestGetJSON_MissingMetric(t *testing.T) {
	router := NewRouter(service.NewMetricsService(repository.NewMemStorage()), zerolog.Nop())
	for _, body := range []string{`{"id":"Missing","type":"gauge"}`, `{"id":"Missing","type":"counter"}`} {
		assert.Equal(t, http.StatusNotFound, postJSON(t, router, "/value", body).Code)
	}
}

func TestJSONMetrics_SharesStorageWithLegacyAPI(t *testing.T) {
	router := NewRouter(service.NewMetricsService(repository.NewMemStorage()), zerolog.Nop())
	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/counter/PollCount/2").Code)
	response := postJSON(t, router, "/value", `{"id":"PollCount","type":"counter","delta":999}`)
	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":"PollCount","type":"counter","delta":2}`, response.Body.String())
	require.Equal(t, http.StatusOK, postJSON(t, router, "/update", `{"id":"PollCount","type":"counter","delta":3}`).Code)
	response = do(t, router, http.MethodGet, "/value/counter/PollCount")
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "5", response.Body.String())
}

func TestJSONMetrics_WrongMethod(t *testing.T) {
	router := NewRouter(&mockService{}, zerolog.Nop())
	for _, path := range []string{"/update", "/value"} {
		assert.Equal(t, http.StatusMethodNotAllowed, do(t, router, http.MethodGet, path).Code)
	}
}

func TestJSONMetrics_ServiceFailure(t *testing.T) {
	metrics := &mockService{updateErr: errors.New("storage unavailable"), getErr: errors.New("storage unavailable")}
	router := NewRouter(metrics, zerolog.Nop())
	for _, path := range []string{"/update", "/value"} {
		response := postJSON(t, router, path, `{"id":"Alloc","type":"gauge","value":1}`)
		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Empty(t, response.Body.String())
	}
	assert.Len(t, metrics.getCalls, 1, "failed update must not read the metric")
}

func TestGetJSON_UnencodableValue(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGaugeValue("NonFinite", math.Inf(1))
	router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())
	response := postJSON(t, router, "/value", `{"id":"NonFinite","type":"gauge"}`)
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Empty(t, response.Body.String())
}
