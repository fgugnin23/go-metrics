package main

import (
	"net/http"

	"github.com/fgugnin23/go-metrics/internal/handler"
	"github.com/fgugnin23/go-metrics/internal/repository"
)

func main() {
	storage := repository.NewMemStorage()

	println("started server on 8080")
	err := http.ListenAndServe(`:8080`, handler.NewRouter(storage))
	if err != nil {
		panic(err)
	}
}
