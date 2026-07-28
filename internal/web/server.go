package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/classify"
	"github.com/Guli-Joy/residential-ip-manager/internal/config"
	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
	"github.com/Guli-Joy/residential-ip-manager/internal/probe"
	"github.com/Guli-Joy/residential-ip-manager/internal/proxycore"
	"github.com/Guli-Joy/residential-ip-manager/internal/source"
	"github.com/Guli-Joy/residential-ip-manager/internal/storage"
	"github.com/Guli-Joy/residential-ip-manager/internal/subscription"
	"github.com/Guli-Joy/residential-ip-manager/internal/tunnel"
	"github.com/Guli-Joy/residential-ip-manager/internal/verify"
)

type AppState struct {
	mu       sync.RWMutex
	cfg      config.Config
	nodes    []domain.VpnNode
	snapshot domain.ConnectionSnapshot
	sessions map[string]time.Time
	logs     []LogEntry
	store    *storage.JSONStore
}

type LogEntry struct {
	At      time.Time `json:"at"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type Server struct {
	cfg        config.Config
	state      *AppState
	logger     *slog.Logger
	tunnel     *tunnel.OpenVPNController
	proxy      *proxycore.Manager
	recoveryMu sync.Mutex
}

func NewServer(cfg config.Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	store := storage.NewJSONStore(cfg.DataDir())
	saved, err := store.Load()
	state := &AppState{
		cfg:      cfg,
		snapshot: domain.InitialSnapshot(),
		sessions: map[string]time.Time{},
		logs:     []LogEntry{},
		store:    store,
	}
	if err == nil {
		state.nodes = saved.Nodes
		state.snapshot = saved.Snapshot
		state.addLog("info", fmt.Sprintf("已从本地状态恢复 %d 个节点", len(saved.Nodes)))
	} else {
		state.addLog("warning", "本地状态恢复失败: "+err.Error())
	}
	state.addLog("info", "服务已启动，数据目录: "+cfg.DataDir())
	server := &Server{
		cfg:    cfg,
		state:  state,
		logger: logger,
		tunnel: tunnel.NewOpenVPNController(cfg.OpenVPN, cfg.DataDir()),
	}
	server.proxy = proxycore.NewManager(cfg.ProxyCore, cfg.Subscription, cfg.DataDir(), server.tunnel)
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/api/login", s.login)
	mux.HandleFunc("/api/logout", s.requireAuth(s.logout))
	mux.HandleFunc("/api/me", s.requireAuth(s.me))
	mux.HandleFunc("/api/status", s.requireAuth(s.status))
	mux.HandleFunc("/api/nodes", s.requireAuth(s.nodes))
	mux.HandleFunc("/api/nodes/refresh", s.requireAuth(s.refreshNodes))
	mux.HandleFunc("/api/nodes/classify", s.requireAuth(s.classifyNodes))
	mux.HandleFunc("/api/nodes/probe", s.requireAuth(s.probeNodes))
	mux.HandleFunc("/api/environment", s.requireAuth(s.environment))
	mux.HandleFunc("/api/exit/verify", s.requireAuth(s.verifyExit))
	mux.HandleFunc("/api/proxy/status", s.requireAuth(s.proxyStatus))
	mux.HandleFunc("/api/proxy/start", s.requireAuth(s.proxyStart))
	mux.HandleFunc("/api/proxy/stop", s.requireAuth(s.proxyStop))
	mux.HandleFunc("/api/connect", s.requireAuth(s.connect))
	mux.HandleFunc("/api/disconnect", s.requireAuth(s.disconnect))
	mux.HandleFunc("/api/logs", s.requireAuth(s.logs))
	mux.HandleFunc("/api/subscription/preview", s.requireAuth(s.subscriptionPreview))
	mux.HandleFunc("/sub/vmess", s.vmessSubscription)
	mux.HandleFunc("/sub/quantumult-x", s.quantumultXSubscription)
	mux.HandleFunc("/sub/clash", s.clashSubscription)
	return securityHeaders(mux)
}

func (s *Server) subscriptionHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/sub/vmess", s.vmessSubscription)
	mux.HandleFunc("/sub/quantumult-x", s.quantumultXSubscription)
	mux.HandleFunc("/sub/clash", s.clashSubscription)
	return securityHeaders(mux)
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	s.startGatewayAutomation(ctx)
	servers := []*http.Server{{
		Addr:              s.cfg.Server.Listen,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}}
	if listen := s.cfg.Subscription.Listen; s.cfg.Subscription.Enabled && listen != "" && listen != s.cfg.Server.Listen {
		servers = append(servers, &http.Server{
			Addr:              listen,
			Handler:           s.subscriptionHandler(),
			ReadHeaderTimeout: 10 * time.Second,
		})
	}
	errCh := make(chan error, len(servers))
	for _, server := range servers {
		s.logger.Info("listening", "addr", server.Addr)
		go func(server *http.Server) {
			errCh <- server.ListenAndServe()
		}(server)
	}
	var serveErr error
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			serveErr = err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, server := range servers {
		_ = server.Shutdown(shutdownCtx)
	}
	_, _ = s.proxy.Stop()
	_ = s.tunnel.Disconnect()
	return serveErr
}

func (s *Server) startGatewayAutomation(ctx context.Context) {
	if s.cfg.Deployment.AutoConnectExit && s.cfg.Failover.Enabled {
		go s.monitorExit(ctx)
	}
	if s.cfg.Deployment.AutoStartVMESS && s.cfg.ProxyCore.Enabled {
		go func() {
			status, err := s.proxy.Start(ctx)
			if err != nil {
				s.state.addLog("warning", "VMESS 自动启动失败: "+err.Error())
				s.logger.Warn("vmess autostart failed", "message", status.Message, "error", err)
				return
			}
			s.state.addLog("info", "VMESS 入口已自动启动")
		}()
	}
	if s.cfg.Deployment.AutoConnectExit {
		go func() {
			timer := time.NewTimer(750 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				s.autoConnectExit(ctx)
			}
		}()
	}
}

func (s *Server) monitorExit(ctx context.Context) {
	transportTicker := time.NewTicker(5 * time.Second)
	defer transportTicker.Stop()

	healthSeconds := s.cfg.Failover.HealthCheckSeconds
	if healthSeconds <= 0 {
		healthSeconds = 60
	}
	failureThreshold := s.cfg.Failover.FailureThreshold
	if failureThreshold <= 0 {
		failureThreshold = 2
	}
	healthTicker := time.NewTicker(time.Duration(healthSeconds) * time.Second)
	defer healthTicker.Stop()
	healthFailures := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-transportTicker.C:
			if s.exitNeedsRecovery() {
				healthFailures = 0
				s.recoverExit(ctx, "")
			}
		case <-healthTicker.C:
			if !s.cfg.Verify.Enabled || !s.exitIsRunning() {
				healthFailures = 0
				continue
			}
			reason, passed := s.checkExitHealth(ctx)
			if passed {
				healthFailures = 0
				continue
			}
			healthFailures++
			s.state.addLog("warning", fmt.Sprintf("VPNGate 出口健康检查失败 %d/%d: %s", healthFailures, failureThreshold, reason))
			if healthFailures >= failureThreshold {
				healthFailures = 0
				_ = s.tunnel.Disconnect()
				s.recoverExit(ctx, reason)
			}
		}
	}
}

func (s *Server) exitIsRunning() bool {
	s.state.mu.RLock()
	connected := s.state.snapshot.State == domain.StateConnected
	s.state.mu.RUnlock()
	return connected && s.tunnel.CheckEnvironment().Running
}

func (s *Server) checkExitHealth(ctx context.Context) (string, bool) {
	node, ok := s.activeNode()
	if !ok {
		return "活动节点状态缺失", false
	}
	result, err := s.verifyNodeExit(ctx, node)
	if err != nil {
		if _, reachabilityErr := s.checkExitReachability(ctx); reachabilityErr == nil {
			return "", true
		} else {
			return fmt.Sprintf("严格验证失败: %v；备用公网探测失败: %v", err, reachabilityErr), false
		}
	}
	if !result.Passed {
		return result.Message, false
	}
	s.recordVerification(result)
	return "", true
}

func (s *Server) checkExitReachability(ctx context.Context) (string, error) {
	timeout := time.Duration(s.cfg.Verify.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:         s.tunnel.DialContext,
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: timeout,
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("public IP probe returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("public IP probe returned invalid address %q", ip)
	}
	return ip, nil
}

func (s *Server) recordVerification(result verify.Result) {
	now := time.Now()
	s.state.mu.Lock()
	s.state.snapshot.LastVerifiedAt = &now
	s.state.snapshot.ExitIP = result.ExitIP
	if s.state.snapshot.Metadata == nil {
		s.state.snapshot.Metadata = map[string]string{}
	}
	s.state.snapshot.Metadata["exit_country"] = result.CountryCode
	s.state.snapshot.Metadata["exit_asn"] = result.ASN
	s.state.snapshot.Metadata["verify_passed"] = fmt.Sprintf("%v", result.Passed)
	s.state.saveLocked()
	s.state.mu.Unlock()
}

func (s *Server) exitNeedsRecovery() bool {
	s.state.mu.RLock()
	connected := s.state.snapshot.State == domain.StateConnected
	s.state.mu.RUnlock()
	environment := s.tunnel.CheckEnvironment()
	return connected && !environment.Running && !environment.Connecting
}

func (s *Server) recoverExit(ctx context.Context, reason string) {
	s.recoveryMu.Lock()
	defer s.recoveryMu.Unlock()
	if !s.exitNeedsRecovery() || ctx.Err() != nil {
		return
	}

	s.state.mu.RLock()
	failedNodeID := s.state.snapshot.ActiveNodeID
	s.state.mu.RUnlock()
	if reason == "" {
		environment := s.tunnel.CheckEnvironment()
		reason = environment.Message
		if reason == "" {
			reason = "VPNGate 数据通道已停止"
		}
	}
	if failedNodeID != "" {
		s.cooldownNode(failedNodeID, reason)
	}
	s.state.addLog("warning", "检测到 VPNGate 出口失效，正在自动切换节点")

	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	candidates := connectionCandidates(nodes, "", s.cfg.Failover.MaxAttempts, time.Now())
	for attempt, node := range candidates {
		s.state.setSnapshot(domain.ConnectionSnapshot{
			State:        domain.StateFailingOver,
			Message:      fmt.Sprintf("出口失效，正在切换节点 %d/%d：%s", attempt+1, len(candidates), node.IP),
			ActiveNodeID: node.ID,
			Metadata:     map[string]string{"attempt": fmt.Sprintf("%d/%d", attempt+1, len(candidates))},
		})
		if _, err := s.connectCandidate(ctx, node); err != nil {
			s.cooldownNode(node.ID, err.Error())
			continue
		}
		s.state.addLog("info", "VPNGate 出口自动切换成功")
		return
	}

	s.state.setSnapshot(domain.ConnectionSnapshot{
		State:    domain.StateError,
		Message:  "VPNGate 出口失效，缓存中的候选节点均不可用",
		Metadata: map[string]string{},
	})
	s.state.addLog("error", "VPNGate 自动切换失败，请刷新节点后重试")
}

func (s *Server) autoConnectExit(ctx context.Context) {
	s.state.addLog("info", "正在自动准备 VPNGate/OpenVPN 出口")
	if env := s.tunnel.CheckEnvironment(); !env.Ready {
		s.state.addLog("warning", "OpenVPN 出口自动连接跳过: "+env.Message)
		return
	}
	s.state.mu.RLock()
	previousNodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	src := source.VPNGateSource{APIURL: s.cfg.VPNGate.APIURL, MaxNodes: s.cfg.VPNGate.MaxNodes}
	s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateFetchingNodes, Message: "正在自动获取 VPNGate 节点", Metadata: map[string]string{}})
	nodes, err := src.Fetch(ctx)
	if err != nil {
		s.state.addLog("warning", "自动获取 VPNGate 节点失败: "+err.Error())
		s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateError, Message: "自动获取 VPNGate 节点失败: " + err.Error(), Metadata: map[string]string{}})
		return
	}
	if s.cfg.Classifier.Enabled {
		s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateProbingNodes, Message: "正在自动执行家宽判定", Metadata: map[string]string{}})
		classifier := classify.StrictClassifier{
			Timeout:        time.Duration(s.cfg.Classifier.TimeoutMS) * time.Millisecond,
			MaxConcurrency: s.cfg.Classifier.MaxConcurrency,
		}
		nodes = classifier.Classify(ctx, nodes)
	}
	s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateProbingNodes, Message: "正在自动 TCP 探活", Metadata: map[string]string{}})
	prober := probe.TCPProbe{
		Timeout:        time.Duration(s.cfg.Probe.TimeoutMS) * time.Millisecond,
		MaxConcurrency: s.cfg.Probe.MaxConcurrency,
		Samples:        s.cfg.Probe.Samples,
	}
	nodes = prober.Probe(ctx, nodes)
	nodes = preserveNodeCooldowns(nodes, previousNodes, time.Now())
	s.state.mu.Lock()
	s.state.nodes = nodes
	s.state.snapshot = domain.InitialSnapshot()
	s.state.snapshot.Message = fmt.Sprintf("自动准备完成，已缓存 %d 个节点", len(nodes))
	s.state.saveLocked()
	s.state.mu.Unlock()
	maxAttempts := s.cfg.Failover.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	candidates := connectionCandidates(nodes, "", maxAttempts, time.Now())
	if len(candidates) == 0 {
		s.state.addLog("warning", "自动连接出口失败: 没有可连接的 VPNGate 节点")
		return
	}
	for attempt, node := range candidates {
		s.state.setSnapshot(domain.ConnectionSnapshot{
			State:        connectionAttemptState(attempt),
			Message:      fmt.Sprintf("正在自动连接 VPNGate 出口 %d/%d：%s", attempt+1, len(candidates), node.IP),
			ActiveNodeID: node.ID,
			Metadata: map[string]string{
				"node_ip": node.IP,
				"attempt": fmt.Sprintf("%d/%d", attempt+1, len(candidates)),
			},
		})
		if _, err := s.connectCandidate(ctx, node); err != nil {
			s.cooldownNode(node.ID, err.Error())
			s.state.addLog("warning", fmt.Sprintf("自动连接节点 %s 失败: %s", node.IP, err))
			continue
		}
		s.state.addLog("info", "VPNGate/OpenVPN 出口已自动连接")
		return
	}
	s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateError, Message: "自动连接 VPNGate/OpenVPN 出口失败", Metadata: map[string]string{}})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Username != s.cfg.Auth.Username || req.Password != s.cfg.Auth.Password {
		s.state.addLog("warning", "登录失败")
		writeError(w, http.StatusUnauthorized, "用户名或密码不正确")
		return
	}
	sessionID, err := randomToken(32)
	if err != nil {
		s.state.addLog("error", "无法生成安全会话")
		writeError(w, http.StatusInternalServerError, "无法创建登录会话")
		return
	}
	s.state.mu.Lock()
	s.state.sessions[sessionID] = time.Now().Add(24 * time.Hour)
	s.state.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     "rim_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
	s.state.addLog("info", "控制台登录成功")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	cookie, err := r.Cookie("rim_session")
	if err == nil {
		s.state.mu.Lock()
		delete(s.state.sessions, cookie.Value)
		s.state.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "rim_session", Value: "", Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"username": s.cfg.Auth.Username})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot": s.state.snapshot,
		"counts":   s.state.countsLocked(),
	})
}

func (s *Server) nodes(w http.ResponseWriter, r *http.Request) {
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) refreshNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	previous := s.currentSnapshot()
	s.state.mu.RLock()
	previousNodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateFetchingNodes, Message: "正在获取 VPNGate 节点", Metadata: map[string]string{}})
	src := source.VPNGateSource{APIURL: s.cfg.VPNGate.APIURL, MaxNodes: s.cfg.VPNGate.MaxNodes}
	nodes, err := src.Fetch(r.Context())
	if err != nil {
		s.state.setSnapshot(nodeMaintenanceSnapshot(previous, s.tunnel.CheckEnvironment(), "节点刷新失败: "+err.Error()))
		s.state.addLog("error", "节点刷新失败: "+err.Error())
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	nodes = preserveNodeCooldowns(nodes, previousNodes, time.Now())
	s.state.mu.Lock()
	s.state.nodes = nodes
	s.state.snapshot = nodeMaintenanceSnapshot(previous, s.tunnel.CheckEnvironment(), fmt.Sprintf("已刷新 %d 个节点", len(nodes)))
	s.state.saveLocked()
	s.state.mu.Unlock()
	s.state.addLog("info", fmt.Sprintf("已刷新 %d 个 VPNGate 节点", len(nodes)))
	writeJSON(w, http.StatusOK, map[string]any{"nodes": len(nodes)})
}

func (s *Server) classifyNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	previous := s.currentSnapshot()
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	if len(nodes) == 0 {
		writeError(w, http.StatusConflict, "请先刷新节点")
		return
	}
	s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateProbingNodes, Message: "正在执行严格家宽判定", Metadata: map[string]string{}})
	classifier := classify.StrictClassifier{Timeout: time.Duration(s.cfg.Classifier.TimeoutMS) * time.Millisecond, MaxConcurrency: s.cfg.Classifier.MaxConcurrency}
	classified := classifier.Classify(r.Context(), nodes)
	s.state.mu.Lock()
	s.state.nodes = classified
	s.state.snapshot = nodeMaintenanceSnapshot(previous, s.tunnel.CheckEnvironment(), "严格家宽判定完成")
	counts := s.state.countsLocked()
	s.state.saveLocked()
	s.state.mu.Unlock()
	s.state.addLog("info", fmt.Sprintf("分类完成：严格家宽 %d，候选 %d", counts["strict_home"], counts["candidate"]))
	writeJSON(w, http.StatusOK, counts)
}

func (s *Server) probeNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	previous := s.currentSnapshot()
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	if len(nodes) == 0 {
		writeError(w, http.StatusConflict, "请先刷新节点")
		return
	}
	s.state.setSnapshot(domain.ConnectionSnapshot{State: domain.StateProbingNodes, Message: "正在 TCP 探活节点", Metadata: map[string]string{}})
	prober := probe.TCPProbe{
		Timeout:        time.Duration(s.cfg.Probe.TimeoutMS) * time.Millisecond,
		MaxConcurrency: s.cfg.Probe.MaxConcurrency,
		Samples:        s.cfg.Probe.Samples,
	}
	probed := prober.Probe(r.Context(), nodes)
	s.state.mu.Lock()
	s.state.nodes = probed
	s.state.snapshot = nodeMaintenanceSnapshot(previous, s.tunnel.CheckEnvironment(), "TCP 探活完成")
	counts := s.state.countsLocked()
	s.state.saveLocked()
	s.state.mu.Unlock()
	s.state.addLog("info", fmt.Sprintf("探活完成：可用 %d，不可用 %d", counts["available"], counts["unavailable"]))
	writeJSON(w, http.StatusOK, counts)
}

func (s *Server) environment(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"openvpn": s.tunnel.CheckEnvironment(),
		"proxy":   s.proxy.Status(),
	})
}

func (s *Server) verifyExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	node, ok := s.activeNode()
	if !ok {
		writeError(w, http.StatusConflict, "当前没有活动节点")
		return
	}
	result, err := s.verifyNodeExit(r.Context(), node)
	if err != nil {
		s.state.addLog("error", "出口复核失败: "+err.Error())
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.applyVerifyResult(node, result)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) proxyStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.proxy.Status())
}

func (s *Server) proxyStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	status, err := s.proxy.Start(context.Background())
	if err != nil {
		s.state.addLog("error", "VMESS 核心启动失败: "+err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "status": status})
		return
	}
	s.state.addLog("info", "VMESS 核心已启动")
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) proxyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	status, err := s.proxy.Stop()
	if err != nil {
		s.state.addLog("error", "VMESS 核心停止失败: "+err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error(), "status": status})
		return
	}
	s.state.addLog("info", "VMESS 核心已停止")
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req struct {
		NodeID string `json:"node_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if env := s.tunnel.CheckEnvironment(); !env.Ready {
		writeError(w, http.StatusBadGateway, env.Message)
		return
	} else if env.Running {
		snapshot := nodeMaintenanceSnapshot(s.currentSnapshot(), env, "VPNGate 出口已保持连接")
		s.state.setSnapshot(snapshot)
		writeJSON(w, http.StatusOK, snapshot)
		return
	}
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	if len(nodes) == 0 {
		writeError(w, http.StatusConflict, "请先刷新节点")
		return
	}
	maxAttempts := 1
	if req.NodeID == "" && s.cfg.Failover.Enabled {
		maxAttempts = s.cfg.Failover.MaxAttempts
	}
	candidates := connectionCandidates(nodes, req.NodeID, maxAttempts, time.Now())
	if len(candidates) == 0 {
		writeError(w, http.StatusNotFound, "没有可连接的节点")
		return
	}
	var lastErr error
	for attempt, node := range candidates {
		s.state.setSnapshot(domain.ConnectionSnapshot{
			State:        connectionAttemptState(attempt),
			Message:      fmt.Sprintf("正在连接节点 %d/%d：%s", attempt+1, len(candidates), node.IP),
			ActiveNodeID: node.ID,
			Metadata: map[string]string{
				"node_ip": fmt.Sprintf("%s", node.IP),
				"attempt": fmt.Sprintf("%d/%d", attempt+1, len(candidates)),
			},
		})
		snapshot, err := s.connectCandidate(r.Context(), node)
		if err != nil {
			lastErr = err
			s.cooldownNode(node.ID, err.Error())
			if attempt < len(candidates)-1 {
				s.state.addLog("warning", fmt.Sprintf("节点 %s 失败，自动尝试下一个：%s", node.IP, err))
			}
			continue
		}
		writeJSON(w, http.StatusOK, snapshot)
		return
	}
	message := "所有候选节点连接失败"
	if lastErr != nil {
		message += ": " + lastErr.Error()
	}
	s.state.mu.Lock()
	s.state.snapshot = domain.ConnectionSnapshot{
		State:    domain.StateError,
		Message:  message,
		Metadata: map[string]string{"attempts": fmt.Sprintf("%d", len(candidates))},
	}
	s.state.logs = appendLogLocked(s.state.logs, "error", message)
	s.state.saveLocked()
	s.state.mu.Unlock()
	writeError(w, http.StatusBadGateway, message)
}

func (s *Server) currentSnapshot() domain.ConnectionSnapshot {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	return s.state.snapshot
}

func nodeMaintenanceSnapshot(previous domain.ConnectionSnapshot, environment tunnel.EnvironmentReport, message string) domain.ConnectionSnapshot {
	if !environment.Running {
		snapshot := domain.InitialSnapshot()
		snapshot.Message = message
		return snapshot
	}
	previous.State = domain.StateConnected
	previous.Message = message
	if environment.NodeID != "" {
		previous.ActiveNodeID = environment.NodeID
	}
	if previous.Metadata == nil {
		previous.Metadata = map[string]string{}
	}
	if environment.NodeID != "" {
		previous.Metadata["node_id"] = environment.NodeID
	}
	return previous
}

func (s *Server) disconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	_ = s.tunnel.Disconnect()
	s.state.setSnapshot(domain.InitialSnapshot())
	s.state.persist()
	s.state.addLog("info", "已断开当前控制状态")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) connectCandidate(ctx context.Context, node domain.VpnNode) (domain.ConnectionSnapshot, error) {
	if err := s.tunnel.Connect(ctx, node); err != nil {
		return domain.ConnectionSnapshot{}, fmt.Errorf("OpenVPN 连接失败: %w", err)
	}
	now := time.Now()
	verifyResult := verify.Result{}
	var verifiedAt *time.Time
	verifyMessage := "OpenVPN 已连接，等待出口复核接入"
	if s.cfg.Verify.Enabled {
		s.state.setSnapshot(domain.ConnectionSnapshot{
			State:          domain.StateVerifying,
			Message:        "正在复核公网出口",
			ActiveNodeID:   node.ID,
			ConnectedSince: &now,
			Metadata:       map[string]string{"node_ip": node.IP},
		})
		var err error
		verifyResult, err = s.verifyNodeExit(ctx, node)
		if err != nil {
			_ = s.tunnel.Disconnect()
			return domain.ConnectionSnapshot{}, fmt.Errorf("出口复核失败: %w", err)
		}
		if !verifyResult.Passed {
			_ = s.tunnel.Disconnect()
			return domain.ConnectionSnapshot{}, fmt.Errorf("出口复核未通过: %s", verifyResult.Message)
		}
		verified := time.Now()
		verifiedAt = &verified
		verifyMessage = "OpenVPN 已连接，出口复核通过"
	}
	snapshot := domain.ConnectionSnapshot{
		State:          domain.StateConnected,
		Message:        verifyMessage,
		ActiveNodeID:   node.ID,
		ExitIP:         verifyResult.ExitIP,
		ConnectedSince: &now,
		LastVerifiedAt: verifiedAt,
		Metadata: map[string]string{
			"node_ip":       node.IP,
			"exit_country":  verifyResult.CountryCode,
			"exit_asn":      verifyResult.ASN,
			"verify_passed": fmt.Sprintf("%v", verifyResult.Passed),
		},
	}
	s.state.mu.Lock()
	s.state.snapshot = snapshot
	s.state.logs = appendLogLocked(s.state.logs, "info", "OpenVPN 已连接节点 "+node.ID)
	s.state.saveLocked()
	s.state.mu.Unlock()
	return snapshot, nil
}

func (s *Server) cooldownNode(nodeID string, message string) {
	cooldown := time.Duration(s.cfg.Failover.CooldownSeconds) * time.Second
	now := time.Now()
	s.state.mu.Lock()
	s.state.nodes = markNodeFailure(s.state.nodes, nodeID, message, cooldown, now)
	s.state.logs = appendLogLocked(s.state.logs, "warning", "节点进入冷却 "+nodeID+": "+message)
	s.state.saveLocked()
	s.state.mu.Unlock()
}

func connectionAttemptState(attempt int) domain.ConnectionState {
	if attempt == 0 {
		return domain.StateConnecting
	}
	return domain.StateFailingOver
}

func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	s.state.mu.RLock()
	logs := append([]LogEntry(nil), s.state.logs...)
	s.state.mu.RUnlock()
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) subscriptionPreview(w http.ResponseWriter, r *http.Request) {
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	body, err := subscription.VMESS(s.cfg.Subscription, nodes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"vmess_url":           "/sub/vmess?token=" + s.cfg.Subscription.Token,
		"quantumult_x_url":    "/sub/quantumult-x?token=" + s.cfg.Subscription.Token,
		"subscription_host":   s.cfg.Subscription.Host,
		"resolved_host":       subscription.ResolveHost(s.cfg.Subscription.Host),
		"server_listen":       s.cfg.Server.Listen,
		"subscription_listen": s.cfg.Subscription.Listen,
		"size":                len(body),
	})
}

func (s *Server) vmessSubscription(w http.ResponseWriter, r *http.Request) {
	if !s.validSubscriptionToken(r) {
		writeError(w, http.StatusUnauthorized, "invalid subscription token")
		return
	}
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	body, err := subscription.VMESS(s.cfg.Subscription, nodes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (s *Server) quantumultXSubscription(w http.ResponseWriter, r *http.Request) {
	if !s.validSubscriptionToken(r) {
		writeError(w, http.StatusUnauthorized, "invalid subscription token")
		return
	}
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	body, err := subscription.QuantumultX(s.cfg.Subscription, nodes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (s *Server) clashSubscription(w http.ResponseWriter, r *http.Request) {
	if !s.validSubscriptionToken(r) {
		writeError(w, http.StatusUnauthorized, "invalid subscription token")
		return
	}
	s.state.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	body, err := subscription.ClashYAML(s.cfg.Subscription, nodes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("rim_session")
		if err != nil || !s.sessionValid(cookie.Value) {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}
		next(w, r)
	}
}

func (s *Server) sessionValid(value string) bool {
	s.state.mu.RLock()
	expires, ok := s.state.sessions[value]
	s.state.mu.RUnlock()
	return ok && time.Now().Before(expires)
}

func (s *Server) validSubscriptionToken(r *http.Request) bool {
	return s.cfg.Subscription.Enabled && r.URL.Query().Get("token") == s.cfg.Subscription.Token
}

func (s *AppState) setSnapshot(snapshot domain.ConnectionSnapshot) {
	s.mu.Lock()
	s.snapshot = snapshot
	s.mu.Unlock()
}

func (s *AppState) persist() {
	s.mu.RLock()
	nodes := append([]domain.VpnNode(nil), s.nodes...)
	snapshot := s.snapshot
	store := s.store
	s.mu.RUnlock()
	if store != nil {
		_ = store.Save(nodes, snapshot)
	}
}

func (s *AppState) saveLocked() {
	if s.store == nil {
		return
	}
	nodes := append([]domain.VpnNode(nil), s.nodes...)
	snapshot := s.snapshot
	if err := s.store.Save(nodes, snapshot); err != nil {
		s.logs = appendLogLocked(s.logs, "warning", "本地状态保存失败: "+err.Error())
	}
}

func (s *AppState) addLog(level, message string) {
	s.mu.Lock()
	s.logs = appendLogLocked(s.logs, level, message)
	s.mu.Unlock()
}

func appendLogLocked(logs []LogEntry, level, message string) []LogEntry {
	logs = append(logs, LogEntry{At: time.Now(), Level: level, Message: message})
	if len(logs) > 300 {
		logs = logs[len(logs)-300:]
	}
	return logs
}

func (s *AppState) countsLocked() map[string]int {
	counts := map[string]int{
		"nodes":       len(s.nodes),
		"available":   0,
		"strict_home": 0,
		"candidate":   0,
		"unavailable": 0,
	}
	for _, node := range s.nodes {
		switch node.Status {
		case domain.NodeAvailable:
			counts["available"]++
		case domain.NodeUnavailable:
			counts["unavailable"]++
		}
		switch node.PurityGrade {
		case domain.PurityStrictHome:
			counts["strict_home"]++
		case domain.PurityCandidate:
			counts["candidate"]++
		}
	}
	return counts
}

func (s *Server) activeNode() (domain.VpnNode, bool) {
	s.state.mu.RLock()
	activeID := s.state.snapshot.ActiveNodeID
	nodes := append([]domain.VpnNode(nil), s.state.nodes...)
	s.state.mu.RUnlock()
	if activeID == "" {
		return domain.VpnNode{}, false
	}
	for _, node := range nodes {
		if node.ID == activeID {
			return node, true
		}
	}
	return domain.VpnNode{}, false
}

func (s *Server) verifyNodeExit(ctx context.Context, node domain.VpnNode) (verify.Result, error) {
	timeout := time.Duration(s.cfg.Verify.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	verifier := verify.ExitVerifier{
		APIURL: s.cfg.Verify.APIURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext:         s.tunnel.DialContext,
				DisableKeepAlives:   true,
				TLSHandshakeTimeout: timeout,
			},
		},
		Timeout: timeout,
	}
	return verifier.Verify(ctx, node)
}

func (s *Server) applyVerifyResult(node domain.VpnNode, result verify.Result) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	now := time.Now()
	state := domain.StateConnected
	if !result.Passed {
		state = domain.StateError
	}
	s.state.snapshot.State = state
	s.state.snapshot.ActiveNodeID = node.ID
	s.state.snapshot.ExitIP = result.ExitIP
	s.state.snapshot.LastVerifiedAt = &now
	s.state.snapshot.Message = result.Message
	if s.state.snapshot.Metadata == nil {
		s.state.snapshot.Metadata = map[string]string{}
	}
	s.state.snapshot.Metadata["node_ip"] = node.IP
	s.state.snapshot.Metadata["exit_country"] = result.CountryCode
	s.state.snapshot.Metadata["exit_asn"] = result.ASN
	s.state.snapshot.Metadata["verify_passed"] = fmt.Sprintf("%v", result.Passed)
	s.state.logs = appendLogLocked(s.state.logs, "info", "出口复核: "+result.Message)
	s.state.saveLocked()
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; connect-src 'self'; font-src 'self'; form-action 'self'; frame-ancestors 'none'; img-src 'self' data:; object-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
