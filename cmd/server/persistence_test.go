package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	config "github.com/fgugnin23/go-metrics/internal/config/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartupRestore(t *testing.T) {
	const snapshot = `[{"id":"g","type":"gauge","value":0},{"id":"c","type":"counter","delta":9223372036854775807}]`
	for _, tt := range []struct {
		name, data       string
		restore, invalid bool
	}{
		{name: "restores absolute values", data: snapshot, restore: true},
		{name: "missing file", restore: true},
		{name: "corrupt file", data: "broken", restore: true, invalid: true},
		{name: "duplicate metrics", data: `[{"id":"c","type":"counter","delta":1},{"id":"c","type":"counter","delta":2}]`, restore: true, invalid: true},
		{name: "restore disabled skips corrupt file", data: "broken"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metrics.json")
			if tt.data != "" {
				require.NoError(t, os.WriteFile(path, []byte(tt.data), 0600))
			}
			server, err := newServer(config.Config{RunAddr: "localhost:8080", FileStoragePath: path, Restore: tt.restore}, zerolog.Nop())
			if tt.invalid {
				require.Error(t, err)
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				for _, metric := range []struct{ path, value string }{
					{"/value/gauge/g", "0"},
					{"/value/counter/c", "9223372036854775807"},
				} {
					response := httptest.NewRecorder()
					server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, metric.path, nil))
					if tt.data == snapshot {
						assert.Equal(t, http.StatusOK, response.Code)
						assert.Equal(t, metric.value, response.Body.String())
					} else {
						assert.Equal(t, http.StatusNotFound, response.Code)
					}
				}
			}
			// Startup only reads the file; saving happens after an update.
			data, err := os.ReadFile(path)
			if tt.data == "" {
				require.ErrorIs(t, err, os.ErrNotExist)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.data, string(data))
			}
		})
	}
	t.Run("unreadable file prevents startup", func(t *testing.T) {
		server, err := newServer(config.Config{FileStoragePath: t.TempDir(), Restore: true}, zerolog.Nop())
		require.Error(t, err)
		assert.Nil(t, server)
	})
}

func TestServerSavesAfterUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	server, err := newServer(config.Config{FileStoragePath: path, Restore: true}, zerolog.Nop())
	require.NoError(t, err)
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/update/counter/count/42", nil))
	require.Equal(t, http.StatusOK, response.Code)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"id":"count","type":"counter","delta":42}]`, string(data))
}
