package config

import "flag"

const EnvAddress = "ADDRESS"

type Config struct {
	RunAddr string
}

func Parse(args []string, getenv func(string) string) (Config, error) {
	var cfg Config
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.StringVar(&cfg.RunAddr, "a", "localhost:8080", "address and port to run server")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	if raw := getenv(EnvAddress); raw != "" {
		cfg.RunAddr = raw
	}

	return cfg, nil
}
