package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
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

func ParseFlags(args []string) (Config, error) {
	cfg := Config{ServerAddr: "http://localhost:8080"}
	flags := flag.NewFlagSet("agent", flag.ContinueOnError)

	flags.Var(serverURL{value: &cfg.ServerAddr}, "a", "address and port to metrics server")
	flags.Int64Var(&cfg.ReportInterval, "r", 10, "send metrics to server frequency")
	flags.Int64Var(&cfg.PollInterval, "p", 2, "poll metrics from runtime frequency")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
