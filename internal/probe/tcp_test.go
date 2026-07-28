package probe

import (
	"context"
	"errors"
	"net"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

func TestTCPProbeMarksReachableNodeAvailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skip("local listeners are disabled by the test sandbox")
		}
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for i := 0; i < 3; i++ {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portText)
	nodes := []domain.VpnNode{{ID: "local", RemoteHost: "127.0.0.1", RemotePort: port, Protocol: "tcp"}}
	got := TCPProbe{Timeout: time.Second, MaxConcurrency: 1, Samples: 3}.Probe(context.Background(), nodes)
	if got[0].Status != domain.NodeAvailable {
		t.Fatalf("expected available, got %s (%s)", got[0].Status, got[0].LastError)
	}
	if got[0].LatencyMS == nil {
		t.Fatal("expected latency")
	}
	if got[0].ProbeAttempts != 3 || got[0].ProbeSuccesses != 3 {
		t.Fatalf("expected 3/3 samples, got %d/%d", got[0].ProbeSuccesses, got[0].ProbeAttempts)
	}
	if got[0].LatencySuspicious {
		t.Fatal("loopback latency must not be marked suspicious")
	}
}

func TestTCPProbeSkipsUDPNode(t *testing.T) {
	nodes := []domain.VpnNode{{ID: "udp", RemoteHost: "203.0.113.1", RemotePort: 1194, Protocol: "udp"}}
	got := TCPProbe{Timeout: time.Millisecond, MaxConcurrency: 1, Samples: 3}.Probe(context.Background(), nodes)
	if got[0].Status != domain.NodeUnknown || got[0].ProbeAttempts != 0 {
		t.Fatalf("expected UDP node to remain unknown without TCP attempts: %#v", got[0])
	}
}

func TestSuspiciousPublicLatency(t *testing.T) {
	if !suspiciousPublicLatency("203.0.113.1", 2) {
		t.Fatal("expected implausibly low public latency to be suspicious")
	}
	if suspiciousPublicLatency("203.0.113.1", 25) {
		t.Fatal("expected plausible public latency to be trusted")
	}
	if suspiciousPublicLatency("127.0.0.1", 1) {
		t.Fatal("loopback latency must not be suspicious")
	}
}

func TestCeilMillisecondsNeverReturnsZero(t *testing.T) {
	if got := ceilMilliseconds(100 * time.Microsecond); got != 1 {
		t.Fatalf("expected sub-millisecond duration to round up to 1, got %d", got)
	}
}

func TestBatchLatencyAnomalyMarksCrossCountryMeasurementsSuspicious(t *testing.T) {
	latency := 6
	nodes := make([]domain.VpnNode, 0, 10)
	for i := 0; i < 10; i++ {
		country := "JP"
		if i >= 5 {
			country = "US"
		}
		nodes = append(nodes, domain.VpnNode{
			RemoteHost:  "203.0.113.1",
			CountryCode: country,
			LatencyMS:   &latency,
		})
	}
	markBatchLatencyAnomaly(nodes)
	for _, node := range nodes {
		if !node.LatencySuspicious {
			t.Fatal("expected cross-country low-latency batch to be suspicious")
		}
	}
}
