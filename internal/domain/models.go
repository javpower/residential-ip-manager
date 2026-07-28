package domain

import "time"

type ConnectionState string

const (
	StateIdle          ConnectionState = "idle"
	StateFetchingNodes ConnectionState = "fetching_nodes"
	StateProbingNodes  ConnectionState = "probing_nodes"
	StateConnecting    ConnectionState = "connecting"
	StateVerifying     ConnectionState = "verifying"
	StateConnected     ConnectionState = "connected"
	StateFailingOver   ConnectionState = "failing_over"
	StateDisconnecting ConnectionState = "disconnecting"
	StateError         ConnectionState = "error"
)

type NodeStatus string

const (
	NodeUnknown     NodeStatus = "unknown"
	NodeChecking    NodeStatus = "checking"
	NodeAvailable   NodeStatus = "available"
	NodeUnavailable NodeStatus = "unavailable"
	NodeCooldown    NodeStatus = "cooldown"
)

type PurityGrade string

const (
	PurityRejected   PurityGrade = "rejected"
	PurityCandidate  PurityGrade = "candidate"
	PurityStrictHome PurityGrade = "strict_home"
)

type ResidentialEvidence struct {
	Provider  string    `json:"provider"`
	Passed    bool      `json:"passed"`
	Summary   string    `json:"summary"`
	CheckedAt time.Time `json:"checked_at"`
}

type VpnNode struct {
	ID                string                `json:"id"`
	IP                string                `json:"ip"`
	RemoteHost        string                `json:"remote_host"`
	RemotePort        int                   `json:"remote_port"`
	Protocol          string                `json:"protocol"`
	CountryCode       string                `json:"country_code"`
	Country           string                `json:"country"`
	City              string                `json:"city"`
	ISP               string                `json:"isp"`
	ASN               string                `json:"asn"`
	ReverseDNS        string                `json:"reverse_dns"`
	Score             int                   `json:"score"`
	AdvertisedPingMS  int                   `json:"advertised_ping_ms"`
	SpeedBPS          int                   `json:"speed_bps"`
	Sessions          int                   `json:"sessions"`
	LatencyMS         *int                  `json:"latency_ms"`
	LatencySuspicious bool                  `json:"latency_suspicious"`
	ProbeAttempts     int                   `json:"probe_attempts"`
	ProbeSuccesses    int                   `json:"probe_successes"`
	PurityGrade       PurityGrade           `json:"purity_grade"`
	PurityScore       int                   `json:"purity_score"`
	Status            NodeStatus            `json:"status"`
	Evidence          []ResidentialEvidence `json:"evidence"`
	LastCheckedAt     *time.Time            `json:"last_checked_at"`
	LastError         string                `json:"last_error"`
	CooldownUntil     *time.Time            `json:"cooldown_until"`
	OpenVPNConfig     string                `json:"-"`
}

type ConnectionSnapshot struct {
	State               ConnectionState   `json:"state"`
	Message             string            `json:"message"`
	ActiveNodeID        string            `json:"active_node_id,omitempty"`
	ExitIP              string            `json:"exit_ip"`
	ConnectedSince      *time.Time        `json:"connected_since"`
	LastVerifiedAt      *time.Time        `json:"last_verified_at"`
	ConsecutiveFailures int               `json:"consecutive_failures"`
	Metadata            map[string]string `json:"metadata"`
}

func InitialSnapshot() ConnectionSnapshot {
	return ConnectionSnapshot{
		State:    StateIdle,
		Message:  "准备就绪",
		Metadata: map[string]string{},
	}
}
