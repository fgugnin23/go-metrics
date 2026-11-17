package handler

import (
	"compress/gzip"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")

		switch strings.ToLower(strings.TrimSpace(strings.Join(r.Header.Values("Content-Encoding"), ","))) {
		case "", "identity":
		case "gzip":
			defer r.Body.Close()
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "invalid gzip body", http.StatusBadRequest)
				return
			}
			defer reader.Close()

			r.Body = reader
			r.ContentLength = -1
			r.Header.Del("Content-Length")
			r.Header.Del("Content-Encoding")
		default:
			http.Error(w, "unsupported content encoding", http.StatusUnsupportedMediaType)
			return
		}

		if r.Method == http.MethodHead || !acceptsGzip(r.Header.Values("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		response := &gzipResponseWriter{ResponseWriter: w}
		defer response.Close()
		next.ServeHTTP(response, r)
	})
}

func acceptsGzip(headers []string) bool {
	var wildcard bool
	for _, header := range headers {
		for _, entry := range strings.Split(header, ",") {
			parts := strings.Split(entry, ";")
			encoding := strings.ToLower(strings.TrimSpace(parts[0]))
			if encoding != "gzip" && encoding != "*" {
				continue
			}

			quality := 1.0
			for _, parameter := range parts[1:] {
				name, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
				if ok && strings.EqualFold(strings.TrimSpace(name), "q") {
					var err error
					quality, err = strconv.ParseFloat(strings.TrimSpace(value), 64)
					if err != nil {
						quality = 0
					}
				}
			}

			allowed := quality > 0 && quality <= 1
			if encoding == "gzip" {
				return allowed
			}
			wildcard = allowed
		}
	}
	return wildcard
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	if status >= 100 && status < 200 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.wroteHeader = true

	contentType, _, _ := mime.ParseMediaType(w.Header().Get("Content-Type"))
	if (contentType == "application/json" || contentType == "text/html") &&
		w.Header().Get("Content-Encoding") == "" &&
		status != http.StatusNoContent && status != http.StatusResetContent && status != http.StatusNotModified {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.writer = gzip.NewWriter(w.ResponseWriter)
	}

	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(body))
		}
		w.WriteHeader(http.StatusOK)
	}
	if w.writer != nil {
		return w.writer.Write(body)
	}
	return w.ResponseWriter.Write(body)
}

func (w *gzipResponseWriter) Close() error {
	if w.writer != nil {
		return w.writer.Close()
	}
	return nil
}

func (w *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
