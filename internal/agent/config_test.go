package agent

import (
	"testing"
	"time"

	config "github.com/fgugnin23/go-metrics/internal/config/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgsConfigs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		env       map[string]string
		collector CollectorConfig
		sender    SenderConfig
	}{
		{
			name:      "defaults",
			collector: DefaultCollectorConfig(),
			sender:    DefaultSenderConfig(),
		},
		{
			name:      "custom flags",
			args:      []string{"-a", "metrics.example:9090/", "-p", "3", "-r", "15"},
			collector: CollectorConfig{PollInterval: 3 * time.Second},
			sender: SenderConfig{
				ReportInterval: 15 * time.Second,
				ServerBaseURL:  "http://metrics.example:9090",
			},
		},
		{
			name:      "environment overrides flags in seconds",
			args:      []string{"-a", "metrics.example:9090", "-p", "3", "-r", "15"},
			env:       map[string]string{config.EnvAddress: "env.example:9091/", config.EnvReportInterval: "30", config.EnvPollInterval: "5"},
			collector: CollectorConfig{PollInterval: 5 * time.Second},
			sender: SenderConfig{
				ReportInterval: 30 * time.Second,
				ServerBaseURL:  "http://env.example:9091",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string {
				return test.env[key]
			}

			cfg, err := config.Parse(test.args, getenv)
			require.NoError(t, err)

			assert.Equal(t, test.collector, ArgsCollectorConfig(cfg))
			assert.Equal(t, test.sender, ArgsSenderConfig(cfg))
		})
	}
}
