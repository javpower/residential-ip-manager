package subscription

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Guli-Joy/residential-ip-manager/internal/config"
	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

func TestVMESSSubscriptionRendersSingleManagedInbound(t *testing.T) {
	cfg := config.SubscriptionConfig{
		Enabled:  true,
		Host:     "vpn.example.com",
		Port:     10086,
		UUID:     "00000000-0000-0000-0000-000000000001",
		Security: "auto",
		Network:  "tcp",
	}
	nodes := []domain.VpnNode{
		{ID: "a", Status: domain.NodeAvailable, PurityGrade: domain.PurityStrictHome},
		{ID: "b", Status: domain.NodeAvailable, PurityGrade: domain.PurityStrictHome},
	}
	got, err := VMESS(cfg, nodes)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(raw), "vmess://"); count != 1 {
		t.Fatalf("expected one managed inbound link, got %d: %s", count, raw)
	}
	if !strings.Contains(string(raw), "vmess://") {
		t.Fatalf("expected vmess link, got %s", raw)
	}
}

func TestClashYAMLUsesConfiguredInboundEndpoint(t *testing.T) {
	cfg := config.SubscriptionConfig{
		Enabled:  true,
		Host:     "vpn.example.com",
		Port:     10086,
		UUID:     "00000000-0000-0000-0000-000000000001",
		Security: "auto",
		Network:  "tcp",
	}
	got, err := ClashYAML(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"server: \"vpn.example.com\"", "port: 10086", "uuid: \"00000000-0000-0000-0000-000000000001\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in clash YAML: %s", want, got)
		}
	}
}

func TestQuantumultXUsesNativeVMESSFormat(t *testing.T) {
	cfg := config.SubscriptionConfig{
		Enabled:  true,
		Host:     "vpn.example.com",
		Port:     10086,
		UUID:     "00000000-0000-0000-0000-000000000001",
		Security: "auto",
		Network:  "tcp",
	}
	got, err := QuantumultX(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"vmess=vpn.example.com:10086",
		"method=chacha20-poly1305",
		"password=00000000-0000-0000-0000-000000000001",
		"obfs=none",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in Quantumult X subscription: %s", want, got)
		}
	}
}

func TestResolveHostSupportsAuto(t *testing.T) {
	got := ResolveHost("auto")
	if got == "" || got == "auto" {
		t.Fatalf("expected auto host to resolve to a real address, got %q", got)
	}
}

func TestResolveHostPrefersPrivateCandidate(t *testing.T) {
	if got := resolveHostFromCandidates("192.168.1.20", "203.0.113.8"); got != "192.168.1.20" {
		t.Fatalf("expected private candidate to win, got %q", got)
	}
}

func TestResolveHostFallsBackToPublicCandidate(t *testing.T) {
	if got := resolveHostFromCandidates("", "203.0.113.8"); got != "203.0.113.8" {
		t.Fatalf("expected public candidate to win, got %q", got)
	}
}

func TestResolveHostKeepsExplicitValue(t *testing.T) {
	if got := ResolveHost("vpn.example.com"); got != "vpn.example.com" {
		t.Fatalf("expected explicit host to be preserved, got %q", got)
	}
}
