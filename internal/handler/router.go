package handler

import "net/http"

func NewRouter(storage Storage) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle(
		"POST /update/{metric_type}/{metric_name}/{metric_value}",
		NewUpdateMetricsHandle(storage),
	)

	return mux
}
