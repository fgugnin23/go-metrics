package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	config "github.com/fgugnin23/go-metrics/internal/config/server"
	"github.com/fgugnin23/go-metrics/internal/handler"
	"github.com/fgugnin23/go-metrics/internal/repository"
	"github.com/fgugnin23/go-metrics/internal/service"
)

func main() {
	cfg, err := config.Parse(os.Args[1:], os.Getenv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatal(err)
	}

	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)

	log.Println("started server on " + cfg.RunAddr)
	err = http.ListenAndServe(cfg.RunAddr, handler.NewRouter(metricsService))
	if err != nil {
		log.Fatal(err)
	}
}
