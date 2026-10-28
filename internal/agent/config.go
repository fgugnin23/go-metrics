package agent

import (
	"time"

	config "github.com/fgugnin23/go-metrics/internal/config/agent"
)

type Config struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerBaseURL  string
}

func DefaultConfig() Config {
	return Config{
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		ServerBaseURL:  "http://localhost:8080",
	}
}

func ArgsConfig() Config {
	config.ParseFlags()
	return Config{
		PollInterval:   time.Duration(config.PollInterval) * time.Second,
		ReportInterval: time.Duration(config.ReportInterval) * time.Second,
		ServerBaseURL:  config.ServerAddr,
	}
}
