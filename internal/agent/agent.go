package agent

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

type agentState struct {
	gauges      []GaugeMetric
	pollCount   CounterMetric
	totalPolls  int64
	randomValue GaugeMetric
}

func newAgentState() agentState {
	return agentState{
		gauges:      newRuntimeGauges(),
		pollCount:   CounterMetric{name: "PollCount"},
		randomValue: GaugeMetric{name: "RandomValue"},
	}
}

func (s *agentState) poll() {
	s.pollCount.value += 1
	s.totalPolls += 1
	s.randomValue.value = rand.NormFloat64()

	err := collectRuntimeMetrics(s.gauges)

	if err != nil {
		log.Println(err)
	}
}

func (s *agentState) report(client http.Client, cfg SenderConfig) []error {
	var errs []error

	for _, gaugeMetric := range s.gauges {
		metric := models.Metrics{ID: gaugeMetric.name, MType: models.Gauge, Value: &gaugeMetric.value}
		if err := sendMetric(client, cfg.ServerBaseURL, metric); err != nil {
			errs = append(errs, err)
		}
	}

	pollCount := models.Metrics{ID: s.pollCount.name, MType: models.Counter, Delta: &s.pollCount.value}
	if err := sendMetric(client, cfg.ServerBaseURL, pollCount); err != nil {
		errs = append(errs, err)
	} else {
		s.pollCount.value = 0
	}

	randomValue := models.Metrics{ID: s.randomValue.name, MType: models.Gauge, Value: &s.randomValue.value}
	if err := sendMetric(client, cfg.ServerBaseURL, randomValue); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func shouldReport(pollCount int64, pollInterval, reportInterval time.Duration) bool {
	if pollInterval <= 0 {
		return true
	}

	pollsPerReport := int64(reportInterval / pollInterval)
	if pollsPerReport <= 0 {
		return true
	}

	return pollCount%pollsPerReport == 0
}

func RunWithConfig(collectorCfg CollectorConfig, senderCfg SenderConfig) {
	state := newAgentState()
	client := http.Client{}

	for {
		state.poll()

		if shouldReport(state.totalPolls, collectorCfg.PollInterval, senderCfg.ReportInterval) {
			for _, err := range state.report(client, senderCfg) {
				log.Println("Ошибка при отправке метрик: ", err.Error())
			}

			log.Println("Метрики отправлены на сервер, опрос произошел: ", state.totalPolls, " раз")
		}

		time.Sleep(collectorCfg.PollInterval)
	}
}
