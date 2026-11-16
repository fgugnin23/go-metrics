package agent

import (
	"bytes"
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

	url := strings.TrimRight(baseURL, "/") + "/update"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")

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
