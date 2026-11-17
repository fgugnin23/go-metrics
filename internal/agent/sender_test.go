package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgugnin23/go-metrics/internal/handler"
	models "github.com/fgugnin23/go-metrics/internal/model"
	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
)

func TestSendMetric_GzipJSON(t *testing.T) {
	zeroGauge, negativeGauge := 0.0, -0.125
	zeroCounter, largeCounter := int64(0), int64(math.MaxInt64)
	tests := []struct {
		name   string
		metric models.Metrics
		want   string
	}{
		{"zero gauge", models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &zeroGauge}, `{"id":"Alloc","type":"gauge","value":0}`},
		{"negative gauge", models.Metrics{ID: "GCCPUFraction", MType: models.Gauge, Value: &negativeGauge}, `{"id":"GCCPUFraction","type":"gauge","value":-0.125}`},
		{"zero counter", models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &zeroCounter}, `{"id":"PollCount","type":"counter","delta":0}`},
		{"large counter", models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &largeCounter}, `{"id":"PollCount","type":"counter","delta":9223372036854775807}`},
		{"name with special characters", models.Metrics{ID: "метрика /?\"", MType: models.Gauge, Value: &zeroGauge}, `{"id":"метрика /?\"","type":"gauge","value":0}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/update", r.URL.Path)
				assert.Empty(t, r.URL.RawQuery)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
				assert.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))
				reader, err := gzip.NewReader(r.Body)
				if !assert.NoError(t, err) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer reader.Close()
				body, err := io.ReadAll(reader)
				if !assert.NoError(t, err) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				assert.JSONEq(t, test.want, string(body))
				var metric models.Metrics
				assert.NoError(t, json.Unmarshal(body, &metric))
				assert.Equal(t, test.metric, metric)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Encoding", "gzip")
				writer := gzip.NewWriter(w)
				_, err = writer.Write(body)
				assert.NoError(t, err)
				assert.NoError(t, writer.Close())
			}))
			defer server.Close()

			require.NoError(t, sendMetric(*server.Client(), server.URL, test.metric))
			require.NoError(t, sendMetric(*server.Client(), server.URL+"/", test.metric))
		})
	}
}

func TestSendMetric_RejectsUnencodableValue(t *testing.T) {
	value := math.NaN()
	metric := models.Metrics{ID: "Invalid", MType: models.Gauge, Value: &value}
	err := sendMetric(http.Client{}, "http://localhost:8080", metric)
	require.ErrorContains(t, err, "encode metric Invalid")
}

func TestAgentState_Report_GzipJSONRoundTrip(t *testing.T) {
	router := handler.NewRouter(service.NewMetricsService(repository.NewMemStorage()), zerolog.Nop())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/update" {
			assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		}
		router.ServeHTTP(w, r)
		assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	}))
	defer server.Close()

	state := agentState{
		gauges:      []GaugeMetric{{name: "Alloc", value: 1.5}, {name: "Zero", value: 0}},
		pollCount:   CounterMetric{name: "PollCount", value: 2},
		randomValue: GaugeMetric{name: "RandomValue", value: -0.25},
	}
	cfg := SenderConfig{ServerBaseURL: server.URL}
	require.Empty(t, state.report(*server.Client(), cfg))
	assert.Zero(t, state.pollCount.value)

	state.gauges[0].value = 2.5
	state.pollCount.value = 3
	require.Empty(t, state.report(*server.Client(), cfg))
	assert.Zero(t, state.pollCount.value)

	for _, test := range []struct {
		request string
		want    string
	}{
		{`{"id":"Alloc","type":"gauge"}`, `{"id":"Alloc","type":"gauge","value":2.5}`},
		{`{"id":"Zero","type":"gauge"}`, `{"id":"Zero","type":"gauge","value":0}`},
		{`{"id":"RandomValue","type":"gauge"}`, `{"id":"RandomValue","type":"gauge","value":-0.25}`},
		{`{"id":"PollCount","type":"counter"}`, `{"id":"PollCount","type":"counter","delta":5}`},
	} {
		response, err := server.Client().Post(server.URL+"/value", "application/json", strings.NewReader(test.request))
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.True(t, response.Uncompressed, "the HTTP client must decompress the server's gzip response")
		assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
		assert.JSONEq(t, test.want, string(body))
	}
}
