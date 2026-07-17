package engine

import (
	"context"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"uptime_ng/internal/model"
)

type CheckResult struct {
	Status     uint16
	PingMS     float64
	Msg        string
	HTTPStatus int16
}

type Checker interface {
	Check(monitor *model.Monitor) (*CheckResult, error)
}

type HTTPChecker struct {
	client *HTTPClient
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		client: NewHTTPClient(),
	}
}

func (c *HTTPChecker) Check(monitor *model.Monitor) (*CheckResult, error) {
	return c.client.DoRequest(monitor)
}

type TCPChecker struct{}

func NewTCPChecker() *TCPChecker {
	return &TCPChecker{}
}

func (c *TCPChecker) Check(monitor *model.Monitor) (*CheckResult, error) {
	start := time.Now()
	timeout := time.Duration(monitor.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	addr := net.JoinHostPort(monitor.Hostname, uint16toa(monitor.Port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	elapsed := float64(time.Since(start).Milliseconds())

	result := &CheckResult{
		Status: model.StatusDown,
		PingMS: elapsed,
	}

	if err != nil {
		result.Msg = "TCP connection failed: " + err.Error()
		return result, nil
	}
	conn.Close()

	result.Status = model.StatusUP
	result.Msg = "TCP connection successful"
	return result, nil
}

type PingChecker struct{}

func NewPingChecker() *PingChecker {
	return &PingChecker{}
}

func (c *PingChecker) Check(monitor *model.Monitor) (*CheckResult, error) {
	start := time.Now()
	count := int(monitor.PingCount)
	if count <= 0 {
		count = 4
	}

	timeout := time.Duration(monitor.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	deadline := strconv.Itoa(int(timeout.Seconds()))
	if deadline == "0" {
		deadline = "30"
	}
	countStr := strconv.Itoa(count)

	cmd := exec.Command("ping", "-c", countStr, "-W", deadline, monitor.Hostname)
	output, err := cmd.Output()
	elapsed := float64(time.Since(start).Milliseconds())

	result := &CheckResult{
		PingMS: elapsed,
	}

	if err != nil {
		result.Status = model.StatusDown
		result.Msg = "ping failed"
		return result, nil
	}

	outStr := string(output)
	totalPing := 0.0
	successCount := 0

	lines := strings.Split(outStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "time=") || strings.Contains(line, "time<") {
			successCount++
			parts := strings.Split(line, " time=")
			if len(parts) < 2 {
				continue
			}
			timePart := strings.TrimSuffix(parts[1], " ms")
			timePart = strings.TrimSuffix(timePart, " µs")
			timePart = strings.TrimSuffix(timePart, " us")
			v, err := strconv.ParseFloat(timePart, 64)
			if err == nil {
				totalPing += v
			}
		}
	}

	if successCount > 0 {
		result.Status = model.StatusUP
		result.Msg = "ping OK"
		result.PingMS = totalPing / float64(successCount)
	} else {
		result.Status = model.StatusDown
		result.Msg = "ping failed: all packets lost"
	}

	return result, nil
}

type DNSChecker struct{}

func NewDNSChecker() *DNSChecker {
	return &DNSChecker{}
}

func (c *DNSChecker) Check(monitor *model.Monitor) (*CheckResult, error) {
	start := time.Now()
	timeout := time.Duration(monitor.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupHost(ctx, monitor.Hostname)
	elapsed := float64(time.Since(start).Milliseconds())

	result := &CheckResult{
		PingMS: elapsed,
	}

	if err != nil {
		result.Status = model.StatusDown
		result.Msg = "DNS lookup failed: " + err.Error()
		return result, nil
	}

	if len(ips) == 0 {
		result.Status = model.StatusDown
		result.Msg = "DNS lookup returned empty result"
		return result, nil
	}

	result.Status = model.StatusUP
	result.Msg = "DNS resolved to " + ips[0]
	return result, nil
}

func uint16toa(v uint16) string {
	if v == 0 {
		return "0"
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

func GetChecker(monitorType string) Checker {
	switch monitorType {
	case model.MonitorTypeHTTP:
		return NewHTTPChecker()
	case model.MonitorTypeTCP:
		return NewTCPChecker()
	case model.MonitorTypePing:
		return NewPingChecker()
	case model.MonitorTypeDNS:
		return NewDNSChecker()
	default:
		return nil
	}
}