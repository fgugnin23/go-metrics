package config

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want Config
	}{
		{name: "defaults", want: Config{RunAddr: "localhost:8080"}},
		{name: "custom address", args: []string{"-a", "127.0.0.1:9090"}, want: Config{RunAddr: "127.0.0.1:9090"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := ParseFlags(test.args)
			require.NoError(t, err)
			assert.Equal(t, test.want, cfg)
		})
	}
}

func TestParseFlags_IndependentCalls(t *testing.T) {
	t.Parallel()

	first, err := ParseFlags([]string{"-a", "127.0.0.1:9090"})
	require.NoError(t, err)
	second, err := ParseFlags(nil)
	require.NoError(t, err)

	assert.Equal(t, Config{RunAddr: "127.0.0.1:9090"}, first)
	assert.Equal(t, Config{RunAddr: "localhost:8080"}, second)
}

func TestParseFlags_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"-unknown"}},
		{name: "missing address", args: []string{"-a"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := ParseFlags(test.args)
			require.Error(t, err)
			assert.Empty(t, cfg)
		})
	}
}

func TestParseFlags_Help(t *testing.T) {
	t.Parallel()

	_, err := ParseFlags([]string{"-h"})
	assert.ErrorIs(t, err, flag.ErrHelp)
}
