package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildMetricURL(t *testing.T) {
	assert.Equal(t,
		"http://localhost:8080/update/gauge/Alloc/1.5",
		buildMetricURL("http://localhost:8080", "gauge", "Alloc", "1.5"),
	)
}

func TestGaugeURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		metric  GaugeMetric
		want    string
	}{
		{
			name:    "дробное значение",
			baseURL: "http://localhost:8080",
			metric:  GaugeMetric{name: "Alloc", value: 1.5},
			want:    "http://localhost:8080/update/gauge/Alloc/1.5",
		},
		{
			name:    "ноль",
			baseURL: "http://localhost:8080",
			metric:  GaugeMetric{name: "Alloc", value: 0},
			want:    "http://localhost:8080/update/gauge/Alloc/0",
		},
		{
			name:    "отрицательное значение",
			baseURL: "http://localhost:8080",
			metric:  GaugeMetric{name: "GCCPUFraction", value: -0.125},
			want:    "http://localhost:8080/update/gauge/GCCPUFraction/-0.125",
		},
		{
			name:    "большое значение без экспоненты",
			baseURL: "http://localhost:8080",
			metric:  GaugeMetric{name: "TotalAlloc", value: 1e21},
			want:    "http://localhost:8080/update/gauge/TotalAlloc/1000000000000000000000",
		},
		{
			name:    "маленькое значение без экспоненты",
			baseURL: "http://localhost:8080",
			metric:  GaugeMetric{name: "GCCPUFraction", value: 0.0000001},
			want:    "http://localhost:8080/update/gauge/GCCPUFraction/0.0000001",
		},
		{
			name:    "другой базовый адрес",
			baseURL: "http://127.0.0.1:1234",
			metric:  GaugeMetric{name: "Sys", value: 2},
			want:    "http://127.0.0.1:1234/update/gauge/Sys/2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, gaugeURL(test.baseURL, test.metric))
		})
	}
}

func TestCounterURL(t *testing.T) {
	tests := []struct {
		name   string
		metric CounterMetric
		want   string
	}{
		{
			name:   "обычное значение",
			metric: CounterMetric{name: "PollCount", value: 5},
			want:   "http://localhost:8080/update/counter/PollCount/5",
		},
		{
			name:   "ноль",
			metric: CounterMetric{name: "PollCount", value: 0},
			want:   "http://localhost:8080/update/counter/PollCount/0",
		},
		{
			name:   "отрицательное значение",
			metric: CounterMetric{name: "PollCount", value: -3},
			want:   "http://localhost:8080/update/counter/PollCount/-3",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, counterURL("http://localhost:8080", test.metric))
		})
	}
}
