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
		{
			name: "defaults",
			want: Config{ServerAddr: "http://localhost:8080", ReportInterval: 10, PollInterval: 2},
		},
		{
			name: "custom flags",
			args: []string{"-a", "metrics.example:9090/", "-r", "15", "-p", "3"},
			want: Config{ServerAddr: "http://metrics.example:9090", ReportInterval: 15, PollInterval: 3},
		},
		{
			name: "explicit scheme",
			args: []string{"-a", "https://metrics.example/"},
			want: Config{ServerAddr: "https://metrics.example", ReportInterval: 10, PollInterval: 2},
		},
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

	first, err := ParseFlags([]string{"-a", "first.example:9090", "-r", "20", "-p", "4"})
	require.NoError(t, err)
	second, err := ParseFlags(nil)
	require.NoError(t, err)

	assert.Equal(t, Config{ServerAddr: "http://first.example:9090", ReportInterval: 20, PollInterval: 4}, first)
	assert.Equal(t, Config{ServerAddr: "http://localhost:8080", ReportInterval: 10, PollInterval: 2}, second)
}

func TestParseFlags_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"-unknown"}},
		{name: "missing value", args: []string{"-a"}},
		{name: "invalid poll interval", args: []string{"-p", "invalid"}},
		{name: "invalid report interval", args: []string{"-r", "invalid"}},
		{name: "empty address", args: []string{"-a", ""}},
		{name: "missing host", args: []string{"-a", "http://"}},
		{name: "malformed address", args: []string{"-a", "http://["}},
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
