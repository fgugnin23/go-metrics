package main

import (
	"bytes"
	"encoding/json"
	"log"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPServerLoggerUsesInfo(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(infoLogWriter{logger: zerolog.New(&output)}, "", 0)
	logger.Print("HTTP server diagnostic")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
	assert.Equal(t, "info", entry["level"])
	assert.Equal(t, "HTTP server diagnostic", entry["message"])
}
