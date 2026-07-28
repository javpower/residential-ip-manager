package proxycore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"testing"

	"github.com/Guli-Joy/residential-ip-manager/internal/config"
)

type testDialer struct{}

func (testDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

func TestRenderXrayConfigBuildsVMESSInbound(t *testing.T) {
	got, err := RenderXrayConfig(
		config.ProxyCoreConfig{Listen: "127.0.0.1", LogLevel: "debug"},
		config.SubscriptionConfig{
			Port:     10086,
			UUID:     "00000000-0000-0000-0000-000000000001",
			AlterID:  0,
			Security: "auto",
			Network:  "tcp",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Inbounds[0].Protocol != "vmess" {
		t.Fatalf("expected vmess inbound, got %s", got.Inbounds[0].Protocol)
	}
	if got.Inbounds[0].Listen != "127.0.0.1" || got.Inbounds[0].Port != 10086 {
		t.Fatalf("unexpected listen endpoint: %#v", got.Inbounds[0])
	}
	settings, ok := got.Inbounds[0].Settings.(xrayVMESSInboundSettings)
	if !ok {
		t.Fatalf("unexpected VMESS settings type: %T", got.Inbounds[0].Settings)
	}
	if settings.Clients[0].ID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("uuid was not rendered")
	}
	if got.Outbounds[0].Protocol != "freedom" {
		t.Fatalf("expected freedom outbound")
	}
}

func TestRenderXrayConfigAddsLoopbackSOCKSInbound(t *testing.T) {
	got, err := RenderXrayConfig(
		config.ProxyCoreConfig{Listen: "0.0.0.0", LocalSOCKSEnabled: true, LocalSOCKSListen: "127.0.0.1:1080"},
		config.SubscriptionConfig{Port: 10086, UUID: "00000000-0000-0000-0000-000000000001", Network: "tcp"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Inbounds) != 2 || got.Inbounds[1].Protocol != "socks" {
		t.Fatalf("expected VMESS and SOCKS inbounds: %#v", got.Inbounds)
	}
	if got.Inbounds[1].Listen != "127.0.0.1" || got.Inbounds[1].Port != 1080 {
		t.Fatalf("unexpected SOCKS endpoint: %#v", got.Inbounds[1])
	}
}

func TestRenderXrayConfigDoesNotEnableTrafficSniffing(t *testing.T) {
	got, err := RenderXrayConfig(
		config.ProxyCoreConfig{LocalSOCKSEnabled: true, LocalSOCKSListen: "127.0.0.1:1080"},
		config.SubscriptionConfig{Port: 10086, UUID: "00000000-0000-0000-0000-000000000001", Network: "tcp"},
	)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rendered), `"sniffing"`) {
		t.Fatalf("traffic sniffing must remain disabled: %s", rendered)
	}
}

func TestRenderXrayConfigRejectsInvalidPort(t *testing.T) {
	_, err := RenderXrayConfig(config.ProxyCoreConfig{}, config.SubscriptionConfig{UUID: "id", Port: 70000})
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestManagerStartsEmbeddedVMESSListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skip("local listeners are disabled by the test sandbox")
		}
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	manager := NewManager(
		config.ProxyCoreConfig{Enabled: true, Type: "embedded", Listen: "127.0.0.1", LogLevel: "none"},
		config.SubscriptionConfig{Port: port, UUID: "00000000-0000-4000-8000-000000000001", Network: "tcp"},
		t.TempDir(),
		testDialer{},
	)
	status, err := manager.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = manager.Stop() })
	if !status.Running || !status.BuiltIn {
		t.Fatalf("unexpected running status: %#v", status)
	}
	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", stringPort(port)))
	if err != nil {
		t.Fatalf("embedded VMESS listener is not reachable: %v", err)
	}
	_ = conn.Close()
	status, err = manager.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if status.Running {
		t.Fatalf("manager remained running after stop: %#v", status)
	}
}

func stringPort(port int) string {
	return fmt.Sprintf("%d", port)
}
