package handler

import (
	"cmp"
	"html/template"
	"io"
	"slices"
	"strconv"

	models "github.com/fgugnin23/go-metrics/internal/model"
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

func newMetricsPageData(metrics []models.Metrics) metricsPageData {
	var data metricsPageData
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			data.Gauges = append(data.Gauges, metricRow{
				Name:  metric.ID,
				Value: strconv.FormatFloat(*metric.Value, 'f', -1, 64),
			})
		case models.Counter:
			data.Counters = append(data.Counters, metricRow{
				Name:  metric.ID,
				Value: strconv.FormatInt(*metric.Delta, 10),
			})
		}
	}
	sortByName := func(a, b metricRow) int { return cmp.Compare(a.Name, b.Name) }
	slices.SortFunc(data.Gauges, sortByName)
	slices.SortFunc(data.Counters, sortByName)

	return data
}

func renderMetricsPage(w io.Writer, data metricsPageData) error {
	return metricsPage.Execute(w, data)
}
