package handler

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
)

func gzipBody(t *testing.T, body string) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return compressed.Bytes()
}

func gunzipBody(t *testing.T, body []byte) string {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(decoded)
}

func TestNewRouter_GzipRequestsAndResponses(t *testing.T) {
	for _, compressRequest := range []bool{false, true} {
		for _, compressResponse := range []bool{false, true} {
			t.Run(fmt.Sprintf("request=%t/response=%t", compressRequest, compressResponse), func(t *testing.T) {
				storage := repository.NewMemStorage()
				var output bytes.Buffer
				router := NewRouter(service.NewMetricsService(storage), zerolog.New(&output))
				for _, step := range []struct {
					path string
					body string
					want string
				}{
					{"/update", `{"id":"Alloc","type":"gauge","value":1.5}`, `{"id":"Alloc","type":"gauge","value":1.5}`},
					{"/update/", `{"id":"PollCount","type":"counter","delta":2}`, `{"id":"PollCount","type":"counter","delta":2}`},
					{"/update", `{"id":"PollCount","type":"counter","delta":3}`, `{"id":"PollCount","type":"counter","delta":5}`},
					{"/value", `{"id":"Alloc","type":"gauge"}`, `{"id":"Alloc","type":"gauge","value":1.5}`},
					{"/value/", `{"id":"PollCount","type":"counter"}`, `{"id":"PollCount","type":"counter","delta":5}`},
				} {
					body := []byte(step.body)
					if compressRequest {
						body = gzipBody(t, step.body)
					}
					request := httptest.NewRequest(http.MethodPost, step.path, bytes.NewReader(body))
					request.Header.Set("Content-Type", "application/json; charset=utf-8")
					if compressRequest {
						request.Header.Set("Content-Encoding", "gzip")
					}
					if compressResponse {
						request.Header.Set("Accept-Encoding", "gzip")
					}
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)

					require.Equal(t, http.StatusOK, response.Code)
					assert.Equal(t, "application/json", response.Result().Header.Get("Content-Type"))
					assert.Contains(t, response.Result().Header.Values("Vary"), "Accept-Encoding")
					decoded := response.Body.String()
					if compressResponse {
						assert.Equal(t, "gzip", response.Result().Header.Get("Content-Encoding"))
						decoded = gunzipBody(t, response.Body.Bytes())
					} else {
						assert.Empty(t, response.Result().Header.Get("Content-Encoding"))
					}
					assert.JSONEq(t, step.want, decoded)

					entries := readLogEntries(t, &output)
					require.Len(t, entries, 1)
					assertRequestLog(t, entries[0], http.MethodPost, step.path, http.StatusOK, response.Body.Len())
				}
				assert.Equal(t, map[string]float64{"Alloc": 1.5}, storage.GetGaugeMetrics())
				assert.Equal(t, map[string]int64{"PollCount": 5}, storage.GetCounterMetrics())
			})
		}
	}
}

func TestNewRouter_GzipNegotiation(t *testing.T) {
	router := NewRouter(service.NewMetricsService(repository.NewMemStorage()), zerolog.Nop())
	plain := do(t, router, http.MethodGet, "/").Body.String()
	for _, test := range []struct {
		name    string
		headers []string
		gzip    bool
	}{
		{"no header", nil, false},
		{"gzip", []string{"gzip"}, true},
		{"list", []string{"br, gzip, deflate"}, true},
		{"case and whitespace", []string{" GZip "}, true},
		{"multiple headers", []string{"br", "gzip"}, true},
		{"other encodings", []string{"br, deflate"}, false},
		{"substring is not gzip", []string{"notgzip"}, false},
		{"positive quality", []string{"gzip;q=0.5"}, true},
		{"zero quality", []string{"gzip;q=0"}, false},
		{"zero decimal quality", []string{"gzip;q=0.000"}, false},
		{"wildcard", []string{"*"}, true},
		{"excluded wildcard", []string{"*;q=0"}, false},
		{"explicit exclusion", []string{"*;q=1, gzip;q=0"}, false},
		{"explicit inclusion", []string{"gzip, *;q=0"}, true},
		{"invalid quality", []string{"gzip;q=invalid"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header["Accept-Encoding"] = test.headers
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, "text/html; charset=utf-8", response.Result().Header.Get("Content-Type"))
			assert.Contains(t, response.Result().Header.Values("Vary"), "Accept-Encoding")
			if test.gzip {
				assert.Equal(t, "gzip", response.Result().Header.Get("Content-Encoding"))
				assert.Equal(t, plain, gunzipBody(t, response.Body.Bytes()))
			} else {
				assert.Empty(t, response.Result().Header.Get("Content-Encoding"))
				assert.Equal(t, plain, response.Body.String())
			}
		})
	}
}

func TestNewRouter_RejectsInvalidGzip(t *testing.T) {
	validJSON := `{"id":"Alloc","type":"gauge","value":99}`
	compressed := gzipBody(t, validJSON)
	badChecksum := bytes.Clone(compressed)
	badChecksum[len(badChecksum)-8] ^= 0xff
	for _, path := range []string{"/update", "/value"} {
		for _, test := range []struct {
			name     string
			body     []byte
			encoding string
			status   int
		}{
			{"empty gzip", nil, "gzip", http.StatusBadRequest},
			{"plain body labelled gzip", []byte(validJSON), "gzip", http.StatusBadRequest},
			{"truncated gzip", compressed[:len(compressed)-1], "gzip", http.StatusBadRequest},
			{"invalid checksum", badChecksum, "gzip", http.StatusBadRequest},
			{"invalid JSON", gzipBody(t, `{`), "gzip", http.StatusBadRequest},
			{"trailing JSON", gzipBody(t, validJSON+` {}`), "gzip", http.StatusBadRequest},
			{"gzip without header", compressed, "", http.StatusBadRequest},
			{"unsupported encoding", []byte(validJSON), "br", http.StatusUnsupportedMediaType},
		} {
			t.Run(path+"/"+test.name, func(t *testing.T) {
				storage := repository.NewMemStorage()
				storage.SetGaugeValue("Alloc", 10)
				router := NewRouter(service.NewMetricsService(storage), zerolog.Nop())
				request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(test.body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Content-Encoding", test.encoding)
				request.Header.Set("Accept-Encoding", "gzip")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				assert.Equal(t, test.status, response.Code)
				assert.Equal(t, map[string]float64{"Alloc": 10}, storage.GetGaugeMetrics())
				assert.Empty(t, storage.GetCounterMetrics())
			})
		}
	}
}

func TestGzipMiddleware_ResponseHeaders(t *testing.T) {
	for _, test := range []struct {
		name        string
		contentType string
		encoding    string
		status      int
		gzip        bool
	}{
		{"JSON with charset", "application/json; charset=utf-8", "", http.StatusCreated, true},
		{"HTML with charset", "text/html; charset=utf-8", "", http.StatusOK, true},
		{"plain text", "text/plain; charset=utf-8", "", http.StatusOK, false},
		{"binary", "application/octet-stream", "", http.StatusOK, false},
		{"already encoded", "application/json", "br", http.StatusOK, false},
		{"no content", "application/json", "", http.StatusNoContent, false},
		{"not modified", "text/html", "", http.StatusNotModified, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := strings.Repeat("content ", 100)
			if test.status == http.StatusNoContent || test.status == http.StatusNotModified {
				body = ""
			}
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				if test.encoding != "" {
					w.Header().Set("Content-Encoding", test.encoding)
				}
				w.WriteHeader(test.status)
				w.WriteHeader(http.StatusInternalServerError)
				if body != "" {
					_, _ = io.WriteString(w, body[:100])
					_, _ = io.WriteString(w, body[100:])
				}
			})
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Accept-Encoding", "gzip")
			response := httptest.NewRecorder()
			gzipMiddleware(next).ServeHTTP(response, request)
			assert.Equal(t, test.status, response.Code)
			assert.Equal(t, test.contentType, response.Result().Header.Get("Content-Type"))
			if test.gzip {
				assert.Empty(t, response.Result().Header.Get("Content-Length"))
				assert.Equal(t, "gzip", response.Result().Header.Get("Content-Encoding"))
				assert.Equal(t, body, gunzipBody(t, response.Body.Bytes()))
			} else {
				assert.Equal(t, test.encoding, response.Result().Header.Get("Content-Encoding"))
				assert.Equal(t, body, response.Body.String())
			}
		})
	}
}
