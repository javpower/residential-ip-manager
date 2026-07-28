package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/config"
	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
	minivpnconfig "github.com/ooni/minivpn/pkg/config"
	minivpntunnel "github.com/ooni/minivpn/pkg/tunnel"
	wgtun "golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

const (
	defaultHandshakeTimeout = 45 * time.Second
	defaultTunnelMTU        = 1420
	openVPNPingInterval     = 3 * time.Second
)

var openVPNPingPayload = []byte{0x2a, 0x18, 0x7b, 0xf3, 0x64, 0x1e, 0xb4, 0xcb, 0x07, 0xed, 0x2d, 0x0a, 0x98, 0x1f, 0xc7, 0x48}

type EnvironmentReport struct {
	OS            string `json:"os"`
	Engine        string `json:"engine"`
	BuiltIn       bool   `json:"built_in"`
	RuntimeDir    string `json:"runtime_dir"`
	PrivilegeHint string `json:"privilege_hint"`
	Ready         bool   `json:"ready"`
	Running       bool   `json:"running"`
	Connecting    bool   `json:"connecting"`
	LocalIP       string `json:"local_ip,omitempty"`
	Gateway       string `json:"gateway,omitempty"`
	NodeID        string `json:"node_id,omitempty"`
	PacketsOut    uint64 `json:"packets_out"`
	PacketsIn     uint64 `json:"packets_in"`
	BytesOut      uint64 `json:"bytes_out"`
	BytesIn       uint64 `json:"bytes_in"`
	Message       string `json:"message"`
}

// OpenVPNController owns an in-process OpenVPN client and a userspace TCP/IP
// stack. DialContext is the only egress used by the embedded proxy core.
type OpenVPNController struct {
	cfg        config.OpenVPNConfig
	runtimeDir string

	mu         sync.RWMutex
	connecting bool
	session    *vpnSession
	lastError  string
}

type vpnSession struct {
	nodeID  string
	localIP string
	gateway string

	vpn         *minivpntunnel.TUN
	device      wgtun.Device
	network     *netstack.Net
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan error
	closed      chan struct{}
	wg          sync.WaitGroup
	closeOnce   sync.Once
	connMu      sync.Mutex
	connCond    *sync.Cond
	connections map[net.Conn]struct{}
	closing     bool
	activeDials int
	packetsOut  atomic.Uint64
	packetsIn   atomic.Uint64
	bytesOut    atomic.Uint64
	bytesIn     atomic.Uint64
}

func NewOpenVPNController(cfg config.OpenVPNConfig, dataDir string) *OpenVPNController {
	return &OpenVPNController{
		cfg:        cfg,
		runtimeDir: filepath.Join(dataDir, "runtime"),
	}
}

func (c *OpenVPNController) CheckEnvironment() EnvironmentReport {
	c.mu.RLock()
	defer c.mu.RUnlock()
	report := EnvironmentReport{
		OS:            runtime.GOOS,
		Engine:        "minivpn-go+gvisor-netstack",
		BuiltIn:       true,
		RuntimeDir:    c.runtimeDir,
		PrivilegeHint: "用户态模式无需管理员权限，也无需安装 OpenVPN 或 TUN 驱动",
		Ready:         true,
		Connecting:    c.connecting,
		Message:       "内置 OpenVPN 协议引擎就绪",
	}
	if c.connecting {
		report.Message = "正在建立 VPNGate 隧道"
	}
	if c.session != nil {
		report.Running = true
		report.LocalIP = c.session.localIP
		report.Gateway = c.session.gateway
		report.NodeID = c.session.nodeID
		report.PacketsOut = c.session.packetsOut.Load()
		report.PacketsIn = c.session.packetsIn.Load()
		report.BytesOut = c.session.bytesOut.Load()
		report.BytesIn = c.session.bytesIn.Load()
		report.Message = "内置 OpenVPN 隧道已连接"
	} else if c.lastError != "" {
		report.Message = "上次连接失败: " + c.lastError
	}
	return report
}

func (c *OpenVPNController) Connect(ctx context.Context, node domain.VpnNode) error {
	c.mu.Lock()
	if c.session != nil {
		c.mu.Unlock()
		return fmt.Errorf("VPNGate tunnel is already connected")
	}
	if c.connecting {
		c.mu.Unlock()
		return fmt.Errorf("VPNGate tunnel connection is already in progress")
	}
	c.connecting = true
	c.lastError = ""
	c.mu.Unlock()

	session, err := c.connect(ctx, node)
	c.mu.Lock()
	c.connecting = false
	if err != nil {
		c.lastError = err.Error()
		c.mu.Unlock()
		return err
	}
	c.session = session
	c.mu.Unlock()

	go c.watch(session)
	return nil
}

func (c *OpenVPNController) connect(ctx context.Context, node domain.VpnNode) (*vpnSession, error) {
	if node.OpenVPNConfig == "" {
		return nil, fmt.Errorf("node does not contain OpenVPN config")
	}
	if err := os.MkdirAll(c.runtimeDir, 0o700); err != nil {
		return nil, err
	}

	cleaned, err := SanitizeOpenVPNConfig(node.OpenVPNConfig)
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(c.runtimeDir, safeFilename(node.ID)+".ovpn")
	if err := os.WriteFile(configPath, []byte(cleaned), 0o600); err != nil {
		return nil, err
	}
	options, err := minivpnconfig.ReadConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("parse VPNGate profile: %w", err)
	}
	options.Username = valueOr(c.cfg.Username, "vpn")
	options.Password = valueOr(c.cfg.Password, "vpn")
	if options.Remote == "" || options.Port == "" {
		return nil, fmt.Errorf("VPNGate profile does not contain a valid remote")
	}
	if options.Cipher == "" {
		return nil, fmt.Errorf("VPNGate profile does not declare a supported cipher")
	}
	if options.Auth == "" {
		return nil, fmt.Errorf("VPNGate profile does not declare a supported auth digest")
	}
	if !options.HasAuthInfo() {
		return nil, fmt.Errorf("VPNGate profile does not contain usable credentials or certificates")
	}

	timeout := time.Duration(c.cfg.ConnectTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultHandshakeTimeout
	}
	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	vpn, err := minivpntunnel.Start(
		handshakeCtx,
		&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second},
		minivpnconfig.NewConfig(minivpnconfig.WithOpenVPNOptions(options)),
	)
	if err != nil {
		return nil, fmt.Errorf("built-in OpenVPN handshake: %w", err)
	}

	localIP, err := netip.ParseAddr(vpn.LocalAddr().String())
	if err != nil || !localIP.Is4() {
		_ = vpn.Close()
		return nil, fmt.Errorf("VPNGate supplied invalid tunnel address %q", vpn.LocalAddr())
	}
	dnsServers, err := configuredDNS(c.cfg.DNSServers)
	if err != nil {
		_ = vpn.Close()
		return nil, err
	}
	mtu := c.cfg.MTU
	if mtu <= 0 {
		mtu = defaultTunnelMTU
	}
	device, network, err := netstack.CreateNetTUN([]netip.Addr{localIP}, dnsServers, mtu)
	if err != nil {
		_ = vpn.Close()
		return nil, fmt.Errorf("create userspace network stack: %w", err)
	}

	bridgeCtx, bridgeCancel := context.WithCancel(context.Background())
	session := &vpnSession{
		nodeID:      node.ID,
		localIP:     localIP.String(),
		gateway:     vpn.RemoteAddr().String(),
		vpn:         vpn,
		device:      device,
		network:     network,
		ctx:         bridgeCtx,
		cancel:      bridgeCancel,
		done:        make(chan error, 1),
		closed:      make(chan struct{}),
		connections: make(map[net.Conn]struct{}),
	}
	session.connCond = sync.NewCond(&session.connMu)
	session.startBridge(bridgeCtx, mtu)
	return session, nil
}

func configuredDNS(values []string) ([]netip.Addr, error) {
	if len(values) == 0 {
		return []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("8.8.8.8")}, nil
	}
	servers := make([]netip.Addr, 0, len(values))
	for _, value := range values {
		ip, err := netip.ParseAddr(value)
		if err != nil {
			return nil, fmt.Errorf("invalid openvpn.dns_servers address %q", value)
		}
		servers = append(servers, ip)
	}
	return servers, nil
}

func (s *vpnSession) startBridge(ctx context.Context, mtu int) {
	packetSize := max(mtu+256, 2048)
	s.wg.Add(3)
	go func() {
		defer s.wg.Done()
		buffers := [][]byte{make([]byte, packetSize)}
		sizes := make([]int, 1)
		for {
			n, err := s.device.Read(buffers, sizes, 0)
			if err != nil {
				s.reportBridgeError(ctx, err)
				return
			}
			for i := 0; i < n; i++ {
				if ctx.Err() != nil {
					continue
				}
				if _, err := s.vpn.Write(buffers[i][:sizes[i]]); err != nil {
					if ctx.Err() != nil {
						continue
					}
					s.reportBridgeError(ctx, err)
					return
				}
				s.packetsOut.Add(1)
				s.bytesOut.Add(uint64(sizes[i]))
			}
		}
	}()
	go func() {
		defer s.wg.Done()
		packet := make([]byte, packetSize)
		for {
			n, err := s.vpn.Read(packet)
			if err != nil {
				s.reportBridgeError(ctx, err)
				return
			}
			s.packetsIn.Add(1)
			s.bytesIn.Add(uint64(n))
			if _, err := s.device.Write([][]byte{packet[:n]}, 0); err != nil {
				s.reportBridgeError(ctx, err)
				return
			}
		}
	}()
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(openVPNPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := s.vpn.Write(openVPNPingPayload); err != nil {
					s.reportBridgeError(ctx, err)
					return
				}
			}
		}
	}()
}

func (s *vpnSession) reportBridgeError(ctx context.Context, err error) {
	if err == nil || errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrClosed) {
		return
	}
	select {
	case <-ctx.Done():
	case s.done <- err:
	default:
	}
}

func (c *OpenVPNController) watch(session *vpnSession) {
	select {
	case err := <-session.done:
		c.mu.Lock()
		if c.session == session {
			c.session = nil
			c.lastError = "tunnel data path stopped: " + err.Error()
		}
		c.mu.Unlock()
		session.close()
	case <-session.closed:
	}
}

func (c *OpenVPNController) Disconnect() error {
	c.mu.Lock()
	session := c.session
	c.session = nil
	c.lastError = ""
	c.mu.Unlock()
	if session == nil {
		return nil
	}
	return session.close()
}

func (s *vpnSession) close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.closeConnections()
		if err := s.vpn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			closeErr = err
		}
		s.waitConnections()
		if err := s.device.Close(); err != nil && closeErr == nil && !errors.Is(err, os.ErrClosed) {
			closeErr = err
		}
		s.wg.Wait()
		close(s.closed)
	})
	return closeErr
}

// DialContext opens a TCP or UDP connection through the active VPNGate tunnel.
func (c *OpenVPNController) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	c.mu.RLock()
	stack := c.session
	c.mu.RUnlock()
	if stack == nil || stack.network == nil {
		return nil, fmt.Errorf("VPNGate exit is not connected")
	}
	return stack.dialContext(ctx, network, address)
}

func (s *vpnSession) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	s.connMu.Lock()
	if s.closing {
		s.connMu.Unlock()
		return nil, net.ErrClosed
	}
	s.activeDials++
	s.connMu.Unlock()

	dialCtx, cancel := context.WithCancel(ctx)
	stopSessionCancel := context.AfterFunc(s.ctx, cancel)
	conn, err := s.network.DialContext(dialCtx, network, address)
	stopSessionCancel()
	cancel()

	s.connMu.Lock()
	if err == nil && !s.closing {
		s.connections[conn] = struct{}{}
		s.activeDials--
		s.connCond.Broadcast()
		s.connMu.Unlock()
		return &trackedConn{Conn: conn, session: s}, nil
	}
	closing := s.closing
	s.connMu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	s.connMu.Lock()
	s.activeDials--
	s.connCond.Broadcast()
	s.connMu.Unlock()
	if closing && err == nil {
		err = net.ErrClosed
	}
	return nil, err
}

func (s *vpnSession) untrackConnection(conn net.Conn) {
	s.connMu.Lock()
	delete(s.connections, conn)
	if s.connCond != nil {
		s.connCond.Broadcast()
	}
	s.connMu.Unlock()
}

func (s *vpnSession) closeConnections() {
	s.connMu.Lock()
	s.closing = true
	s.cancel()
	connections := make([]net.Conn, 0, len(s.connections))
	for conn := range s.connections {
		connections = append(connections, conn)
	}
	s.connMu.Unlock()

	for _, conn := range connections {
		_ = conn.Close()
		s.untrackConnection(conn)
	}

}

func (s *vpnSession) waitConnections() {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	for len(s.connections) > 0 || s.activeDials > 0 {
		s.connCond.Wait()
	}
}

type trackedConn struct {
	net.Conn
	session *vpnSession
	once    sync.Once
}

func (c *trackedConn) Close() error {
	var err error
	c.once.Do(func() {
		err = c.Conn.Close()
		c.session.untrackConnection(c.Conn)
	})
	return err
}

func safeFilename(value string) string {
	if value == "" {
		return "node"
	}
	result := make([]rune, 0, len(value))
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9', char == '-', char == '_', char == '.':
			result = append(result, char)
		default:
			result = append(result, '_')
		}
	}
	return string(result)
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
