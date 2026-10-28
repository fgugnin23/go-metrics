package main

import (
	"net/http"

	config "github.com/fgugnin23/go-metrics/internal/config/server"
	"github.com/fgugnin23/go-metrics/internal/handler"
	"github.com/fgugnin23/go-metrics/internal/repository"
)

func main() {
	storage := repository.NewMemStorage()
	config.ParseFlags()

	println("started server on " + config.FlagRunAddr)
	err := http.ListenAndServe(config.FlagRunAddr, handler.NewRouter(storage))
	if err != nil {
		panic(err)
	}
}
