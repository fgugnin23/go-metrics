package handler

import (
	"html/template"
	"io"
	"maps"
	"slices"
	"strconv"
)

const metricsPageTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Метрики</title>
</head>
<body>
	<h1>Список всех метрик</h1>
	<h3>Список gauge метрик</h3>
	<ul>{{range .Gauges}}<li><b>{{.Name}}: </b>{{.Value}}</li>{{end}}</ul>
	<h3>Список counter метрик</h3>
	<ul>{{range .Counters}}<li><b>{{.Name}}: </b>{{.Value}}</li>{{end}}</ul>
</body>
</html>
`

var metricsPage = template.Must(template.New("metrics").Parse(metricsPageTemplate))

type metricRow struct {
	Name  string
	Value string
}

type metricsPageData struct {
	Gauges   []metricRow
	Counters []metricRow
}

func newMetricsPageData(storage Storage) metricsPageData {
	gaugeMetrics := storage.GetGaugeMetrics()
	counterMetrics := storage.GetCounterMetrics()

	data := metricsPageData{
		Gauges:   make([]metricRow, 0, len(gaugeMetrics)),
		Counters: make([]metricRow, 0, len(counterMetrics)),
	}

	for _, name := range slices.Sorted(maps.Keys(gaugeMetrics)) {
		data.Gauges = append(data.Gauges, metricRow{
			Name:  name,
			Value: strconv.FormatFloat(gaugeMetrics[name], 'f', -1, 64),
		})
	}

	for _, name := range slices.Sorted(maps.Keys(counterMetrics)) {
		data.Counters = append(data.Counters, metricRow{
			Name:  name,
			Value: strconv.FormatInt(counterMetrics[name], 10),
		})
	}

	return data
}

func renderMetricsPage(w io.Writer, data metricsPageData) error {
	return metricsPage.Execute(w, data)
}
