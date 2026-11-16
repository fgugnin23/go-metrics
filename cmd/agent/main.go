package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/fgugnin23/go-metrics/internal/agent"
	config "github.com/fgugnin23/go-metrics/internal/config/agent"
)

func main() {
	cfg, err := config.Parse(os.Args[1:], os.Getenv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatal(err)
	}

	agent.RunWithConfig(agent.ArgsCollectorConfig(cfg), agent.ArgsSenderConfig(cfg))
}
