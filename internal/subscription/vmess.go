package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/Guli-Joy/residential-ip-manager/internal/config"
	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

type vmessLink struct {
	Version  string `json:"v"`
	Name     string `json:"ps"`
	Address  string `json:"add"`
	Port     string `json:"port"`
	ID       string `json:"id"`
	AlterID  string `json:"aid"`
	Security string `json:"scy"`
	Network  string `json:"net"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Path     string `json:"path"`
	TLS      string `json:"tls"`
	SNI      string `json:"sni"`
}

func VMESS(cfg config.SubscriptionConfig, nodes []domain.VpnNode) (string, error) {
	if !cfg.Enabled {
		return "", fmt.Errorf("subscription is disabled")
	}
	payload := vmessLink{
		Version:  "2",
		Name:     subscriptionName(nodes),
		Address:  ResolveHost(cfg.Host),
		Port:     fmt.Sprintf("%d", cfg.Port),
		ID:       cfg.UUID,
		AlterID:  fmt.Sprintf("%d", cfg.AlterID),
		Security: valueOr(cfg.Security, "auto"),
		Network:  valueOr(cfg.Network, "tcp"),
		Type:     "none",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	links := []string{"vmess://" + base64.StdEncoding.EncodeToString(data)}
	return base64.StdEncoding.EncodeToString([]byte(joinLines(links))), nil
}

func ClashYAML(cfg config.SubscriptionConfig, nodes []domain.VpnNode) (string, error) {
	if !cfg.Enabled {
		return "", fmt.Errorf("subscription is disabled")
	}
	out := "proxies:\n"
	out += fmt.Sprintf(
		"  - { name: \"%s\", type: vmess, server: \"%s\", port: %d, uuid: \"%s\", alterId: %d, cipher: \"%s\", network: \"%s\", udp: true }\n",
		yamlQuote(subscriptionName(nodes)),
		yamlQuote(ResolveHost(cfg.Host)),
		cfg.Port,
		yamlQuote(cfg.UUID),
		cfg.AlterID,
		yamlQuote(valueOr(cfg.Security, "auto")),
		yamlQuote(valueOr(cfg.Network, "tcp")),
	)
	return out, nil
}

func QuantumultX(cfg config.SubscriptionConfig, nodes []domain.VpnNode) (string, error) {
	if !cfg.Enabled {
		return "", fmt.Errorf("subscription is disabled")
	}
	if network := valueOr(cfg.Network, "tcp"); !strings.EqualFold(network, "tcp") {
		return "", fmt.Errorf("Quantumult X subscription currently supports tcp transport")
	}
	method := strings.ToLower(strings.TrimSpace(cfg.Security))
	switch method {
	case "aes-128-gcm", "chacha20-poly1305", "none":
	default:
		method = "chacha20-poly1305"
	}
	name := strings.NewReplacer(",", " ", "\n", " ", "\r", " ").Replace(subscriptionName(nodes))
	return fmt.Sprintf(
		"vmess=%s:%d, method=%s, password=%s, obfs=none, fast-open=false, udp-relay=true, tag=%s\n",
		ResolveHost(cfg.Host),
		cfg.Port,
		method,
		cfg.UUID,
		name,
	), nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func joinLines(lines []string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	return out
}

func subscriptionName(nodes []domain.VpnNode) string {
	strict := 0
	available := 0
	for _, node := range nodes {
		if node.PurityGrade == domain.PurityStrictHome {
			strict++
		}
		if node.Status == domain.NodeAvailable {
			available++
		}
	}
	if strict > 0 || available > 0 {
		return fmt.Sprintf("Residential IP Manager %d strict %d available", strict, available)
	}
	return "Residential IP Manager"
}

func yamlQuote(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func ResolveHost(host string) string {
	trimmed := strings.TrimSpace(host)
	if trimmed != "" && !strings.EqualFold(trimmed, "auto") {
		return trimmed
	}
	private, public := detectLocalHostCandidates()
	return resolveHostFromCandidates(private, public)
}

func resolveHostFromCandidates(private, public string) string {
	if private != "" {
		return private
	}
	if public != "" {
		return public
	}
	return "127.0.0.1"
}

func detectLocalHostCandidates() (string, string) {
	var private string
	var public string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", ""
	}
	for _, addr := range addrs {
		ip := extractIP(addr)
		if ip == nil || !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsUnspecified() || ip.To4() == nil {
			continue
		}
		if ip.IsPrivate() {
			if private == "" {
				private = ip.String()
			}
			continue
		}
		if public == "" {
			public = ip.String()
		}
	}
	return private, public
}

func extractIP(addr net.Addr) net.IP {
	switch value := addr.(type) {
	case *net.IPNet:
		return value.IP
	case *net.IPAddr:
		return value.IP
	default:
		return nil
	}
}

func splitLines(value string) []string {
	result := []string{}
	for _, line := range []byte(value) {
		_ = line
	}
	current := ""
	for _, r := range value {
		if r == '\n' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
			continue
		}
		current += string(r)
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
