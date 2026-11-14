package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/fgugnin23/go-metrics/internal/model"
	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
)

type mockService struct {
	updateCalls []models.Metrics
	updateErr   error
	getCalls    []struct {
		metricType string
		name       string
	}
	getResult models.Metrics
	getErr    error
	metrics   []models.Metrics
	listCalls int
}

func (s *mockService) UpdateMetric(metric models.Metrics) error {
	s.updateCalls = append(s.updateCalls, metric)
	return s.updateErr
}

func (s *mockService) GetMetric(metricType, name string) (models.Metrics, error) {
	s.getCalls = append(s.getCalls, struct {
		metricType string
		name       string
	}{metricType, name})
	return s.getResult, s.getErr
}

func (s *mockService) ListMetrics() []models.Metrics {
	s.listCalls++
	return s.metrics
}

func do(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))

	return recorder
}

func withURLParams(r *http.Request, params map[string]string) *http.Request {
	routeContext := chi.NewRouteContext()
	for name, value := range params {
		routeContext.URLParams.Add(name, value)
	}

	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext))
}

func TestUpdateMetricsHandle_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"валидный gauge", http.MethodPost, "/update/gauge/Alloc/1.5", http.StatusOK},
		{"валидный counter", http.MethodPost, "/update/counter/PollCount/2", http.StatusOK},
		{"gauge в экспоненциальной записи", http.MethodPost, "/update/gauge/Alloc/1e3", http.StatusOK},
		{"отрицательный gauge", http.MethodPost, "/update/gauge/Alloc/-0.5", http.StatusOK},
		{"неизвестный тип метрики", http.MethodPost, "/update/unknown/X/1", http.StatusBadRequest},
		{"нечисловое значение gauge", http.MethodPost, "/update/gauge/X/abc", http.StatusBadRequest},
		{"дробное значение counter", http.MethodPost, "/update/counter/X/1.5", http.StatusBadRequest},
		{"пустое значение", http.MethodPost, "/update/counter/X/", http.StatusNotFound},
		{"нет сегмента значения", http.MethodPost, "/update/gauge/X", http.StatusNotFound},
		{"лишний сегмент", http.MethodPost, "/update/gauge/X/1/2", http.StatusNotFound},
		{"неверный метод", http.MethodGet, "/update/gauge/X/1", http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter(service.NewMetricsService(repository.NewMemStorage()), zerolog.Nop())

			recorder := do(t, router, test.method, test.path)

			assert.Equal(t, test.wantStatus, recorder.Code)
		})
	}
}

func TestUpdateMetricsHandle_StoresValues(t *testing.T) {
	storage := repository.NewMemStorage()
	router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())

	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/gauge/Alloc/1.5").Code)

	gauge, ok := storage.GetGaugeValue("Alloc")
	require.True(t, ok)
	assert.Equal(t, 1.5, gauge)

	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/gauge/Alloc/2.5").Code)

	gauge, ok = storage.GetGaugeValue("Alloc")
	require.True(t, ok)
	assert.Equal(t, 2.5, gauge, "gauge перезаписывается")
}

func TestUpdateMetricsHandle_CounterAccumulates(t *testing.T) {
	storage := repository.NewMemStorage()
	router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())

	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/counter/PollCount/2").Code)
	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/counter/PollCount/2").Code)

	counter, ok := storage.GetCounterValue("PollCount")
	require.True(t, ok)
	assert.Equal(t, int64(4), counter)
}

func TestUpdateMetricsHandle_DoesNotCallServiceOnInvalidValue(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"нечисловой gauge", "/update/gauge/X/abc"},
		{"дробный counter", "/update/counter/X/1.5"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metrics := &mockService{}
			router := NewRouter(metrics, zerolog.Nop())

			recorder := do(t, router, http.MethodPost, test.path)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Empty(t, metrics.updateCalls)
		})
	}
}

func TestUpdateMetricsHandle_EmptyNameIsNotFound(t *testing.T) {
	metrics := &mockService{updateErr: models.ErrInvalidMetricName}
	handle := NewMetricsHandler(metrics, zerolog.Nop())

	request := withURLParams(
		httptest.NewRequest(http.MethodPost, "/update/gauge//1", nil),
		map[string]string{"metric_type": "gauge", "metric_name": "", "metric_value": "1"},
	)

	recorder := httptest.NewRecorder()
	handle.Update(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, []models.Metrics{testGauge("", 1)}, metrics.updateCalls)
}

func TestUpdateMetricsHandle_UnknownTypeIsBadRequest(t *testing.T) {
	metrics := &mockService{updateErr: models.ErrInvalidMetricType}
	recorder := do(t, NewRouter(metrics, zerolog.Nop()), http.MethodPost, "/update/unknown/X/1")

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, []models.Metrics{{ID: "X", MType: "unknown"}}, metrics.updateCalls)
}

func TestGetMetricsHandle(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGaugeValue("Fractional", 1.5)
	storage.SetGaugeValue("Integral", 100)
	storage.SetGaugeValue("Negative", -0.25)
	storage.SetCounterValue("PollCount", 3)

	router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"дробный gauge", "/value/gauge/Fractional", http.StatusOK, "1.5"},
		{"целый gauge без дробной части", "/value/gauge/Integral", http.StatusOK, "100"},
		{"отрицательный gauge", "/value/gauge/Negative", http.StatusOK, "-0.25"},
		{"counter", "/value/counter/PollCount", http.StatusOK, "3"},
		{"неизвестное имя gauge", "/value/gauge/Unknown", http.StatusNotFound, ""},
		{"неизвестное имя counter", "/value/counter/Unknown", http.StatusNotFound, ""},
		{"неизвестный тип", "/value/unknown/Fractional", http.StatusBadRequest, ""},
		{"нет имени метрики", "/value/gauge", http.StatusNotFound, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := do(t, router, http.MethodGet, test.path)

			require.Equal(t, test.wantStatus, recorder.Code)

			if test.wantStatus != http.StatusOK {
				return
			}

			assert.Equal(t, test.wantBody, recorder.Body.String())
			assert.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
		})
	}
}

func TestGetMetricsHandle_WrongMethod(t *testing.T) {
	router := NewRouter(service.NewMetricsService(repository.NewMemStorage()), zerolog.Nop())

	recorder := do(t, router, http.MethodPost, "/value/gauge/Alloc")

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestGetMetricsHandle_EmptyNameIsNotFound(t *testing.T) {
	handle := NewMetricsHandler(&mockService{getErr: models.ErrInvalidMetricName}, zerolog.Nop())

	request := withURLParams(
		httptest.NewRequest(http.MethodGet, "/value/gauge/", nil),
		map[string]string{"metric_type": "gauge", "metric_name": ""},
	)

	recorder := httptest.NewRecorder()
	handle.Get(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestMainPage(t *testing.T) {
	storage := repository.NewMemStorage()
	router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())

	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/gauge/Alloc/1.5").Code)
	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/counter/PollCount/3").Code)

	recorder := do(t, router, http.MethodGet, "/")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))

	body := recorder.Body.String()
	assert.Contains(t, body, "<li><b>Alloc: </b>1.5</li>")
	assert.Contains(t, body, "<li><b>PollCount: </b>3</li>")
}

func TestMainPage_EmptyStorage(t *testing.T) {
	recorder := do(t, NewRouter(&mockService{}, zerolog.Nop()), http.MethodGet, "/")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "<h1>Список всех метрик</h1>")
}

func TestMainPage_WrongMethod(t *testing.T) {
	recorder := do(t, NewRouter(&mockService{}, zerolog.Nop()), http.MethodPost, "/")

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

func TestNewRouter_UnknownPath(t *testing.T) {
	router := NewRouter(&mockService{}, zerolog.Nop())

	assert.Equal(t, http.StatusNotFound, do(t, router, http.MethodGet, "/unknown").Code)
}

func TestUpdateMetricsHandle_ForwardsMetric(t *testing.T) {
	tests := []struct {
		path string
		want models.Metrics
	}{
		{path: "/update/gauge/Alloc/1.5", want: testGauge("Alloc", 1.5)},
		{path: "/update/counter/PollCount/2", want: testCounter("PollCount", 2)},
		{path: "/update/gauge/Alloc/0", want: testGauge("Alloc", 0)},
		{path: "/update/counter/PollCount/0", want: testCounter("PollCount", 0)},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			metrics := &mockService{}
			recorder := do(t, NewRouter(metrics, zerolog.Nop()), http.MethodPost, test.path)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, []models.Metrics{test.want}, metrics.updateCalls)
		})
	}
}

func TestMetricsHandler_ServiceErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "invalid type", err: models.ErrInvalidMetricType, status: http.StatusBadRequest},
		{name: "invalid value", err: models.ErrInvalidMetricValue, status: http.StatusBadRequest},
		{name: "empty name", err: models.ErrInvalidMetricName, status: http.StatusNotFound},
		{name: "not found", err: models.ErrMetricNotFound, status: http.StatusNotFound},
		{name: "wrapped error", err: fmt.Errorf("lookup: %w", models.ErrMetricNotFound), status: http.StatusNotFound},
		{name: "unexpected error", err: errors.New("internal failure"), status: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metrics := &mockService{updateErr: test.err, getErr: test.err}
			router := NewRouter(metrics, zerolog.Nop())
			update := do(t, router, http.MethodPost, "/update/gauge/Alloc/1.5")
			get := do(t, router, http.MethodGet, "/value/gauge/Alloc")

			assert.Equal(t, test.status, update.Code)
			assert.Equal(t, test.status, get.Code)
			assert.Empty(t, update.Body.String())
			assert.Empty(t, get.Body.String())
			assert.Len(t, metrics.updateCalls, 1)
			assert.Len(t, metrics.getCalls, 1)
		})
	}
}
