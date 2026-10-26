package main

import (
	"net/http"

	"github.com/fgugnin23/go-metrics/internal/handler"
	"github.com/fgugnin23/go-metrics/internal/repository"
)

func main() {
	storage := repository.NewMemStorage()

	http.Handle(
		"POST /update/{metric_type}/{metric_name}/{metric_value}",
		handler.NewUpdateMetricsHandle(storage),
	)
	println("started server on 8080")
	err := http.ListenAndServe(`:8080`, nil)
	if err != nil {
		panic(err)
	}
}
