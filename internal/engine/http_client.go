package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/Azure/go-ntlmssp"

	"uptime_ng/internal/model"
)

type HTTPClient struct{}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{}
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

	client, err := httpClientForMonitor(monitor, timeout)
	if err != nil {
		return &CheckResult{Status: model.StatusDown, PingMS: 0, Msg: err.Error()}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if monitor.CacheBust {
		url = addCacheBust(url, start)
	}

	req, err := requestForMonitor(ctx, monitor, method, url)
	if err != nil {
		return &CheckResult{
			Status: model.StatusDown,
			PingMS: 0,
			Msg:    "Failed to create request: " + err.Error(),
		}, nil
	}

	if err := applyMonitorAuth(ctx, client, req, monitor); err != nil {
		return &CheckResult{
			Status: model.StatusDown,
			PingMS: float64(time.Since(start).Milliseconds()),
			Msg:    "OAuth2 token request failed: " + err.Error(),
		}, nil
	}
	applyMonitorHeaders(req, monitor)

	resp, err := client.Do(req)
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
		if monitor.SaveErrorResponse {
			result.Msg += responsePreview(bodyStr, monitor.ResponseMaxLength)
		}
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
				if monitor.SaveErrorResponse {
					result.Msg += responsePreview(bodyStr, monitor.ResponseMaxLength)
				}
				return result, nil
			}
		}
	}

	result.Status = model.StatusUP
	if monitor.SaveResponse {
		result.Msg += responsePreview(bodyStr, monitor.ResponseMaxLength)
	}
	return result, nil
}

func httpClientForMonitor(monitor *model.Monitor, timeout time.Duration) (*http.Client, error) {
	maxRedirects := int(monitor.MaxRedirects)
	if maxRedirects <= 0 {
		maxRedirects = model.DefaultHTTPMaxRedirects
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: monitor.IgnoreTLS,
	}
	if monitor.AuthMethod == "mtls" {
		cert, err := tls.X509KeyPair([]byte(monitor.TLSCert), []byte(monitor.TLSKey))
		if err != nil {
			return nil, fmt.Errorf("Failed to load mTLS certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if monitor.TLSCa != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(monitor.TLSCa)) {
			return nil, fmt.Errorf("Failed to load TLS CA")
		}
		tlsConfig.RootCAs = pool
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}
	if monitor.AuthMethod == "ntlm" {
		client.Transport = ntlmssp.Negotiator{RoundTripper: transport}
	} else {
		client.Transport = transport
	}
	return client, nil
}

func requestForMonitor(ctx context.Context, monitor *model.Monitor, method string, rawURL string) (*http.Request, error) {
	var reqBody io.Reader
	if monitor.Body != "" {
		reqBody = strings.NewReader(monitor.Body)
	}
	return http.NewRequestWithContext(ctx, strings.ToUpper(method), rawURL, reqBody)
}

func applyMonitorAuth(ctx context.Context, client *http.Client, req *http.Request, monitor *model.Monitor) error {
	switch monitor.AuthMethod {
	case "basic":
		if monitor.BasicAuthUser != "" {
			req.SetBasicAuth(monitor.BasicAuthUser, monitor.BasicAuthPass)
		}
	case "ntlm":
		if monitor.BasicAuthUser != "" {
			user := monitor.BasicAuthUser
			if monitor.AuthDomain != "" && !strings.Contains(user, `\`) {
				user = monitor.AuthDomain + `\` + user
			}
			req.SetBasicAuth(user, monitor.BasicAuthPass)
		}
	case "bearer":
		if monitor.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+monitor.BearerToken)
		}
	case "oauth2-cc":
		token, err := fetchOAuthClientCredentialsToken(ctx, client, monitor)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func applyMonitorHeaders(req *http.Request, monitor *model.Monitor) {
	req.Header.Set("User-Agent", "uptime_ng/1.0")
	req.Header.Set("Accept", "text/html,application/json,*/*")
	for k, v := range parseHeaders(monitor.Headers) {
		req.Header.Set(k, v)
	}
	if monitor.Body == "" {
		return
	}
	switch monitor.HTTPBodyEncoding {
	case "json":
		req.Header.Set("Content-Type", "application/json")
	case "form":
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	case "xml":
		req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	}
}

func addCacheBust(rawURL string, t time.Time) string {
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := parsed.Query()
	q.Set("_uptime_ng", fmt.Sprintf("%d", t.UnixNano()))
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func fetchOAuthClientCredentialsToken(ctx context.Context, client *http.Client, monitor *model.Monitor) (string, error) {
	if monitor.OAuthTokenURL == "" || monitor.OAuthClientID == "" || monitor.OAuthClientSecret == "" {
		return "", fmt.Errorf("oauth token_url, client_id and client_secret are required")
	}
	form := neturl.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", monitor.OAuthClientID)
	form.Set("client_secret", monitor.OAuthClientSecret)
	if monitor.OAuthScopes != "" {
		form.Set("scope", monitor.OAuthScopes)
	}
	if monitor.OAuthAudience != "" {
		form.Set("audience", monitor.OAuthAudience)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, monitor.OAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if monitor.OAuthAuthMethod == "basic" {
		req.SetBasicAuth(monitor.OAuthClientID, monitor.OAuthClientSecret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("access_token missing in token response")
	}
	return parsed.AccessToken, nil
}

func responsePreview(body string, maxLen uint32) string {
	if body == "" {
		return ""
	}
	limit := int(maxLen)
	if limit <= 0 {
		limit = model.DefaultResponseMaxLen
	}
	if len(body) > limit {
		body = body[:limit] + "..."
	}
	return ", response: " + body
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
