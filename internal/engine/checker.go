package engine

import (
	"context"
	"net"
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
		perRequestTimeout = 1000 * time.Millisecond
	}

	totalPing, successCount := c.pingHost(monitor.Hostname, count, timeout, perRequestTimeout)
	elapsed := float64(time.Since(start).Milliseconds())

	result := &CheckResult{
		PingMS: totalPing / float64(count),
	}

	if successCount > 0 {
		result.Status = model.StatusUP
		result.Msg = "ping OK"
	} else {
		result.Status = model.StatusDown
		result.Msg = "ping failed: all packets lost"
		result.PingMS = elapsed
	}

	return result, nil
}

func (c *PingChecker) pingHost(hostname string, count int, timeout, perRequestTimeout time.Duration) (totalPing float64, success int) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil || len(ips) == 0 {
		return 0, 0
	}

	targetIP := ips[0].IP.To4()
	if targetIP == nil {
		targetIP = ips[0].IP.To16()
	}

	addr := &net.IPAddr{IP: targetIP}

	for i := 0; i < count; i++ {
		start := time.Now()
		conn, err := net.DialIP("ip4:icmp", nil, addr)
		if err != nil {
			continue
		}

		conn.SetDeadline(time.Now().Add(perRequestTimeout))

		msg := buildICMPEchoRequest(uint16(i), uint16(i+1))
		_, err = conn.Write(msg)
		if err != nil {
			conn.Close()
			continue
		}

		reply := make([]byte, 128)
		_, err = conn.Read(reply)
		conn.Close()

		if err != nil {
			continue
		}

		elapsed := float64(time.Since(start).Milliseconds())
		totalPing += elapsed
		success++
	}

	return totalPing, success
}

func buildICMPEchoRequest(id, seq uint16) []byte {
	var msg [8]byte
	msg[0] = 8 // ICMP Echo
	msg[1] = 0
	msg[2] = 0 // Checksum placeholder
	msg[3] = 0
	msg[4] = byte(id >> 8)
	msg[5] = byte(id & 0xff)
	msg[6] = byte(seq >> 8)
	msg[7] = byte(seq & 0xff)

	checksum := icmpChecksum(msg[:])
	msg[2] = byte(checksum >> 8)
	msg[3] = byte(checksum & 0xff)

	return msg[:]
}

func icmpChecksum(data []byte) uint16 {
	sum := uint32(0)
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	return uint16(^sum)
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