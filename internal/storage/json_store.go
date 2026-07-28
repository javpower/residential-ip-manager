package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

type State struct {
	Version   int                       `json:"version"`
	Nodes     []domain.VpnNode          `json:"nodes"`
	Snapshot  domain.ConnectionSnapshot `json:"snapshot"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

type JSONStore struct {
	path string
}

func NewJSONStore(dataDir string) *JSONStore {
	return &JSONStore{path: filepath.Join(dataDir, "state.json")}
}

func (s *JSONStore) Load() (State, error) {
	var state State
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Version: 1, Snapshot: domain.InitialSnapshot()}, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Snapshot.State != domain.StateIdle {
		state.Snapshot = domain.InitialSnapshot()
		state.Snapshot.Message = "已恢复节点缓存，请重新连接"
	}
	if state.Snapshot.Metadata == nil {
		state.Snapshot.Metadata = map[string]string{}
	}
	return state, nil
}

func (s *JSONStore) Save(nodes []domain.VpnNode, snapshot domain.ConnectionSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	state := State{
		Version:   1,
		Nodes:     nodes,
		Snapshot:  snapshot,
		UpdatedAt: time.Now(),
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
