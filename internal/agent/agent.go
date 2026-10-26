package agent

import (
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

	collectRuntimeMetrics(s.gauges)
}

func (s *agentState) report(client http.Client, cfg Config) []error {
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

func shouldReport(pollCount int64, cfg Config) bool {
	if cfg.PollInterval <= 0 {
		return true
	}

	pollsPerReport := int64(cfg.ReportInterval / cfg.PollInterval)
	if pollsPerReport <= 0 {
		return true
	}

	return pollCount%pollsPerReport == 0
}

func Run() {
	RunWithConfig(DefaultConfig())
}

func RunWithConfig(cfg Config) {
	state := newAgentState()
	client := http.Client{}

	for {
		state.poll()

		if shouldReport(state.totalPolls, cfg) {
			for _, err := range state.report(client, cfg) {
				println("Ошибка при отправке метрик: ", err.Error())
			}

			println("Метрики отправлены на сервер, опрос произошел: ", state.totalPolls, " раз")
		}

		time.Sleep(cfg.PollInterval)
	}
}
