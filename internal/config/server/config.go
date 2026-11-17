package config

import (
	"flag"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	EnvAddress         = "ADDRESS"
	EnvStoreInterval   = "STORE_INTERVAL"
	EnvFileStoragePath = "FILE_STORAGE_PATH"
	EnvRestore         = "RESTORE"
)

type Config struct {
	RunAddr         string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
}

func Parse(args []string, lookupEnv func(string) (string, bool)) (Config, error) {
	var cfg Config
	var seconds int64
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.StringVar(&cfg.RunAddr, "a", "localhost:8080", "address and port to run server")
	flags.Int64Var(&seconds, "i", 300, "storage interval in seconds (reserved; currently saves after every update)")
	flags.StringVar(&cfg.FileStoragePath, "f", "metrics.json", "metrics storage file")
	flags.BoolVar(&cfg.Restore, "r", true, "restore metrics from file at startup")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	if raw, ok := lookupEnv(EnvAddress); ok && raw != "" {
		cfg.RunAddr = raw
	}
	if raw, ok := lookupEnv(EnvStoreInterval); ok {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", EnvStoreInterval, err)
		}
		seconds = value
	}
	if raw, ok := lookupEnv(EnvFileStoragePath); ok {
		cfg.FileStoragePath = raw
	}
	if raw, ok := lookupEnv(EnvRestore); ok {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", EnvRestore, err)
		}
		cfg.Restore = value
	}
	if seconds < 0 || seconds > math.MaxInt64/int64(time.Second) {
		return Config{}, fmt.Errorf("store interval is outside the supported duration range")
	}
	if cfg.FileStoragePath == "" {
		return Config{}, fmt.Errorf("file storage path must not be empty")
	}
	cfg.StoreInterval = time.Duration(seconds) * time.Second
	return cfg, nil
}
