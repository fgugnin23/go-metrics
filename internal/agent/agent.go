package agent

import (
	"math/rand"
	"time"
)

const sendMetricsAfterPollCount = reportInterval / pollInterval

func Run() {
	gauges := newRuntimeGauges()
	pollCount := CounterMetric{name: "PollCount"}
	randomValue := GaugeMetric{name: "RandomValue"}

	for {
		pollCount.value += 1
		randomValue.value = rand.NormFloat64()

		collectRuntimeMetrics(gauges)

		if pollCount.value%sendMetricsAfterPollCount == 0 {
			reportMetrics(gauges, pollCount, randomValue)

			println("Metrics sent to server: ", pollCount.value/sendMetricsAfterPollCount, " times")
		}

		time.Sleep(pollInterval * time.Second)
	}
}

// reportMetrics отправляет на сервер все собранные метрики.
func reportMetrics(gauges []GaugeMetric, pollCount CounterMetric, randomValue GaugeMetric) {
	for _, gaugeMetric := range gauges {
		sendMetric(
			gaugeURL(gaugeMetric),
			"Error when making gauge runtime metric request",
			"Error when getting gauge runtime metric response",
		)
	}

	sendMetric(
		counterURL(pollCount),
		"Error when sending poll count metric",
		"Error when getting gauge runtime metric response",
	)

	sendMetric(
		gaugeURL(randomValue),
		"Error when sending poll count metric",
		"Error when getting gauge runtime metric response",
	)
}
