package handler

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"uptime_ng/internal/model"
)

// PingRangeRequest is a CreateMonitorRequest with an additional ip_range field.
// The ip_range can be:
//   - "192.168.1.1-192.168.1.254" (dash-separated range)
//   - "10.0.0.0/24" (CIDR notation)
//   - "192.168.1.1,192.168.1.2,192.168.1.3" (comma-separated list)
type PingRangeRequest struct {
	CreateMonitorRequest
	IPRange string `json:"ip_range" binding:"required"`
}

// PingRangeResult summarizes the batch creation.
type PingRangeResult struct {
	Total   int      `json:"total"`
	Created int      `json:"created"`
	Errors  []string `json:"errors"`
	MonitorIDs []uint `json:"monitor_ids"`
}

// CreatePingRange creates monitors for each IP in the given range.
func (h *MonitorHandler) CreatePingRange(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req PingRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid_request", err.Error())
		return
	}

	// Override type to ping
	req.Type = model.MonitorTypePing
	normalizeMonitorRequest(&req.CreateMonitorRequest)

	if validationErr, lookupErr := h.validateMonitorRequest(userID, 0, req.CreateMonitorRequest); lookupErr != nil {
		errorResponse(c, http.StatusInternalServerError, "monitor_validation_failed", lookupErr.Error())
		return
	} else if validationErr != nil {
		badRequest(c, validationErr.code, validationErr.message)
		return
	}

	ips, parseErrs := parseIPRange(req.IPRange)
	if len(ips) == 0 {
		errMsg := "no valid IPs found in range"
		if len(parseErrs) > 0 {
			errMsg = parseErrs[0]
		}
		badRequest(c, "invalid_ip_range", errMsg)
		return
	}

	result := PingRangeResult{
		Total:   len(ips),
		Created: 0,
		Errors:  make([]string, 0),
		MonitorIDs: make([]uint, 0),
	}

	for _, ip := range ips {
		monitorReq := req.CreateMonitorRequest
		monitorReq.Hostname = ip.String()
		if monitorReq.Name == "" {
			monitorReq.Name = ip.String()
		} else {
			monitorReq.Name = fmt.Sprintf("%s (%s)", monitorReq.Name, ip.String())
		}

		monitor := newMonitorFromRequest(userID, monitorReq)

		err := runTransaction(h.DB, func(tx *gorm.DB) error {
			if err := createMonitor(tx, &monitor); err != nil {
				return err
			}
			return attachMonitorAssociations(tx, monitor.ID, req.NotificationIDs, req.TagNames, req.TagColors)
		})

		if err != nil {
			errMsg := fmt.Sprintf("Failed to create monitor for %s: %v", ip.String(), err)
			log.Println(errMsg)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		if h.Scheduler != nil {
			if err := h.Scheduler.StartMonitor(&monitor); err != nil {
				errMsg := fmt.Sprintf("Failed to schedule monitor for %s: %v", ip.String(), err)
				log.Println(errMsg)
				result.Errors = append(result.Errors, errMsg)
				// Monitor was created but scheduler failed; deactivate it
				_ = rollbackCreatedMonitor(h.DB, h.Scheduler, monitor, errors.New(errMsg))
				continue
			}
		}

		result.Created++
		result.MonitorIDs = append(result.MonitorIDs, monitor.ID)
	}

	status := http.StatusCreated
	if result.Created == 0 && len(result.Errors) > 0 {
		status = http.StatusInternalServerError
	} else if len(result.Errors) > 0 {
		status = http.StatusMultiStatus // 207
	}

	c.JSON(status, result)
}

// parseIPRange parses the ip_range string into a list of net.IP.
//
// Supported formats:
//   - CIDR: "10.0.0.0/24" — enumerates all usable IPs in the subnet
//   - Range: "192.168.1.1-192.168.1.254" — enumerates inclusive range
//   - List:  "192.168.1.1, 192.168.1.2" — comma-separated IPs
func parseIPRange(raw string) ([]net.IP, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, []string{"ip_range is empty"}
	}

	// Try CIDR first
	if strings.Contains(raw, "/") {
		return parseCIDR(raw)
	}

	// Try dash range
	if strings.Contains(raw, "-") {
		return parseDashRange(raw)
	}

	// Try comma-separated list
	if strings.Contains(raw, ",") {
		return parseList(raw)
	}

	// Single IP
	ip := net.ParseIP(raw)
	if ip == nil {
		return nil, []string{fmt.Sprintf("invalid IP: %s", raw)}
	}
	return []net.IP{ip}, nil
}

func parseCIDR(cidr string) ([]net.IP, []string) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, []string{fmt.Sprintf("invalid CIDR: %s", cidr)}
	}

	var ips []net.IP
	ip := ipNet.IP.Mask(ipNet.Mask)
	inc := ipIncrementer(ip)

	for ip := ip; ipNet.Contains(ip); inc(&ip) {
		// Skip network address and broadcast address
		if ip.Equal(ipNet.IP.Mask(ipNet.Mask)) {
			continue
		}
		bcast := broadcastAddr(ipNet)
		if ip.Equal(bcast) {
			continue
		}
		ips = append(ips, copyIP(ip))
	}

	return ips, nil
}

func parseDashRange(dashRange string) ([]net.IP, []string) {
	parts := strings.SplitN(dashRange, "-", 2)
	if len(parts) != 2 {
		return nil, []string{fmt.Sprintf("invalid range format: %s", dashRange)}
	}

	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))

	if startIP == nil {
		return nil, []string{fmt.Sprintf("invalid start IP: %s", parts[0])}
	}
	if endIP == nil {
		return nil, []string{fmt.Sprintf("invalid end IP: %s", parts[1])}
	}

	// Ensure both are IPv4 for simple range enumeration
	start4 := startIP.To4()
	end4 := endIP.To4()
	if start4 == nil || end4 == nil {
		return nil, []string{"only IPv4 dash ranges are supported; use comma-separated list for IPv6"}
	}

	startUint := ip4ToUint32(start4)
	endUint := ip4ToUint32(end4)

	if startUint > endUint {
		return nil, []string{fmt.Sprintf("start IP is greater than end IP: %s > %s", parts[0], parts[1])}
	}

	// Cap at 500 IPs to avoid accidental huge ranges
	maxIPs := 1024
	count := int(endUint - startUint + 1)
	if count > maxIPs {
		return nil, []string{fmt.Sprintf("range too large (%d IPs); max %d", count, maxIPs)}
	}

	ips := make([]net.IP, 0, count)
	for u := startUint; u <= endUint; u++ {
		ips = append(ips, uint32ToIP4(u))
	}
	return ips, nil
}

func parseList(list string) ([]net.IP, []string) {
	parts := strings.Split(list, ",")
	ips := make([]net.IP, 0, len(parts))
	errs := make([]string, 0)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		ip := net.ParseIP(part)
		if ip == nil {
			errs = append(errs, fmt.Sprintf("invalid IP: %s", part))
			continue
		}
		ips = append(ips, ip)
	}
	return ips, errs
}

func ip4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIP4(u uint32) net.IP {
	return net.IPv4(byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
}

func ipIncrementer(ip net.IP) func(*net.IP) {
	ip4 := ip.To4()
	if ip4 != nil {
		return func(p *net.IP) {
			u := ip4ToUint32(*p)
			*p = uint32ToIP4(u + 1)
		}
	}
	return func(p *net.IP) {
		raw := []byte(*p)
		for i := len(raw) - 1; i >= 0; i-- {
			raw[i]++
			if raw[i] != 0 {
				break
			}
		}
		*p = net.IP(raw)
	}
}

func broadcastAddr(ipNet *net.IPNet) net.IP {
	mask := ipNet.Mask
	ip := ipNet.IP.Mask(mask)
	bcast := make(net.IP, len(ip))
	for i := range ip {
		bcast[i] = ip[i] | ^mask[i]
	}
	return bcast
}

func copyIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func parseUint16(s string) uint16 {
	v, _ := strconv.ParseUint(s, 10, 16)
	return uint16(v)
}
