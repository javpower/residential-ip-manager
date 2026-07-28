package source

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

var unsafeID = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

type VPNGateSource struct {
	APIURL     string
	MaxNodes   int
	HTTPClient *http.Client
}

func (s VPNGateSource) Fetch(ctx context.Context) ([]domain.VpnNode, error) {
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.APIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "residential-ip-manager-go/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vpngate returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	nodes, err := ParseVPNGateCSV(string(body))
	if err != nil {
		return nil, err
	}
	if s.MaxNodes > 0 && len(nodes) > s.MaxNodes {
		nodes = nodes[:s.MaxNodes]
	}
	return nodes, nil
}

func ParseVPNGateCSV(text string) ([]domain.VpnNode, error) {
	payload := csvPayload(text)
	if payload == "" {
		return nil, fmt.Errorf("vpngate CSV header not found")
	}
	reader := csv.NewReader(bytes.NewBufferString(payload))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("vpngate CSV has no rows")
	}
	header := rows[0]
	index := map[string]int{}
	for i, name := range header {
		index[strings.TrimPrefix(strings.TrimSpace(name), "#")] = i
	}
	required := []string{"IP", "OpenVPN_ConfigData_Base64"}
	for _, name := range required {
		if _, ok := index[name]; !ok {
			return nil, fmt.Errorf("vpngate CSV missing %s", name)
		}
	}

	seen := map[string]struct{}{}
	nodes := make([]domain.VpnNode, 0, len(rows)-1)
	for _, row := range rows[1:] {
		node, ok := rowToNode(row, index)
		if !ok {
			continue
		}
		if _, exists := seen[node.ID]; exists {
			continue
		}
		seen[node.ID] = struct{}{}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("vpngate CSV contains no safe public OpenVPN nodes")
	}
	return nodes, nil
}

func csvPayload(text string) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	headerIndex := -1
	for i, line := range lines {
		clean := strings.TrimPrefix(strings.TrimLeft(line, "\ufeff "), "#")
		if strings.HasPrefix(clean, "HostName,") {
			headerIndex = i
			break
		}
	}
	if headerIndex < 0 {
		return ""
	}
	out := []string{strings.TrimPrefix(strings.TrimLeft(lines[headerIndex], "\ufeff "), "#")}
	for _, line := range lines[headerIndex+1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func rowToNode(row []string, index map[string]int) (domain.VpnNode, bool) {
	value := func(name string) string {
		i, ok := index[name]
		if !ok || i < 0 || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	publicIP := net.ParseIP(value("IP"))
	if publicIP == nil || !publicIP.IsGlobalUnicast() || isPrivateOrLoopback(publicIP) {
		return domain.VpnNode{}, false
	}
	decoded, err := decodeConfig(value("OpenVPN_ConfigData_Base64"))
	if err != nil {
		return domain.VpnNode{}, false
	}
	remoteHost, remotePort, protocol := parseRemote(decoded)
	remoteIP := net.ParseIP(remoteHost)
	if remoteIP == nil || !remoteIP.Equal(publicIP) || remotePort < 1 || remotePort > 65535 {
		return domain.VpnNode{}, false
	}
	if protocol != "tcp" && protocol != "udp" {
		return domain.VpnNode{}, false
	}
	countryCode := strings.ToUpper(value("CountryShort"))
	if countryCode == "" {
		countryCode = "XX"
	}
	id := unsafeID.ReplaceAllString(fmt.Sprintf("%s_%s_%d_%s", countryCode, publicIP.String(), remotePort, protocol), "_")
	return domain.VpnNode{
		ID:               strings.Trim(id, "_"),
		IP:               publicIP.String(),
		RemoteHost:       remoteIP.String(),
		RemotePort:       remotePort,
		Protocol:         protocol,
		CountryCode:      countryCode,
		Country:          value("CountryLong"),
		Score:            safeInt(value("Score")),
		AdvertisedPingMS: safeInt(value("Ping")),
		SpeedBPS:         safeInt(value("Speed")),
		Sessions:         safeInt(value("NumVpnSessions")),
		PurityGrade:      domain.PurityCandidate,
		Status:           domain.NodeUnknown,
		OpenVPNConfig:    decoded,
	}, true
}

func decodeConfig(encoded string) (string, error) {
	compact := strings.Join(strings.Fields(encoded), "")
	if compact == "" {
		return "", fmt.Errorf("empty OpenVPN config")
	}
	if rem := len(compact) % 4; rem != 0 {
		compact += strings.Repeat("=", 4-rem)
	}
	raw, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(raw), "\ufeff"), nil
}

func parseRemote(configText string) (string, int, string) {
	protocol := "unknown"
	for _, rawLine := range strings.Split(configText, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimPrefix(strings.ToLower(fields[0]), "--")
		if name == "proto" && len(fields) >= 2 {
			protocol = normalizeProtocol(fields[1])
		}
		if name == "remote" && len(fields) >= 3 {
			host := strings.Trim(fields[1], `"'`)
			port, _ := strconv.Atoi(fields[2])
			if len(fields) >= 4 {
				protocol = normalizeProtocol(fields[3])
			}
			return host, port, protocol
		}
	}
	return "", 0, protocol
}

func normalizeProtocol(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(value, "tcp") {
		return "tcp"
	}
	if strings.Contains(value, "udp") {
		return "udp"
	}
	return "unknown"
}

func safeInt(value string) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return n
}

func isPrivateOrLoopback(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()
}
