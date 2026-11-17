package config

import (
	"flag"
	"testing"
	"time"

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
		{name: "defaults", want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "localhost:8080"}},
		{name: "custom address", args: []string{"-a", "127.0.0.1:9090"}, want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "127.0.0.1:9090"}},
		{
			name: "environment only",
			env:  map[string]string{EnvAddress: "127.0.0.1:9091"},
			want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "127.0.0.1:9091"},
		},
		{
			name: "environment overrides flag",
			args: []string{"-a", "127.0.0.1:9090"},
			env:  map[string]string{EnvAddress: "127.0.0.1:9091"},
			want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "127.0.0.1:9091"},
		},
		{
			name: "empty environment uses flag",
			args: []string{"-a", "127.0.0.1:9090"},
			env:  map[string]string{EnvAddress: ""},
			want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "127.0.0.1:9090"},
		},
		{
			name: "empty environment uses default",
			env:  map[string]string{EnvAddress: ""},
			want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "localhost:8080"},
		},
		{
			name: "listen on all interfaces",
			env:  map[string]string{EnvAddress: ":9090"},
			want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: ":9090"},
		},
		{
			name: "IPv6 address",
			env:  map[string]string{EnvAddress: "[::1]:9090"},
			want: Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "[::1]:9090"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) (string, bool) {
				value, ok := test.env[key]
				return value, ok
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
	getenv := func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	}

	first, err := Parse([]string{"-a", "127.0.0.1:9090"}, getenv)
	require.NoError(t, err)
	env = map[string]string{EnvAddress: "127.0.0.1:9091"}
	second, err := Parse(nil, getenv)
	require.NoError(t, err)
	env = nil
	third, err := Parse(nil, getenv)
	require.NoError(t, err)

	assert.Equal(t, Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "127.0.0.1:9090"}, first)
	assert.Equal(t, Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "127.0.0.1:9091"}, second)
	assert.Equal(t, Config{StoreInterval: 300 * time.Second, FileStoragePath: "metrics.json", Restore: true, RunAddr: "localhost:8080"}, third)
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	env := map[string]string{EnvAddress: "127.0.0.1:9091"}
	getenv := func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
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

	getenv := func(string) (string, bool) {
		t.Fatal("environment must not be read when showing help")
		return "", false
	}

	_, err := Parse([]string{"-h"}, getenv)
	assert.ErrorIs(t, err, flag.ErrHelp)
}

func TestPersistenceConfig(t *testing.T) {
	type testCase struct {
		name    string
		args    []string
		env     map[string]string
		path    string
		restore bool
		invalid bool
	}
	tests := []testCase{
		{name: "flags", args: []string{"-f", "custom.json", "-r=false"}, path: "custom.json"},
		{name: "environment overrides flags", args: []string{"-f", "flag.json", "-r=false"}, env: map[string]string{EnvFileStoragePath: "env.json", EnvRestore: "true"}, path: "env.json", restore: true},
		{name: "environment false", env: map[string]string{EnvRestore: "false"}, path: "metrics.json"},
	}
	for _, key := range []string{EnvFileStoragePath, EnvRestore} {
		tests = append(tests, testCase{name: "empty " + key, env: map[string]string{key: ""}, invalid: true})
	}
	tests = append(tests,
		testCase{name: "invalid restore env", env: map[string]string{EnvRestore: "maybe"}, invalid: true},
		testCase{name: "invalid restore flag", args: []string{"-r=maybe"}, invalid: true},
		testCase{name: "empty path flag", args: []string{"-f", ""}, invalid: true},
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.args, func(key string) (string, bool) { v, ok := tt.env[key]; return v, ok })
			if tt.invalid {
				require.Error(t, err)
				assert.Empty(t, cfg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.path, cfg.FileStoragePath)
			assert.Equal(t, tt.restore, cfg.Restore)
		})
	}
}

func TestStoreInterval(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    time.Duration
		invalid bool
	}{
		{name: "default", want: 300 * time.Second},
		{name: "flag", args: []string{"-i", "12"}, want: 12 * time.Second},
		{name: "zero", args: []string{"-i", "0"}},
		{name: "environment overrides flag", args: []string{"-i", "12"}, env: map[string]string{EnvStoreInterval: "42"}, want: 42 * time.Second},
		{name: "environment zero", args: []string{"-i", "12"}, env: map[string]string{EnvStoreInterval: "0"}},
		{name: "maximum", args: []string{"-i", "9223372036"}, want: 9223372036 * time.Second},
		{name: "missing flag value", args: []string{"-i"}, invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.args, func(key string) (string, bool) { value, ok := tt.env[key]; return value, ok })
			if tt.invalid {
				require.Error(t, err)
				assert.Empty(t, cfg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.StoreInterval)
		})
	}
	for _, raw := range []string{"", "-1", "1.5", "invalid", "9223372037", "9223372036854775808"} {
		t.Run("invalid flag "+raw, func(t *testing.T) {
			cfg, err := Parse([]string{"-i", raw}, func(string) (string, bool) { return "", false })
			require.Error(t, err)
			assert.Empty(t, cfg)
		})
		t.Run("invalid environment "+raw, func(t *testing.T) {
			cfg, err := Parse([]string{"-i", "12"}, func(key string) (string, bool) { return raw, key == EnvStoreInterval })
			require.Error(t, err)
			assert.Empty(t, cfg)
		})
	}
}
