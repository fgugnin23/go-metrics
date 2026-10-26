package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(storage Storage) *chi.Mux {
	metrics := NewMetricsHandler(storage)

	router := chi.NewRouter()
	router.Use(middleware.Logger, middleware.Recoverer)

	router.Post("/update/{metric_type}/{metric_name}/{metric_value}", metrics.Update)
	router.Get("/value/{metric_type}/{metric_name}", metrics.Get)

	return router
}
