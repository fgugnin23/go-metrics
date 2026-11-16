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
		{
			name: "environment only",
			env:  map[string]string{EnvAddress: "env.example:9091/", EnvReportInterval: "30", EnvPollInterval: "5"},
			want: Config{ServerAddr: "http://env.example:9091", ReportInterval: 30, PollInterval: 5},
		},
		{
			name: "environment overrides flags",
			args: []string{"-a", "flags.example:9090", "-r", "15", "-p", "3"},
			env:  map[string]string{EnvAddress: "env.example:9091/", EnvReportInterval: "30", EnvPollInterval: "5"},
			want: Config{ServerAddr: "http://env.example:9091", ReportInterval: 30, PollInterval: 5},
		},
		{
			name: "environment address with explicit scheme",
			env:  map[string]string{EnvAddress: "https://env.example/"},
			want: Config{ServerAddr: "https://env.example", ReportInterval: 10, PollInterval: 2},
		},
		{
			name: "mixed environment flags and defaults",
			args: []string{"-a", "flags.example:9090", "-r", "15"},
			env:  map[string]string{EnvReportInterval: "30"},
			want: Config{ServerAddr: "http://flags.example:9090", ReportInterval: 30, PollInterval: 2},
		},
		{
			name: "empty environment uses flags",
			args: []string{"-a", "flags.example:9090", "-r", "15", "-p", "3"},
			env:  map[string]string{EnvAddress: "", EnvReportInterval: "", EnvPollInterval: ""},
			want: Config{ServerAddr: "http://flags.example:9090", ReportInterval: 15, PollInterval: 3},
		},
		{
			name: "empty environment uses defaults",
			env:  map[string]string{EnvAddress: "", EnvReportInterval: "", EnvPollInterval: ""},
			want: Config{ServerAddr: "http://localhost:8080", ReportInterval: 10, PollInterval: 2},
		},
		{
			name: "decimal intervals",
			env:  map[string]string{EnvReportInterval: "010", EnvPollInterval: "008"},
			want: Config{ServerAddr: "http://localhost:8080", ReportInterval: 10, PollInterval: 8},
		},
		{
			name: "zero and negative intervals remain accepted",
			env:  map[string]string{EnvReportInterval: "0", EnvPollInterval: "-1"},
			want: Config{ServerAddr: "http://localhost:8080", ReportInterval: 0, PollInterval: -1},
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

	first, err := Parse([]string{"-a", "first.example:9090", "-r", "20", "-p", "4"}, getenv)
	require.NoError(t, err)
	env = map[string]string{EnvAddress: "second.example:9091", EnvReportInterval: "30", EnvPollInterval: "5"}
	second, err := Parse(nil, getenv)
	require.NoError(t, err)
	env = nil
	third, err := Parse(nil, getenv)
	require.NoError(t, err)

	assert.Equal(t, Config{ServerAddr: "http://first.example:9090", ReportInterval: 20, PollInterval: 4}, first)
	assert.Equal(t, Config{ServerAddr: "http://second.example:9091", ReportInterval: 30, PollInterval: 5}, second)
	assert.Equal(t, Config{ServerAddr: "http://localhost:8080", ReportInterval: 10, PollInterval: 2}, third)
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	env := map[string]string{EnvAddress: "env.example:9091", EnvReportInterval: "30", EnvPollInterval: "5"}
	getenv := func(key string) string {
		return env[key]
	}

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

			cfg, err := Parse(test.args, getenv)
			require.Error(t, err)
			assert.Empty(t, cfg)
		})
	}
}

func TestParse_EnvironmentErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
		raw  string
	}{
		{name: "missing host", env: EnvAddress, raw: "http://"},
		{name: "malformed address", env: EnvAddress, raw: "http://["},
		{name: "whitespace address", env: EnvAddress, raw: " "},
	}
	for _, env := range []string{EnvReportInterval, EnvPollInterval} {
		for _, raw := range []string{"invalid", "1.5", "10s", " ", "9223372036854775808", "-9223372036854775809"} {
			tests = append(tests, struct {
				name string
				env  string
				raw  string
			}{name: env + "/" + raw, env: env, raw: raw})
		}
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			env := map[string]string{test.env: test.raw}
			getenv := func(key string) string {
				return env[key]
			}

			cfg, err := Parse([]string{"-a", "flags.example:9090", "-r", "15", "-p", "3"}, getenv)
			require.ErrorContains(t, err, test.env)
			assert.Empty(t, cfg)
		})
	}
}

func TestParse_Help(t *testing.T) {
	t.Parallel()

	env := map[string]string{EnvAddress: "http://", EnvReportInterval: "invalid", EnvPollInterval: "invalid"}
	getenv := func(key string) string {
		return env[key]
	}

	_, err := Parse([]string{"-h"}, getenv)
	assert.ErrorIs(t, err, flag.ErrHelp)
}
