package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStorage struct {
	gaugeCalls []struct {
		name  string
		value float64
	}
	counterCalls []struct {
		name  string
		value int64
	}
}

func (s *mockStorage) SetGaugeValue(name string, value float64) {
	s.gaugeCalls = append(s.gaugeCalls, struct {
		name  string
		value float64
	}{name, value})
}

func (s *mockStorage) SetCounterValue(name string, value int64) {
	s.counterCalls = append(s.counterCalls, struct {
		name  string
		value int64
	}{name, value})
}

func do(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))

	return recorder
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
			router := NewRouter(repository.NewMemStorage())

			recorder := do(t, router, test.method, test.path)

			assert.Equal(t, test.wantStatus, recorder.Code)
		})
	}
}

func TestUpdateMetricsHandle_StoresValues(t *testing.T) {
	storage := repository.NewMemStorage()
	router := NewRouter(storage)

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
	router := NewRouter(storage)

	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/counter/PollCount/2").Code)
	require.Equal(t, http.StatusOK, do(t, router, http.MethodPost, "/update/counter/PollCount/2").Code)

	counter, ok := storage.GetCounterValue("PollCount")
	require.True(t, ok)
	assert.Equal(t, int64(4), counter)
}

func TestUpdateMetricsHandle_DoesNotWriteOnInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"нечисловой gauge", "/update/gauge/X/abc"},
		{"дробный counter", "/update/counter/X/1.5"},
		{"неизвестный тип", "/update/unknown/X/1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := &mockStorage{}
			router := NewRouter(storage)

			do(t, router, http.MethodPost, test.path)

			assert.Empty(t, storage.gaugeCalls)
			assert.Empty(t, storage.counterCalls)
		})
	}
}

func TestUpdateMetricsHandle_EmptyNameIsNotFound(t *testing.T) {
	storage := &mockStorage{}
	handle := NewUpdateMetricsHandle(storage)

	request := httptest.NewRequest(http.MethodPost, "/update/gauge//1", nil)
	request.SetPathValue("metric_type", "gauge")
	request.SetPathValue("metric_name", "")
	request.SetPathValue("metric_value", "1")

	recorder := httptest.NewRecorder()
	handle.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Empty(t, storage.gaugeCalls)
}

func TestNewRouter_UnknownPath(t *testing.T) {
	router := NewRouter(repository.NewMemStorage())

	assert.Equal(t, http.StatusNotFound, do(t, router, http.MethodGet, "/unknown").Code)
}
