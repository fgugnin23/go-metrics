package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
)

func readLogEntries(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()

	var entries []map[string]any
	decoder := json.NewDecoder(output)
	for {
		var entry map[string]any
		err := decoder.Decode(&entry)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		assert.Equal(t, "info", entry["level"])
		entries = append(entries, entry)
	}
	return entries
}

func assertRequestLog(t *testing.T, entry map[string]any, method, uri string, status, size int) {
	t.Helper()

	assert.Equal(t, "HTTP request completed", entry["message"])
	assert.Equal(t, method, entry["method"])
	assert.Equal(t, uri, entry["uri"])
	assert.Equal(t, float64(status), entry["status"])
	assert.Equal(t, float64(size), entry["size"])
	duration, ok := entry["duration_ms"].(float64)
	require.True(t, ok, "duration_ms must be a number")
	assert.GreaterOrEqual(t, duration, float64(0))
}

func TestRequestLogger_Response(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantBody   string
	}{
		{
			name:       "empty implicit OK",
			handler:    func(http.ResponseWriter, *http.Request) {},
			wantStatus: http.StatusOK,
		},
		{
			name: "implicit OK with body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("hello"))
			},
			wantStatus: http.StatusOK,
			wantBody:   "hello",
		},
		{
			name: "explicit status and multiple Unicode writes",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("Привет, "))
				_, _ = w.Write([]byte("мир!"))
			},
			wantStatus: http.StatusCreated,
			wantBody:   "Привет, мир!",
		},
		{
			name: "first final status is preserved",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			wrapped := requestLogger(zerolog.New(&output))(test.handler)
			uri := "/metrics?name=Alloc&label=a%20b"
			response := do(t, wrapped, http.MethodPost, uri)

			assert.Equal(t, test.wantStatus, response.Code)
			assert.Equal(t, test.wantBody, response.Body.String())
			entries := readLogEntries(t, &output)
			require.Len(t, entries, 1)
			assertRequestLog(t, entries[0], http.MethodPost, uri, test.wantStatus, len(test.wantBody))
		})
	}
}

func TestRequestLogger_DurationIncludesHandler(t *testing.T) {
	var output bytes.Buffer
	var elapsed time.Duration
	wrapped := requestLogger(zerolog.New(&output))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, output.String(), "the request must be logged after the handler returns")
		started := time.Now()
		time.Sleep(10 * time.Millisecond)
		elapsed = time.Since(started)
	}))

	do(t, wrapped, http.MethodGet, "/")
	entries := readLogEntries(t, &output)
	require.Len(t, entries, 1)
	duration, ok := entries[0]["duration_ms"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, duration, float64(elapsed)/float64(time.Millisecond))
}

type shortResponseWriter struct {
	*httptest.ResponseRecorder
}

func (w shortResponseWriter) Write(p []byte) (int, error) {
	n, _ := w.ResponseRecorder.Write(p[:2])
	return n, io.ErrShortWrite
}

func TestRequestLogger_CountsActuallyWrittenBytes(t *testing.T) {
	var output bytes.Buffer
	wrapped := requestLogger(zerolog.New(&output))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n, err := w.Write([]byte("hello"))
		assert.Equal(t, 2, n)
		assert.ErrorIs(t, err, io.ErrShortWrite)
	}))
	response := shortResponseWriter{httptest.NewRecorder()}
	wrapped.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, "he", response.Body.String())
	entries := readLogEntries(t, &output)
	require.Len(t, entries, 1)
	assertRequestLog(t, entries[0], http.MethodGet, "/", http.StatusOK, 2)
}

func TestNewRouter_LogsAllRoutes(t *testing.T) {
	storage := repository.NewMemStorage()
	storage.SetGaugeValue("Alloc", 1.5)
	metrics := service.NewMetricsService(storage)
	tests := []struct {
		method string
		uri    string
		status int
	}{
		{http.MethodGet, "/", http.StatusOK},
		{http.MethodPost, "/update/gauge/Alloc/2.5", http.StatusOK},
		{http.MethodGet, "/value/gauge/Alloc?source=test", http.StatusOK},
		{http.MethodPost, "/update/gauge/Alloc/invalid", http.StatusBadRequest},
		{http.MethodGet, "/value/gauge/Unknown", http.StatusNotFound},
		{http.MethodGet, "/unknown", http.StatusNotFound},
		{http.MethodGet, "/update/gauge/Alloc/2.5", http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.uri, func(t *testing.T) {
			var output bytes.Buffer
			router := NewRouter(metrics, zerolog.New(&output))
			response := do(t, router, test.method, test.uri)

			assert.Equal(t, test.status, response.Code)
			entries := readLogEntries(t, &output)
			require.Len(t, entries, 1)
			assertRequestLog(t, entries[0], test.method, test.uri, test.status, response.Body.Len())
		})
	}
}

func TestNewRouter_LogsServiceErrorsAtInfo(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			var output bytes.Buffer
			err := errors.New("storage unavailable")
			router := NewRouter(&mockService{getErr: err, updateErr: err}, zerolog.New(&output))
			uri := "/value/gauge/Alloc"
			if method == http.MethodPost {
				uri = "/update/gauge/Alloc/1.5"
			}
			response := do(t, router, method, uri)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
			assert.Empty(t, response.Body.String())
			entries := readLogEntries(t, &output)
			require.Len(t, entries, 2)
			assert.Equal(t, "storage unavailable", entries[0]["error"])
			assertRequestLog(t, entries[1], method, uri, http.StatusInternalServerError, 0)
		})
	}
}

func TestNewRouter_RecoversAndLogsPanicsAtInfo(t *testing.T) {
	var output bytes.Buffer
	router := NewRouter(&mockService{}, zerolog.New(&output))
	router.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("handler failed")
	})

	response := do(t, router, http.MethodGet, "/panic")
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Empty(t, response.Body.String())
	entries := readLogEntries(t, &output)
	require.Len(t, entries, 2)
	assert.Equal(t, "handler failed", entries[0]["panic"])
	assert.NotEmpty(t, entries[0]["stack"])
	assertRequestLog(t, entries[1], http.MethodGet, "/panic", http.StatusInternalServerError, 0)
}

func TestNewRouter_AbortHandlerIsNotRecovered(t *testing.T) {
	var output bytes.Buffer
	router := NewRouter(&mockService{}, zerolog.New(&output))
	router.Get("/abort", func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	})

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		do(t, router, http.MethodGet, "/abort")
	})
	assert.Empty(t, output.String())
}
