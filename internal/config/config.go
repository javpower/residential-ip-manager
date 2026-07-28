package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	Deployment   DeploymentConfig   `json:"deployment"`
	Server       ServerConfig       `json:"server"`
	Auth         AuthConfig         `json:"auth"`
	Data         DataConfig         `json:"data"`
	VPNGate      VPNGateConfig      `json:"vpngate"`
	Classifier   ClassifierConfig   `json:"classifier"`
	Probe        ProbeConfig        `json:"probe"`
	Failover     FailoverConfig     `json:"failover"`
	Verify       VerifyConfig       `json:"verify"`
	OpenVPN      OpenVPNConfig      `json:"openvpn"`
	Subscription SubscriptionConfig `json:"subscription"`
	ProxyCore    ProxyCoreConfig    `json:"proxy_core"`
}

type DeploymentConfig struct {
	Mode            string `json:"mode"`
	AutoStartVMESS  bool   `json:"auto_start_vmess"`
	AutoConnectExit bool   `json:"auto_connect_exit"`
}

type ServerConfig struct {
	Listen        string `json:"listen"`
	SessionSecret string `json:"session_secret"`
}

type AuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type DataConfig struct {
	Dir string `json:"dir"`
}

type VPNGateConfig struct {
	APIURL   string `json:"api_url"`
	MaxNodes int    `json:"max_nodes"`
}

type ClassifierConfig struct {
	Enabled        bool `json:"enabled"`
	StrictHomeOnly bool `json:"strict_home_only"`
	TimeoutMS      int  `json:"timeout_ms"`
	MaxConcurrency int  `json:"max_concurrency"`
}

type ProbeConfig struct {
	TimeoutMS      int `json:"timeout_ms"`
	MaxConcurrency int `json:"max_concurrency"`
	Samples        int `json:"samples"`
}

type FailoverConfig struct {
	Enabled            bool `json:"enabled"`
	MaxAttempts        int  `json:"max_attempts"`
	CooldownSeconds    int  `json:"cooldown_seconds"`
	HealthCheckSeconds int  `json:"health_check_seconds"`
	FailureThreshold   int  `json:"failure_threshold"`
}

type VerifyConfig struct {
	Enabled   bool   `json:"enabled"`
	APIURL    string `json:"api_url"`
	TimeoutMS int    `json:"timeout_ms"`
}

type OpenVPNConfig struct {
	Username         string   `json:"username"`
	Password         string   `json:"password"`
	ConnectTimeoutMS int      `json:"connect_timeout_ms"`
	MTU              int      `json:"mtu"`
	DNSServers       []string `json:"dns_servers"`
}

type SubscriptionConfig struct {
	Enabled  bool   `json:"enabled"`
	Listen   string `json:"listen"`
	Token    string `json:"token"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	UUID     string `json:"uuid"`
	AlterID  int    `json:"alter_id"`
	Security string `json:"security"`
	Network  string `json:"network"`
}

type ProxyCoreConfig struct {
	Enabled           bool   `json:"enabled"`
	Type              string `json:"type"`
	Listen            string `json:"listen"`
	LogLevel          string `json:"log_level"`
	LocalSOCKSEnabled bool   `json:"local_socks_enabled"`
	LocalSOCKSListen  string `json:"local_socks_listen"`
}

func Default() Config {
	return Config{
		Deployment: DeploymentConfig{
			Mode:            "gateway",
			AutoStartVMESS:  true,
			AutoConnectExit: true,
		},
		Server: ServerConfig{
			Listen: "127.0.0.1:8899",
		},
		Auth: AuthConfig{
			Username: "admin",
		},
		Data: DataConfig{},
		VPNGate: VPNGateConfig{
			APIURL:   "https://www.vpngate.net/api/iphone/",
			MaxNodes: 300,
		},
		Classifier: ClassifierConfig{
			Enabled:        true,
			StrictHomeOnly: true,
			TimeoutMS:      10000,
			MaxConcurrency: 50,
		},
		Probe: ProbeConfig{
			TimeoutMS:      3000,
			MaxConcurrency: 64,
			Samples:        3,
		},
		Failover: FailoverConfig{
			Enabled:            true,
			MaxAttempts:        5,
			CooldownSeconds:    300,
			HealthCheckSeconds: 20,
			FailureThreshold:   2,
		},
		Verify: VerifyConfig{
			Enabled:   true,
			APIURL:    "http://ip-api.com/json/?fields=status,message,query,country,countryCode,isp,as,asname,proxy,hosting,mobile",
			TimeoutMS: 10000,
		},
		OpenVPN: OpenVPNConfig{
			Username:         "vpn",
			Password:         "vpn",
			ConnectTimeoutMS: 45000,
			MTU:              1420,
			DNSServers:       []string{"1.1.1.1", "8.8.8.8"},
		},
		Subscription: SubscriptionConfig{
			Enabled:  true,
			Listen:   "0.0.0.0:8898",
			Host:     "auto",
			Port:     10086,
			AlterID:  0,
			Security: "auto",
			Network:  "tcp",
		},
		ProxyCore: ProxyCoreConfig{
			Enabled:           true,
			Type:              "embedded",
			Listen:            "0.0.0.0",
			LogLevel:          "warning",
			LocalSOCKSEnabled: true,
			LocalSOCKSListen:  "127.0.0.1:1080",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, cfg.Validate()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}

func Init(path string) (Config, error) {
	if path == "" {
		path = "config.json"
	}
	if _, err := os.Stat(path); err == nil {
		return Config{}, fmt.Errorf("%s already exists", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	cfg := Default()
	var err error
	if cfg.Auth.Password, err = randomHex(12); err != nil {
		return Config{}, fmt.Errorf("generate admin password: %w", err)
	}
	if cfg.Server.SessionSecret, err = randomHex(32); err != nil {
		return Config{}, fmt.Errorf("generate session secret: %w", err)
	}
	if cfg.Subscription.Token, err = randomHex(24); err != nil {
		return Config{}, fmt.Errorf("generate subscription token: %w", err)
	}
	if cfg.Subscription.UUID, err = randomUUID(); err != nil {
		return Config{}, fmt.Errorf("generate VMESS UUID: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return Config{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return Config{}, err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) Validate() error {
	switch cfg.Deployment.Mode {
	case "", "gateway":
	default:
		return errors.New("deployment.mode must be gateway")
	}
	if cfg.Server.Listen == "" {
		return errors.New("server.listen is required")
	}
	if _, _, err := net.SplitHostPort(cfg.Server.Listen); err != nil {
		return fmt.Errorf("server.listen is invalid: %w", err)
	}
	if cfg.Server.SessionSecret == "" {
		return errors.New("server.session_secret is required")
	}
	if cfg.Auth.Username == "" || cfg.Auth.Password == "" {
		return errors.New("auth.username and auth.password are required")
	}
	if cfg.VPNGate.APIURL == "" {
		return errors.New("vpngate.api_url is required")
	}
	if cfg.Classifier.TimeoutMS < 0 || cfg.Classifier.MaxConcurrency < 0 {
		return errors.New("classifier timeout and concurrency must not be negative")
	}
	if cfg.Probe.TimeoutMS < 0 || cfg.Probe.MaxConcurrency < 0 || cfg.Probe.Samples < 0 {
		return errors.New("probe timeout, concurrency, and samples must not be negative")
	}
	if cfg.Failover.MaxAttempts < 0 || cfg.Failover.CooldownSeconds < 0 || cfg.Failover.HealthCheckSeconds < 0 || cfg.Failover.FailureThreshold < 0 {
		return errors.New("failover values must not be negative")
	}
	if cfg.Verify.Enabled && cfg.Verify.APIURL == "" {
		return errors.New("verify.api_url is required when verify is enabled")
	}
	if cfg.Verify.TimeoutMS < 0 {
		return errors.New("verify.timeout_ms must not be negative")
	}
	if cfg.Deployment.Mode == "gateway" {
		if !cfg.Subscription.Enabled {
			return errors.New("subscription.enabled must be true in gateway mode")
		}
		if !cfg.ProxyCore.Enabled {
			return errors.New("proxy_core.enabled must be true in gateway mode")
		}
	}
	if cfg.OpenVPN.ConnectTimeoutMS < 0 {
		return errors.New("openvpn.connect_timeout_ms must not be negative")
	}
	if cfg.OpenVPN.MTU != 0 && (cfg.OpenVPN.MTU < 576 || cfg.OpenVPN.MTU > 9000) {
		return errors.New("openvpn.mtu must be between 576 and 9000")
	}
	for _, server := range cfg.OpenVPN.DNSServers {
		if net.ParseIP(server) == nil {
			return fmt.Errorf("openvpn.dns_servers contains invalid IP %q", server)
		}
	}
	if cfg.Subscription.Enabled {
		if cfg.Subscription.Listen != "" {
			if _, _, err := net.SplitHostPort(cfg.Subscription.Listen); err != nil {
				return fmt.Errorf("subscription.listen is invalid: %w", err)
			}
		}
		if cfg.Subscription.Token == "" {
			return errors.New("subscription.token is required when subscription is enabled")
		}
		if cfg.Subscription.Port < 1 || cfg.Subscription.Port > 65535 {
			return errors.New("subscription port is required")
		}
		if cfg.Subscription.UUID == "" {
			return errors.New("subscription.uuid is required")
		}
	}
	if cfg.ProxyCore.Enabled {
		if cfg.ProxyCore.Type == "" {
			return errors.New("proxy_core.type is required when proxy core is enabled")
		}
		if cfg.ProxyCore.Type != "embedded" {
			return errors.New("proxy_core.type must be embedded")
		}
		if cfg.ProxyCore.Listen == "" {
			return errors.New("proxy_core.listen is required when proxy core is enabled")
		}
		if net.ParseIP(cfg.ProxyCore.Listen) == nil {
			return errors.New("proxy_core.listen must be an IP address")
		}
		if cfg.ProxyCore.LocalSOCKSEnabled {
			host, _, err := net.SplitHostPort(cfg.ProxyCore.LocalSOCKSListen)
			if err != nil {
				return fmt.Errorf("proxy_core.local_socks_listen is invalid: %w", err)
			}
			ip := net.ParseIP(host)
			if ip == nil || !ip.IsLoopback() {
				return errors.New("proxy_core.local_socks_listen must use a loopback IP")
			}
		}
	}
	return nil
}

func (cfg Config) DataDir() string {
	if cfg.Data.Dir != "" {
		return cfg.Data.Dir
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "ResidentialIPManagerGo")
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "ResidentialIPManagerGo")
	}
	return filepath.Join(os.TempDir(), "residential-ip-manager-go")
}

func randomHex(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func randomUUID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buffer[0:4],
		buffer[4:6],
		buffer[6:8],
		buffer[8:10],
		buffer[10:16],
	), nil
}
