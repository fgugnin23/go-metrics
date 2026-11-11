package config

import "flag"

type Config struct {
	RunAddr string
}

func ParseFlags(args []string) (Config, error) {
	var cfg Config
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.StringVar(&cfg.RunAddr, "a", "localhost:8080", "address and port to run server")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
