package agent

import (
	"fmt"
	"io"
	"net/http"
)

func sendMetric(client http.Client, url string) error {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("build request %s: %w", url, err)
	}

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
