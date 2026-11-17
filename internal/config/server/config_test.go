package config

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		env  map[string]string
		want Config
	}{
		{name: "defaults", want: Config{RunAddr: "localhost:8080"}},
		{name: "custom address", args: []string{"-a", "127.0.0.1:9090"}, want: Config{RunAddr: "127.0.0.1:9090"}},
		{
			name: "environment only",
			env:  map[string]string{EnvAddress: "127.0.0.1:9091"},
			want: Config{RunAddr: "127.0.0.1:9091"},
		},
		{
			name: "environment overrides flag",
			args: []string{"-a", "127.0.0.1:9090"},
			env:  map[string]string{EnvAddress: "127.0.0.1:9091"},
			want: Config{RunAddr: "127.0.0.1:9091"},
		},
		{
			name: "empty environment uses flag",
			args: []string{"-a", "127.0.0.1:9090"},
			env:  map[string]string{EnvAddress: ""},
			want: Config{RunAddr: "127.0.0.1:9090"},
		},
		{
			name: "empty environment uses default",
			env:  map[string]string{EnvAddress: ""},
			want: Config{RunAddr: "localhost:8080"},
		},
		{
			name: "listen on all interfaces",
			env:  map[string]string{EnvAddress: ":9090"},
			want: Config{RunAddr: ":9090"},
		},
		{
			name: "IPv6 address",
			env:  map[string]string{EnvAddress: "[::1]:9090"},
			want: Config{RunAddr: "[::1]:9090"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string {
				return test.env[key]
			}

			cfg, err := Parse(test.args, getenv)
			require.NoError(t, err)
			assert.Equal(t, test.want, cfg)
		})
	}
}

func TestParse_IndependentCalls(t *testing.T) {
	t.Parallel()

	var env map[string]string
	getenv := func(key string) string {
		return env[key]
	}

	first, err := Parse([]string{"-a", "127.0.0.1:9090"}, getenv)
	require.NoError(t, err)
	env = map[string]string{EnvAddress: "127.0.0.1:9091"}
	second, err := Parse(nil, getenv)
	require.NoError(t, err)
	env = nil
	third, err := Parse(nil, getenv)
	require.NoError(t, err)

	assert.Equal(t, Config{RunAddr: "127.0.0.1:9090"}, first)
	assert.Equal(t, Config{RunAddr: "127.0.0.1:9091"}, second)
	assert.Equal(t, Config{RunAddr: "localhost:8080"}, third)
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	env := map[string]string{EnvAddress: "127.0.0.1:9091"}
	getenv := func(key string) string {
		return env[key]
	}

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

			cfg, err := Parse(test.args, getenv)
			require.Error(t, err)
			assert.Empty(t, cfg)
		})
	}
}

func TestParse_Help(t *testing.T) {
	t.Parallel()

	getenv := func(string) string {
		t.Fatal("environment must not be read when showing help")
		return ""
	}

	_, err := Parse([]string{"-h"}, getenv)
	assert.ErrorIs(t, err, flag.ErrHelp)
}
