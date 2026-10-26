package agent

type GaugeMetric struct {
	name  string
	value float64
}

type CounterMetric struct {
	name  string
	value int64
}

var runtimeGaugeNames = []string{
	"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"HeapAlloc",
	"HeapIdle",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"MCacheInuse",
	"MCacheSys",
	"MSpanSys",
	"Mallocs",
	"NextGC",
	"NumForcedGC",
	"NumGC",
	"OtherSys",
	"PauseTotalNs",
	"StackInuse",
	"StackSys",
	"Sys",
	"TotalAlloc",
}

func newRuntimeGauges() []GaugeMetric {
	gauges := make([]GaugeMetric, 0, len(runtimeGaugeNames))

	for _, name := range runtimeGaugeNames {
		gauges = append(gauges, GaugeMetric{name: name})
	}

	return gauges
}
