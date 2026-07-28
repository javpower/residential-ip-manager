package tunnel

import (
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strings"
)

var executionDirectives = map[string]struct{}{
	"auth-user-pass-verify": {},
	"client-connect":        {},
	"client-disconnect":     {},
	"down":                  {},
	"engine":                {},
	"ipchange":              {},
	"iproute":               {},
	"learn-address":         {},
	"plugin":                {},
	"providers":             {},
	"pkcs11-providers":      {},
	"route-pre-down":        {},
	"route-up":              {},
	"script-security":       {},
	"tls-crypt-v2-verify":   {},
	"tls-verify":            {},
	"up":                    {},
}

var replacedDirectives = map[string]struct{}{
	"auth-user-pass":             {},
	"block-outside-dns":          {},
	"http-proxy":                 {},
	"http-proxy-retry":           {},
	"http-proxy-user-pass":       {},
	"management":                 {},
	"management-client":          {},
	"management-client-auth":     {},
	"management-client-group":    {},
	"management-client-pf":       {},
	"management-client-user":     {},
	"management-external-cert":   {},
	"management-external-key":    {},
	"management-hold":            {},
	"management-log-cache":       {},
	"management-query-passwords": {},
	"management-signal":          {},
	"management-up-down":         {},
	"socks-proxy":                {},
	"socks-proxy-retry":          {},
	"register-dns":               {},
}

type RenderOpenVPNOptions struct {
	AuthPath  string
	UseProxy  bool
	ProxyHost string
	ProxyPort int
	BypassIPs []string
	GOOS      string
}

func RenderOpenVPNConfig(configText string, options RenderOpenVPNOptions) (string, error) {
	cleaned, err := SanitizeOpenVPNConfig(configText)
	if err != nil {
		return "", err
	}
	if options.AuthPath == "" {
		return "", fmt.Errorf("auth path is required")
	}
	goos := options.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	lines := []string{
		"",
		"# Managed by Residential IP Manager",
		fmt.Sprintf("auth-user-pass \"%s\"", openvpnPath(options.AuthPath)),
		"auth-nocache",
	}
	if options.UseProxy {
		proxyIP := net.ParseIP(options.ProxyHost)
		if proxyIP == nil || !proxyIP.IsLoopback() {
			return "", fmt.Errorf("proxy host must be a loopback IP")
		}
		if options.ProxyPort < 1 || options.ProxyPort > 65535 {
			return "", fmt.Errorf("proxy port must be between 1 and 65535")
		}
		lines = append(lines,
			"",
			"# Optional local proxy relay",
			fmt.Sprintf("socks-proxy %s %d", proxyIP.String(), options.ProxyPort),
			"socks-proxy-retry",
		)
	}
	if goos == "windows" {
		lines = append(lines, "", "# Windows DNS leak protection", "block-outside-dns", "register-dns")
	}
	for _, raw := range options.BypassIPs {
		ip := net.ParseIP(strings.TrimSpace(raw))
		if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() {
			return "", fmt.Errorf("bypass IP must be public: %s", raw)
		}
		if ip.To4() != nil {
			lines = append(lines, fmt.Sprintf("route %s 255.255.255.255 net_gateway", ip.String()))
		} else {
			lines = append(lines, fmt.Sprintf("route-ipv6 %s/128 net_gateway", ip.String()))
		}
	}
	return cleaned + strings.Join(lines, "\n") + "\n", nil
}

func SanitizeOpenVPNConfig(configText string) (string, error) {
	normalized := strings.ReplaceAll(strings.ReplaceAll(configText, "\r\n", "\n"), "\r", "\n")
	lines := []string{}
	skippedBlock := ""
	skipContinuation := false
	for _, rawLine := range strings.Split(normalized, "\n") {
		stripped := strings.TrimSpace(rawLine)
		lower := strings.ToLower(stripped)
		if skippedBlock != "" {
			if lower == "</"+skippedBlock+">" {
				skippedBlock = ""
			}
			continue
		}
		if strings.HasPrefix(lower, "<") && strings.HasSuffix(lower, ">") && !strings.HasPrefix(lower, "</") {
			blockName := strings.Fields(strings.Trim(lower, "<>"))
			if len(blockName) > 0 && (blockName[0] == "auth-user-pass" || blockName[0] == "http-proxy-user-pass") {
				skippedBlock = blockName[0]
				continue
			}
		}
		if skipContinuation {
			skipContinuation = strings.HasSuffix(strings.TrimRight(rawLine, " \t"), "\\")
			continue
		}
		name := directiveName(stripped)
		if _, ok := executionDirectives[name]; ok {
			skipContinuation = strings.HasSuffix(strings.TrimRight(rawLine, " \t"), "\\")
			continue
		}
		if _, ok := replacedDirectives[name]; ok {
			skipContinuation = strings.HasSuffix(strings.TrimRight(rawLine, " \t"), "\\")
			continue
		}
		lines = append(lines, strings.TrimRight(rawLine, " \t"))
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, ";") {
			return strings.Join(lines, "\n") + "\n", nil
		}
	}
	return "", fmt.Errorf("OpenVPN config is empty after sanitization")
}

func directiveName(line string) string {
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "<") {
		return ""
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(fields[0]), "--")
}

func openvpnPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), `"`, `\"`)
}
