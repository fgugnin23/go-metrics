package agent

import "time"

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
