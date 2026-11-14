package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func NewRouter(service MetricsService, logger zerolog.Logger) *chi.Mux {
	metrics := NewMetricsHandler(service, logger)

	router := chi.NewRouter()
	router.Use(requestLogger(logger), recoverer(logger))

	router.Post("/update/{metric_type}/{metric_name}/{metric_value}", metrics.Update)
	router.Get("/value/{metric_type}/{metric_name}", metrics.Get)
	router.Get("/", metrics.MainPage)

	return router
}
