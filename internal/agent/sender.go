package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	models "github.com/fgugnin23/go-metrics/internal/model"
)

func sendMetric(client http.Client, baseURL string, metric models.Metrics) error {
	body, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("encode metric %s: %w", metric.ID, err)
	}

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return fmt.Errorf("compress metric %s: %w", metric.ID, err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("compress metric %s: %w", metric.ID, err)
	}

	url := strings.TrimRight(baseURL, "/") + "/update"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(compressed.Bytes()))
	if err != nil {
		return fmt.Errorf("build request %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request %s: %w", url, err)
	}

	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("запрос на адрес %s: неожиданный статус %d", url, resp.StatusCode)
	}

	return nil
}
