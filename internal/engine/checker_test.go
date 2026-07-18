package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestPingCommandConfigFromMonitor(t *testing.T) {
	cfg := pingCommandConfigFromMonitor(&model.Monitor{})
	if cfg.count != 4 {
		t.Fatalf("count=%d want 4", cfg.count)
	}
	if cfg.timeout != 30*time.Second {
		t.Fatalf("timeout=%s want 30s", cfg.timeout)
	}
	if cfg.deadlineSeconds != 1 {
		t.Fatalf("deadline=%d want 1", cfg.deadlineSeconds)
	}

	cfg = pingCommandConfigFromMonitor(&model.Monitor{
		Timeout:               5,
		PingCount:             6,
		PingPerRequestTimeout: 1501,
	})
	if cfg.count != 6 || cfg.timeout != 5*time.Second || cfg.deadlineSeconds != 2 {
		t.Fatalf("config=%+v", cfg)
	}
}

func TestParsePingOutput(t *testing.T) {
	output := strings.Join([]string{
		"64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=1.25 ms",
		"64 bytes from 127.0.0.1: icmp_seq=2 ttl=64 time=2.75 ms",
		"64 bytes from 127.0.0.1: icmp_seq=3 ttl=64 time<1 ms",
	}, "\n")

	stats := parsePingOutput(output)
	if stats.successCount != 3 {
		t.Fatalf("success=%d want 3", stats.successCount)
	}
	if stats.totalPing != 4.0 {
		t.Fatalf("totalPing=%f want 4.0", stats.totalPing)
	}
	if stats.avgPing() != 4.0/3.0 {
		t.Fatalf("avg=%f want %f", stats.avgPing(), 4.0/3.0)
	}

	empty := parsePingOutput("100% packet loss")
	if empty.successCount != 0 || empty.avgPing() != 0 {
		t.Fatalf("empty stats=%+v avg=%f", empty, empty.avgPing())
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

func TestDNSRecordFormatters(t *testing.T) {
	ips := ipStrings([]net.IP{net.ParseIP("192.0.2.1"), net.ParseIP("2001:db8::1")})
	if len(ips) != 2 || ips[0] != "192.0.2.1" || ips[1] != "2001:db8::1" {
		t.Fatalf("ips=%v", ips)
	}

	mx := mxRecordStrings([]*net.MX{{Host: "mx1.example.com.", Pref: 10}, {Host: "mx2.example.com.", Pref: 20}})
	if len(mx) != 2 || mx[0] != "mx1.example.com.(10)" || mx[1] != "mx2.example.com.(20)" {
		t.Fatalf("mx=%v", mx)
	}

	ns := nsRecordStrings([]*net.NS{{Host: "ns1.example.com."}, {Host: "ns2.example.com."}})
	if len(ns) != 2 || ns[0] != "ns1.example.com." || ns[1] != "ns2.example.com." {
		t.Fatalf("ns=%v", ns)
	}
}
