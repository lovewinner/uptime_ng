package engine

import (
	"context"
	"fmt"
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

	perRequestTimeout := time.Duration(monitor.PingPerRequestTimeout) * time.Millisecond
	if perRequestTimeout <= 0 {
		perRequestTimeout = time.Second
	}
	deadlineSeconds := int((perRequestTimeout + time.Second - 1) / time.Second)
	if deadlineSeconds <= 0 {
		deadlineSeconds = 1
	}
	deadline := strconv.Itoa(deadlineSeconds)
	countStr := strconv.Itoa(count)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", countStr, "-W", deadline, monitor.Hostname)
	output, err := cmd.Output()
	elapsed := float64(time.Since(start).Milliseconds())

	result := &CheckResult{
		PingMS: elapsed,
	}

	if err != nil {
		result.Status = model.StatusDown
		if ctx.Err() == context.DeadlineExceeded {
			result.Msg = "ping failed: timeout"
		} else {
			result.Msg = "ping failed"
		}
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
		loss := 100 - int(float64(successCount)/float64(count)*100)
		result.Msg = fmt.Sprintf("ping OK, packet loss %d%% (%d/%d received)", loss, successCount, count)
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
	if monitor.DNSResolveServer != "" {
		server := monitor.DNSResolveServer
		if _, _, err := net.SplitHostPort(server); err != nil {
			server = net.JoinHostPort(server, "53")
		}
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "udp", server)
			},
		}
	}

	recordType := strings.ToUpper(strings.TrimSpace(monitor.DNSResolveType))
	if recordType == "" {
		recordType = "A"
	}
	values, err := lookupDNSRecord(ctx, resolver, recordType, monitor.Hostname)
	elapsed := float64(time.Since(start).Milliseconds())

	result := &CheckResult{
		PingMS: elapsed,
	}

	if err != nil {
		result.Status = model.StatusDown
		result.Msg = "DNS lookup failed: " + err.Error()
		return result, nil
	}

	if len(values) == 0 {
		result.Status = model.StatusDown
		result.Msg = "DNS lookup returned empty result"
		return result, nil
	}

	result.Status = model.StatusUP
	result.Msg = fmt.Sprintf("DNS %s resolved to %s", recordType, strings.Join(values, ", "))
	return result, nil
}

func lookupDNSRecord(ctx context.Context, resolver *net.Resolver, recordType string, hostname string) ([]string, error) {
	switch recordType {
	case "A":
		ips, err := resolver.LookupIP(ctx, "ip4", hostname)
		return ipStrings(ips), err
	case "AAAA":
		ips, err := resolver.LookupIP(ctx, "ip6", hostname)
		return ipStrings(ips), err
	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, hostname)
		if err != nil {
			return nil, err
		}
		return []string{cname}, nil
	case "MX":
		records, err := resolver.LookupMX(ctx, hostname)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(records))
		for _, record := range records {
			out = append(out, fmt.Sprintf("%s(%d)", record.Host, record.Pref))
		}
		return out, nil
	case "TXT":
		return resolver.LookupTXT(ctx, hostname)
	case "NS":
		records, err := resolver.LookupNS(ctx, hostname)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(records))
		for _, record := range records {
			out = append(out, record.Host)
		}
		return out, nil
	default:
		return resolver.LookupHost(ctx, hostname)
	}
}

func ipStrings(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
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
