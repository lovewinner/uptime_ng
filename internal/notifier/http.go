package notifier

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func httpPost(url, contentType string, body []byte) ([]byte, error) {
	resp, err := httpClient.Post(url, contentType, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("http error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
