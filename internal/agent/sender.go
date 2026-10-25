package agent

import "net/http"

func sendMetric(url, requestErrMsg, responseErrMsg string) {
	req, requestErr := http.NewRequest(http.MethodPost, url, nil)

	if requestErr != nil {
		panic(requestErrMsg)
	}

	request, responseErr := http.DefaultClient.Do(req)

	if responseErr != nil {
		panic(responseErrMsg)
	}

	defer request.Body.Close()
}
