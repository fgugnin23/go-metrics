package handler

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func requestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(response, r)

			status := response.Status()
			if status == 0 {
				status = http.StatusOK
			}

			logger.Info().
				Str("uri", r.RequestURI).
				Str("method", r.Method).
				Float64("duration_ms", float64(time.Since(started))/float64(time.Millisecond)).
				Int("status", status).
				Int("size", response.BytesWritten()).
				Msg("HTTP request completed")
		})
	}
}

func recoverer(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if value := recover(); value != nil {
					if value == http.ErrAbortHandler {
						panic(value)
					}

					logger.Info().
						Str("panic", fmt.Sprint(value)).
						Str("stack", string(debug.Stack())).
						Msg("panic while handling HTTP request")

					if r.Header.Get("Connection") != "Upgrade" {
						w.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
