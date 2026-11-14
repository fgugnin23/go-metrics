package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"

	config "github.com/fgugnin23/go-metrics/internal/config/server"
	"github.com/fgugnin23/go-metrics/internal/handler"
	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
)

func main() {
	logger := zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	cfg, err := config.Parse(os.Args[1:], os.Getenv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		logger.Info().Err(err).Msg("invalid server configuration")
		os.Exit(1)
	}

	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	server := &http.Server{
		Addr:     cfg.RunAddr,
		Handler:  handler.NewRouter(metricsService, logger),
		ErrorLog: log.New(infoLogWriter{logger: logger}, "", 0),
	}

	logger.Info().Str("address", cfg.RunAddr).Msg("started server")
	err = server.ListenAndServe()
	if err != nil {
		logger.Info().Err(err).Msg("server stopped")
		os.Exit(1)
	}
}

type infoLogWriter struct {
	logger zerolog.Logger
}

func (w infoLogWriter) Write(p []byte) (int, error) {
	w.logger.Info().Msg(strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}
