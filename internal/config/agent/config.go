package config

import (
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	EnvAddress        = "ADDRESS"
	EnvReportInterval = "REPORT_INTERVAL"
	EnvPollInterval   = "POLL_INTERVAL"
)

type Config struct {
	ServerAddr     string
	ReportInterval int64
	PollInterval   int64
}

type serverURL struct {
	value *string
}

func (u serverURL) String() string {
	if u.value == nil {
		return ""
	}

	return *u.value
}

func (u serverURL) Set(raw string) error {
	if raw == "" {
		return fmt.Errorf("адрес сервера не может быть пустым")
	}

	if !strings.Contains(raw, "://") {
		raw = "http://" + strings.TrimPrefix(raw, "//")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("разбор адреса %q: %w", raw, err)
	}

	if parsed.Host == "" {
		return fmt.Errorf("в адресе %q не указан хост", raw)
	}

	*u.value = strings.TrimSuffix(raw, "/")

	return nil
}

func Parse(args []string, getenv func(string) string) (Config, error) {
	cfg := Config{ServerAddr: "http://localhost:8080"}
	flags := flag.NewFlagSet("agent", flag.ContinueOnError)

	flags.Var(serverURL{value: &cfg.ServerAddr}, "a", "address and port to metrics server")
	flags.Int64Var(&cfg.ReportInterval, "r", 10, "interval between metric reports in seconds")
	flags.Int64Var(&cfg.PollInterval, "p", 2, "interval between metric polls in seconds")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	if raw := getenv(EnvAddress); raw != "" {
		if err := (serverURL{value: &cfg.ServerAddr}).Set(raw); err != nil {
			return Config{}, fmt.Errorf("%s: %w", EnvAddress, err)
		}
	}

	if raw := getenv(EnvReportInterval); raw != "" {
		interval, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", EnvReportInterval, err)
		}
		cfg.ReportInterval = interval
	}

	if raw := getenv(EnvPollInterval); raw != "" {
		interval, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", EnvPollInterval, err)
		}
		cfg.PollInterval = interval
	}

	return cfg, nil
}
