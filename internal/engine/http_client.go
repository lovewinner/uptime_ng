package engine

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"uptime_ng/internal/model"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

func (h *HTTPClient) DoRequest(monitor *model.Monitor) (*CheckResult, error) {
	start := time.Now()

	timeout := time.Duration(monitor.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	method := monitor.Method
	if method == "" {
		method = "GET"
	}

	url := monitor.URL
	if url == "" {
		return &CheckResult{
			Status: model.StatusDown,
			PingMS: 0,
			Msg:    "URL is empty",
		}, nil
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: monitor.IgnoreTLS,
		},
	}

	if monitor.IgnoreTLS {
		h.client.Transport = transport
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var reqBody io.Reader
	if monitor.Body != "" {
		reqBody = strings.NewReader(monitor.Body)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, reqBody)
	if err != nil {
		return &CheckResult{
			Status: model.StatusDown,
			PingMS: 0,
			Msg:    "Failed to create request: " + err.Error(),
		}, nil
	}

	req.Header.Set("User-Agent", "uptime_ng/1.0")
	req.Header.Set("Accept", "text/html,application/json,*/*")

	if monitor.AuthMethod == "basic" && monitor.BasicAuthUser != "" {
		req.SetBasicAuth(monitor.BasicAuthUser, monitor.BasicAuthPass)
	}
	if monitor.AuthMethod == "bearer" && monitor.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+monitor.BearerToken)
	}

	if monitor.Headers != "" {
		headers := parseHeaders(monitor.Headers)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	if monitor.Body != "" && monitor.HTTPBodyEncoding == "json" {
		req.Header.Set("Content-Type", "application/json")
	} else if monitor.Body != "" && monitor.HTTPBodyEncoding == "form" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if monitor.Body != "" && monitor.HTTPBodyEncoding == "xml" {
		req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	}

	resp, err := h.client.Do(req)
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		return &CheckResult{
			Status: model.StatusDown,
			PingMS: elapsed,
			Msg:    "Request failed: " + err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	statusOK := model.CheckStatusCode(resp.StatusCode, monitor.AcceptedStatusCodes)

	result := &CheckResult{
		PingMS:     elapsed,
		HTTPStatus: int16(resp.StatusCode),
		Msg:        fmt.Sprintf("%d - %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
	}

	if !statusOK {
		result.Status = model.StatusDown
		result.Msg += " (unexpected status code)"
		return result, nil
	}

	if monitor.Keyword != "" {
		keywordFound := strings.Contains(bodyStr, monitor.Keyword)
		if monitor.InvertKeyword {
			if keywordFound {
				result.Status = model.StatusDown
				result.Msg += ", keyword found (inverted check failed)"
				return result, nil
			}
		} else {
			if !keywordFound {
				result.Status = model.StatusDown
				preview := bodyStr
				if len(preview) > 100 {
					preview = preview[:97] + "..."
				}
				result.Msg += ", keyword not found in response: " + preview
				return result, nil
			}
		}
	}

	result.Status = model.StatusUP
	return result, nil
}

func parseHeaders(raw string) map[string]string {
	result := make(map[string]string)
	if raw == "" {
		return result
	}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx > 0 {
			key := strings.TrimSpace(line[:colonIdx])
			val := strings.TrimSpace(line[colonIdx+1:])
			result[key] = val
		}
	}
	return result
}