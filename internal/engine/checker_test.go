package engine

import (
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
