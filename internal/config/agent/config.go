package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
)

var ServerAddr string
var ReportInterval int64
var PollInterval int64

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

func ParseFlags() {
	addr := serverURL{value: &ServerAddr}
	if err := addr.Set("localhost:8080"); err != nil {
		panic(err)
	}

	flag.Var(addr, "a", "address and port to metrics server")
	flag.Int64Var(&ReportInterval, "r", 10, "send metrics to server frequency")
	flag.Int64Var(&PollInterval, "p", 2, "poll metrics from runtime frequency")
	flag.Parse()
}
