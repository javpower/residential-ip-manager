package web

import (
	"testing"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/config"
	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
	"github.com/Guli-Joy/residential-ip-manager/internal/tunnel"
)

func TestConnectionCandidatesPreferAvailableStrictHome(t *testing.T) {
	latencyFast := 20
	latencySlow := 200
	now := time.Now()
	nodes := []domain.VpnNode{
		{ID: "candidate", Protocol: "tcp", Status: domain.NodeAvailable, PurityGrade: domain.PurityCandidate, LatencyMS: &latencyFast, Score: 100},
		{ID: "strict-slow", Protocol: "tcp", Status: domain.NodeAvailable, PurityGrade: domain.PurityStrictHome, LatencyMS: &latencySlow, Score: 10},
		{ID: "rejected", Protocol: "tcp", Status: domain.NodeAvailable, PurityGrade: domain.PurityRejected},
		{ID: "udp", Protocol: "udp", Status: domain.NodeAvailable, PurityGrade: domain.PurityStrictHome},
	}
	got := connectionCandidates(nodes, "", 3, now)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].ID != "strict-slow" {
		t.Fatalf("expected strict home first, got %s", got[0].ID)
	}
}

func TestConnectionCandidatesIgnoreSuspiciousLowLatency(t *testing.T) {
	suspicious := 1
	trusted := 80
	now := time.Now()
	nodes := []domain.VpnNode{
		{ID: "intercepted", Protocol: "tcp", Status: domain.NodeAvailable, PurityGrade: domain.PurityCandidate, LatencyMS: &suspicious, LatencySuspicious: true, AdvertisedPingMS: 300},
		{ID: "trusted", Protocol: "tcp", Status: domain.NodeAvailable, PurityGrade: domain.PurityCandidate, LatencyMS: &trusted},
	}
	got := connectionCandidates(nodes, "", 2, now)
	if len(got) != 2 || got[0].ID != "trusted" {
		t.Fatalf("expected trusted latency first, got %#v", got)
	}
}

func TestExitNeedsRecoveryWhenConnectedTunnelStops(t *testing.T) {
	server := &Server{
		state:  &AppState{snapshot: domain.ConnectionSnapshot{State: domain.StateConnected}},
		tunnel: tunnel.NewOpenVPNController(config.OpenVPNConfig{}, t.TempDir()),
	}
	if !server.exitNeedsRecovery() {
		t.Fatal("expected connected snapshot without a running tunnel to trigger recovery")
	}
	server.state.snapshot = domain.InitialSnapshot()
	if server.exitNeedsRecovery() {
		t.Fatal("idle snapshot must not trigger recovery")
	}
}

func TestNodeMaintenanceKeepsRunningTunnelConnected(t *testing.T) {
	previous := domain.ConnectionSnapshot{State: domain.StateError, ActiveNodeID: "stale", Metadata: map[string]string{"exit_country": "JP"}}
	got := nodeMaintenanceSnapshot(previous, tunnel.EnvironmentReport{Running: true, NodeID: "active-node"}, "节点已刷新")
	if got.State != domain.StateConnected || got.ActiveNodeID != "active-node" {
		t.Fatalf("running tunnel state was not preserved: %#v", got)
	}
	if got.Message != "节点已刷新" || got.Metadata["exit_country"] != "JP" {
		t.Fatalf("connection metadata was lost: %#v", got)
	}
}

func TestNodeMaintenanceResetsOnlyWhenTunnelIsStopped(t *testing.T) {
	got := nodeMaintenanceSnapshot(domain.ConnectionSnapshot{State: domain.StateConnected}, tunnel.EnvironmentReport{}, "探活完成")
	if got.State != domain.StateIdle || got.Message != "探活完成" {
		t.Fatalf("stopped tunnel should reset maintenance snapshot: %#v", got)
	}
}

func TestConnectionCandidatesSkipCooldown(t *testing.T) {
	now := time.Now()
	until := now.Add(time.Minute)
	nodes := []domain.VpnNode{
		{ID: "cooling", Protocol: "tcp", Status: domain.NodeCooldown, PurityGrade: domain.PurityStrictHome, CooldownUntil: &until},
		{ID: "ready", Protocol: "tcp", Status: domain.NodeAvailable, PurityGrade: domain.PurityStrictHome},
	}
	got := connectionCandidates(nodes, "", 5, now)
	if len(got) != 1 || got[0].ID != "ready" {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}

func TestMarkNodeFailureSetsCooldown(t *testing.T) {
	now := time.Now()
	nodes := []domain.VpnNode{{ID: "n1", Protocol: "tcp", Status: domain.NodeAvailable}}
	got := markNodeFailure(nodes, "n1", "failed", time.Minute, now)
	if got[0].Status != domain.NodeCooldown {
		t.Fatalf("expected cooldown status, got %s", got[0].Status)
	}
	if got[0].CooldownUntil == nil || !got[0].CooldownUntil.After(now) {
		t.Fatalf("expected cooldown timestamp")
	}
	if got[0].LastError != "failed" {
		t.Fatalf("expected last error to be recorded")
	}
}

func TestPreserveNodeCooldownsAcrossRefresh(t *testing.T) {
	now := time.Now()
	until := now.Add(time.Minute)
	previous := []domain.VpnNode{{ID: "failed", Status: domain.NodeCooldown, CooldownUntil: &until, LastError: "handshake timeout"}}
	fresh := []domain.VpnNode{{ID: "failed", Status: domain.NodeAvailable}, {ID: "new", Status: domain.NodeAvailable}}
	got := preserveNodeCooldowns(fresh, previous, now)
	if got[0].Status != domain.NodeCooldown || got[0].CooldownUntil != &until || got[0].LastError != "handshake timeout" {
		t.Fatalf("expected cooldown to survive refresh: %#v", got[0])
	}
	if got[1].Status != domain.NodeAvailable {
		t.Fatalf("new node state should be unchanged: %#v", got[1])
	}
}
