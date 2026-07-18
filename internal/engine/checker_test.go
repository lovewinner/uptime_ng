package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uptime_ng/internal/model"
)

func TestHTTPCheckerAdvancedOptions(t *testing.T) {
	var sawCacheBust bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("_uptime_ng") != "" {
			sawCacheBust = true
		}
		if r.Header.Get("X-Test") != "yes" {
			t.Fatalf("missing custom header")
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	result, err := NewHTTPChecker().Check(&model.Monitor{
		Type:                model.MonitorTypeHTTP,
		URL:                 server.URL,
		Method:              http.MethodGet,
		Timeout:             5,
		MaxRedirects:        1,
		AcceptedStatusCodes: `["200"]`,
		Headers:             "X-Test: yes",
		Keyword:             "ok",
		CacheBust:           true,
		SaveResponse:        true,
		ResponseMaxLength:   20,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Status != model.StatusUP {
		t.Fatalf("status=%d msg=%s", result.Status, result.Msg)
	}
	if !sawCacheBust {
		t.Fatalf("cache bust query was not sent")
	}
	if !strings.Contains(result.Msg, "response:") {
		t.Fatalf("response preview missing: %s", result.Msg)
	}
}

func TestHTTPMonitorRequestHeadersAndAuth(t *testing.T) {
	req, err := requestForMonitor(context.Background(), &model.Monitor{
		Body: "a=1",
	}, http.MethodPost, "https://example.com")
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	monitor := &model.Monitor{
		Headers:          "X-Test: yes\nX-Mode: refactor",
		HTTPBodyEncoding: "form",
		AuthMethod:       "bearer",
		BearerToken:      "token",
		Body:             "a=1",
	}
	if err := applyMonitorAuth(context.Background(), &http.Client{}, req, monitor); err != nil {
		t.Fatalf("auth: %v", err)
	}
	applyMonitorHeaders(req, monitor)

	if req.Header.Get("Authorization") != "Bearer token" {
		t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
	}
	if req.Header.Get("X-Test") != "yes" || req.Header.Get("X-Mode") != "refactor" {
		t.Fatalf("custom headers not applied: %#v", req.Header)
	}
	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("content-type=%q", req.Header.Get("Content-Type"))
	}

	ntlmReq, err := requestForMonitor(context.Background(), &model.Monitor{}, http.MethodGet, "https://example.com")
	if err != nil {
		t.Fatalf("ntlm request: %v", err)
	}
	if err := applyMonitorAuth(context.Background(), &http.Client{}, ntlmReq, &model.Monitor{
		AuthMethod:    "ntlm",
		BasicAuthUser: "alice",
		BasicAuthPass: "secret",
		AuthDomain:    "DOMAIN",
	}); err != nil {
		t.Fatalf("ntlm auth: %v", err)
	}
	user, pass, ok := ntlmReq.BasicAuth()
	if !ok || user != `DOMAIN\alice` || pass != "secret" {
		t.Fatalf("ntlm basic auth user=%q pass=%q ok=%v", user, pass, ok)
	}
}

func TestTCPCheckerWithLocalListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	var p uint16
	fmt.Sscanf(port, "%d", &p)

	result, err := NewTCPChecker().Check(&model.Monitor{Hostname: "127.0.0.1", Port: p, Timeout: 2})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Status != model.StatusUP {
		t.Fatalf("status=%d msg=%s", result.Status, result.Msg)
	}
}

func TestDNSCheckerInvalidHost(t *testing.T) {
	result, err := NewDNSChecker().Check(&model.Monitor{
		Hostname:       "invalid.invalid.",
		DNSResolveType: "A",
		Timeout:        2,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Status != model.StatusDown {
		t.Fatalf("status=%d msg=%s", result.Status, result.Msg)
	}
}
