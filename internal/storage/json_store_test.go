package storage

import (
	"testing"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

func TestJSONStoreRestoresNodesAndResetsActiveSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := NewJSONStore(dir)
	now := time.Now()
	nodes := []domain.VpnNode{{ID: "n1", IP: "8.8.8.8", CountryCode: "US"}}
	if err := store.Save(nodes, domain.ConnectionSnapshot{
		State:          domain.StateConnected,
		Message:        "connected",
		ActiveNodeID:   "n1",
		ConnectedSince: &now,
		Metadata:       map[string]string{"mode": "test"},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "n1" {
		t.Fatalf("unexpected nodes: %#v", got.Nodes)
	}
	if got.Snapshot.State != domain.StateIdle {
		t.Fatalf("expected restored snapshot to be idle, got %s", got.Snapshot.State)
	}
	if got.Snapshot.ActiveNodeID != "" {
		t.Fatalf("expected active node to be cleared, got %q", got.Snapshot.ActiveNodeID)
	}
}
