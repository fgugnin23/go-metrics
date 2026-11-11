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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.ParseFlags(test.args)
			require.NoError(t, err)

			assert.Equal(t, test.collector, ArgsCollectorConfig(cfg))
			assert.Equal(t, test.sender, ArgsSenderConfig(cfg))
		})
	}
}
