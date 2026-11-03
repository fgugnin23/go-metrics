package agent

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentState(t *testing.T) {
	state := newAgentState()

	assert.Len(t, state.gauges, len(runtimeGaugeNames))
	assert.Equal(t, "PollCount", state.pollCount.name)
	assert.Zero(t, state.pollCount.value)
	assert.Equal(t, "RandomValue", state.randomValue.name)
	assert.Zero(t, state.randomValue.value)
}

func TestAgentState_Poll(t *testing.T) {
	state := newAgentState()

	state.poll()

	assert.Equal(t, int64(1), state.pollCount.value)
	assert.Equal(t, int64(1), state.totalPolls)
	assert.NotZero(t, state.randomValue.value)

	alloc, ok := findGauge(state.gauges, "Alloc")
	require.True(t, ok)
	assert.Positive(t, alloc, "poll должен заполнять runtime-метрики")

	state.poll()

	assert.Equal(t, int64(2), state.pollCount.value)
	assert.Equal(t, int64(2), state.totalPolls)
}

func TestShouldReport(t *testing.T) {
	tests := []struct {
		name           string
		pollInterval   time.Duration
		reportInterval time.Duration
		pollCount      int64
		want           bool
	}{
		{"2s/10s, 1 опрос", 2 * time.Second, 10 * time.Second, 1, false},
		{"2s/10s, 4 опроса", 2 * time.Second, 10 * time.Second, 4, false},
		{"2s/10s, 5 опросов", 2 * time.Second, 10 * time.Second, 5, true},
		{"2s/10s, 9 опросов", 2 * time.Second, 10 * time.Second, 9, false},
		{"2s/10s, 10 опросов", 2 * time.Second, 10 * time.Second, 10, true},
		{"равные интервалы", time.Second, time.Second, 1, true},
		{"report меньше poll", 10 * time.Second, time.Second, 1, true},
		{"нулевой poll interval", 0, 10 * time.Second, 1, true},
		{"неровное деление 3s/10s", 3 * time.Second, 10 * time.Second, 3, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, shouldReport(test.pollCount, test.pollInterval, test.reportInterval))
		})
	}
}

func TestShouldReport_DefaultConfig(t *testing.T) {
	collectorCfg := DefaultCollectorConfig()
	senderCfg := DefaultSenderConfig()

	reported := 0
	for pollCount := int64(1); pollCount <= 10; pollCount++ {
		if shouldReport(pollCount, collectorCfg.PollInterval, senderCfg.ReportInterval) {
			reported++
		}
	}

	assert.Equal(t, 2, reported, "при 2s/10s за 10 опросов должно быть 2 отправки")
}

func TestAgentState_Report_SendsAllMetrics(t *testing.T) {
	rec := &recorder{}
	server := httptest.NewServer(rec)
	defer server.Close()

	state := newAgentState()
	state.poll()

	cfg := SenderConfig{ServerBaseURL: server.URL}
	errs := state.report(*server.Client(), cfg)

	require.Empty(t, errs)

	methods, paths := rec.snapshot()
	assert.Len(t, paths, len(state.gauges)+2, "все gauge + PollCount + RandomValue")

	for _, method := range methods {
		assert.Equal(t, http.MethodPost, method)
	}

	joined := strings.Join(paths, "\n")
	assert.Contains(t, joined, "/update/counter/PollCount/1")
	assert.Contains(t, joined, "/update/gauge/RandomValue/")
	assert.Contains(t, joined, "/update/gauge/Alloc/")

	for _, gauge := range state.gauges {
		assert.Contains(t, joined, "/update/gauge/"+gauge.name+"/")
	}
}

func TestAgentState_Report_ResetsPollCountDelta(t *testing.T) {
	rec := &recorder{}
	server := httptest.NewServer(rec)
	defer server.Close()

	state := newAgentState()
	state.poll()
	state.poll()

	require.Empty(t, state.report(*server.Client(), SenderConfig{ServerBaseURL: server.URL}))

	assert.Zero(t, state.pollCount.value, "после успешной отправки приращение сбрасывается")
	assert.Equal(t, int64(2), state.totalPolls, "общее число опросов не сбрасывается")
	assert.Equal(t, []int64{2}, pollCountValues(rec))
}

func TestAgentState_Report_KeepsPollCountDeltaOnError(t *testing.T) {
	failing := httptest.NewServer(&recorder{status: http.StatusInternalServerError})
	defer failing.Close()

	state := newAgentState()
	state.poll()

	require.NotEmpty(t, state.report(*failing.Client(), SenderConfig{ServerBaseURL: failing.URL}))
	assert.Equal(t, int64(1), state.pollCount.value, "недоставленное приращение сохраняется")

	rec := &recorder{}
	server := httptest.NewServer(rec)
	defer server.Close()

	state.poll()
	require.Empty(t, state.report(*server.Client(), SenderConfig{ServerBaseURL: server.URL}))

	assert.Equal(t, []int64{2}, pollCountValues(rec), "приращение уходит целиком следующей отправкой")
}

func TestAgentState_PollCountSumMatchesTotalPolls(t *testing.T) {
	rec := &recorder{}
	server := httptest.NewServer(rec)
	defer server.Close()

	collectorCfg := DefaultCollectorConfig()
	senderCfg := SenderConfig{ServerBaseURL: server.URL, ReportInterval: 10 * time.Second}
	state := newAgentState()

	for poll := 0; poll < 20; poll++ {
		state.poll()

		if shouldReport(state.totalPolls, collectorCfg.PollInterval, senderCfg.ReportInterval) {
			require.Empty(t, state.report(*server.Client(), senderCfg))
		}
	}

	var sum int64
	for _, value := range pollCountValues(rec) {
		sum += value
	}

	assert.Equal(t, int64(20), state.totalPolls)
	assert.Equal(t, int64(20), sum, "сумма приращений на сервере равна числу опросов")
}

func TestAgentState_Report_CollectsErrors(t *testing.T) {
	server := httptest.NewServer(&recorder{status: http.StatusInternalServerError})
	defer server.Close()

	state := newAgentState()
	cfg := SenderConfig{ServerBaseURL: server.URL}

	errs := state.report(*server.Client(), cfg)

	assert.Len(t, errs, len(state.gauges)+2, "ошибка на каждую метрику")
}

func TestAgentState_Report_ContinuesAfterError(t *testing.T) {
	rec := &recorder{status: http.StatusBadRequest}
	server := httptest.NewServer(rec)
	defer server.Close()

	state := newAgentState()
	errs := state.report(*server.Client(), SenderConfig{ServerBaseURL: server.URL})

	require.NotEmpty(t, errs)

	_, paths := rec.snapshot()
	assert.Len(t, paths, len(state.gauges)+2, "отправка не должна прерываться на первой ошибке")
}

func TestDefaultConfigs(t *testing.T) {
	collectorCfg := DefaultCollectorConfig()
	senderCfg := DefaultSenderConfig()

	assert.Equal(t, 2*time.Second, collectorCfg.PollInterval)
	assert.Equal(t, 10*time.Second, senderCfg.ReportInterval)
	assert.Equal(t, "http://localhost:8080", senderCfg.ServerBaseURL)
}

func pollCountValues(rec *recorder) []int64 {
	_, paths := rec.snapshot()

	var values []int64

	for _, path := range paths {
		raw, isPollCount := strings.CutPrefix(path, "/update/counter/PollCount/")
		if !isPollCount {
			continue
		}

		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			continue
		}

		values = append(values, value)
	}

	return values
}

func findGauge(gauges []GaugeMetric, name string) (float64, bool) {
	for _, gauge := range gauges {
		if gauge.name == name {
			return gauge.value, true
		}
	}

	return 0, false
}

type recorder struct {
	status  int
	methods []string
	paths   []string
}

func (r *recorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.methods = append(r.methods, req.Method)
	r.paths = append(r.paths, req.URL.Path)

	if r.status != 0 {
		w.WriteHeader(r.status)
	}
}

func (r *recorder) snapshot() (methods, paths []string) {
	return append([]string(nil), r.methods...), append([]string(nil), r.paths...)
}
