package agent

import (
	"time"

	config "github.com/fgugnin23/go-metrics/internal/config/agent"
)

type SenderConfig struct {
	ReportInterval time.Duration
	ServerBaseURL  string
}

type CollectorConfig struct {
	PollInterval time.Duration
}

func DefaultSenderConfig() SenderConfig {
	return SenderConfig{
		ReportInterval: 10 * time.Second,
		ServerBaseURL:  "http://localhost:8080",
	}
}

func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		PollInterval: 2 * time.Second,
	}
}

func ArgsSenderConfig(cfg config.Config) SenderConfig {
	return SenderConfig{
		ReportInterval: time.Duration(cfg.ReportInterval) * time.Second,
		ServerBaseURL:  cfg.ServerAddr,
	}
}

func ArgsCollectorConfig(cfg config.Config) CollectorConfig {
	return CollectorConfig{
		PollInterval: time.Duration(cfg.PollInterval) * time.Second,
	}
}
