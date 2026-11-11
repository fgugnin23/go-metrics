package agent

import (
	"log"
	"math/rand"
	"net/http"
	"time"
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
		if err := sendMetric(client, gaugeURL(cfg.ServerBaseURL, gaugeMetric)); err != nil {
			errs = append(errs, err)
		}
	}

	if err := sendMetric(client, counterURL(cfg.ServerBaseURL, s.pollCount)); err != nil {
		errs = append(errs, err)
	} else {
		s.pollCount.value = 0
	}

	if err := sendMetric(client, gaugeURL(cfg.ServerBaseURL, s.randomValue)); err != nil {
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
