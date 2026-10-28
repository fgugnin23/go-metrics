package config

import "flag"

var ServerAddr string
var ReportInterval int64
var PollInterval int64

func ParseFlags() {
	flag.StringVar(&ServerAddr, "a", "http://localhost:8080", "address and port to metrics server")
	flag.Int64Var(&ReportInterval, "r", 10, "send metrics to server frequency")
	flag.Int64Var(&PollInterval, "p", 2, "poll metrics from runtime frequency")
	flag.Parse()
}
