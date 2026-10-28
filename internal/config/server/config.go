package config

import "flag"

var FlagRunAddr string

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.Parse()
}
