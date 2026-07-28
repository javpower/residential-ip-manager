package proxycore

import (
	"context"
	"encoding/json"
	"fmt"
	stdnet "net"
	"strconv"
	"sync"

	"github.com/Guli-Joy/residential-ip-manager/internal/config"
	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/policy"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/json"
	_ "github.com/xtls/xray-core/proxy/freedom"
	_ "github.com/xtls/xray-core/proxy/socks"
	_ "github.com/xtls/xray-core/proxy/vmess/inbound"
	"github.com/xtls/xray-core/transport/internet"
	_ "github.com/xtls/xray-core/transport/internet/headers/noop"
	_ "github.com/xtls/xray-core/transport/internet/tcp"
	_ "github.com/xtls/xray-core/transport/internet/udp"
)

type ContextDialer interface {
	DialContext(ctx context.Context, network, address string) (stdnet.Conn, error)
}

type Status struct {
	Enabled    bool   `json:"enabled"`
	Type       string `json:"type"`
	BuiltIn    bool   `json:"built_in"`
	Running    bool   `json:"running"`
	ConfigPath string `json:"config_path"`
	LocalSOCKS string `json:"local_socks,omitempty"`
	Message    string `json:"message"`
}

type Manager struct {
	cfg    config.ProxyCoreConfig
	sub    config.SubscriptionConfig
	dialer ContextDialer

	mu       sync.Mutex
	instance *core.Instance
}

func NewManager(cfg config.ProxyCoreConfig, sub config.SubscriptionConfig, _ string, dialer ContextDialer) *Manager {
	return &Manager{cfg: cfg, sub: sub, dialer: dialer}
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked("")
}

func (m *Manager) Start(ctx context.Context) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.cfg.Enabled {
		return m.statusLocked("VMESS 服务未启用"), fmt.Errorf("proxy core is disabled")
	}
	if m.instance != nil {
		return m.statusLocked("内置 VMESS 服务已运行"), nil
	}
	if m.dialer == nil {
		return m.statusLocked("未配置 VPNGate 出口拨号器"), fmt.Errorf("proxy dialer is required")
	}
	if err := ctx.Err(); err != nil {
		return m.statusLocked("VMESS 服务启动已取消"), err
	}

	rendered, err := RenderXrayConfig(m.cfg, m.sub)
	if err != nil {
		return m.statusLocked("VMESS 配置生成失败"), err
	}
	data, err := json.Marshal(rendered)
	if err != nil {
		return m.statusLocked("VMESS 配置序列化失败"), err
	}
	internet.UseAlternativeSystemDialer(&vpnSystemDialer{dialer: m.dialer})
	instance, err := core.StartInstance("json", data)
	if err != nil {
		internet.UseAlternativeSystemDialer(nil)
		return m.statusLocked("内置 VMESS 服务启动失败"), err
	}
	m.instance = instance
	return m.statusLocked("内置 VMESS 服务已启动"), nil
}

func (m *Manager) Stop() (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instance == nil {
		return m.statusLocked("内置 VMESS 服务未运行"), nil
	}
	err := m.instance.Close()
	m.instance = nil
	internet.UseAlternativeSystemDialer(nil)
	return m.statusLocked("内置 VMESS 服务已停止"), err
}

func (m *Manager) statusLocked(message string) Status {
	status := Status{
		Enabled:    m.cfg.Enabled,
		Type:       "embedded-vmess",
		BuiltIn:    true,
		Running:    m.instance != nil,
		ConfigPath: "in-memory",
		Message:    message,
	}
	if m.cfg.LocalSOCKSEnabled {
		status.LocalSOCKS = m.cfg.LocalSOCKSListen
	}
	if message == "" {
		switch {
		case !m.cfg.Enabled:
			status.Message = "VMESS 服务未启用"
		case m.instance == nil:
			status.Message = "内置 VMESS 服务未运行"
		default:
			status.Message = "内置 VMESS 服务运行中"
		}
	}
	return status
}

type vpnSystemDialer struct {
	dialer ContextDialer
}

func (d *vpnSystemDialer) Dial(ctx context.Context, _ xnet.Address, destination xnet.Destination, _ *internet.SocketConfig) (stdnet.Conn, error) {
	return d.dialer.DialContext(ctx, destination.Network.SystemString(), destination.NetAddr())
}

func (*vpnSystemDialer) DestIpAddress() xnet.IP {
	return nil
}

type xrayConfig struct {
	Log       xrayLog        `json:"log"`
	Policy    map[string]any `json:"policy"`
	Inbounds  []xrayInbound  `json:"inbounds"`
	Outbounds []xrayOutbound `json:"outbounds"`
}

type xrayLog struct {
	LogLevel string `json:"loglevel"`
}

type xrayInbound struct {
	Tag            string             `json:"tag"`
	Listen         string             `json:"listen"`
	Port           int                `json:"port"`
	Protocol       string             `json:"protocol"`
	Settings       any                `json:"settings"`
	StreamSettings xrayStreamSettings `json:"streamSettings,omitempty"`
}

type xrayVMESSInboundSettings struct {
	Clients []xrayClient `json:"clients"`
}

type xraySOCKSInboundSettings struct {
	Auth string `json:"auth"`
	UDP  bool   `json:"udp"`
	IP   string `json:"ip"`
}

type xrayClient struct {
	ID       string `json:"id"`
	AlterID  int    `json:"alterId"`
	Security string `json:"security"`
}

type xrayStreamSettings struct {
	Network string `json:"network,omitempty"`
}

type xrayOutbound struct {
	Protocol string         `json:"protocol"`
	Tag      string         `json:"tag"`
	Settings map[string]any `json:"settings"`
}

func RenderXrayConfig(coreConfig config.ProxyCoreConfig, sub config.SubscriptionConfig) (xrayConfig, error) {
	if sub.UUID == "" {
		return xrayConfig{}, fmt.Errorf("subscription uuid is required")
	}
	if sub.Port < 1 || sub.Port > 65535 {
		return xrayConfig{}, fmt.Errorf("subscription port must be between 1 and 65535")
	}
	if valueOr(sub.Network, "tcp") != "tcp" {
		return xrayConfig{}, fmt.Errorf("embedded VMESS currently supports tcp transport")
	}
	inbounds := []xrayInbound{
		{
			Tag:      "vmess-in",
			Listen:   valueOr(coreConfig.Listen, "0.0.0.0"),
			Port:     sub.Port,
			Protocol: "vmess",
			Settings: xrayVMESSInboundSettings{Clients: []xrayClient{
				{ID: sub.UUID, AlterID: sub.AlterID, Security: valueOr(sub.Security, "auto")},
			}},
			StreamSettings: xrayStreamSettings{Network: "tcp"},
		},
	}
	if coreConfig.LocalSOCKSEnabled {
		host, rawPort, err := stdnet.SplitHostPort(coreConfig.LocalSOCKSListen)
		if err != nil {
			return xrayConfig{}, fmt.Errorf("invalid local SOCKS listen address: %w", err)
		}
		port, err := strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return xrayConfig{}, fmt.Errorf("invalid local SOCKS port")
		}
		inbounds = append(inbounds, xrayInbound{
			Tag:      "socks-in",
			Listen:   host,
			Port:     port,
			Protocol: "socks",
			Settings: xraySOCKSInboundSettings{Auth: "noauth", UDP: true, IP: host},
		})
	}
	return xrayConfig{
		Log:      xrayLog{LogLevel: valueOr(coreConfig.LogLevel, "warning")},
		Policy:   map[string]any{},
		Inbounds: inbounds,
		Outbounds: []xrayOutbound{
			{Protocol: "freedom", Tag: "vpngate-exit", Settings: map[string]any{}},
		},
	}, nil
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
