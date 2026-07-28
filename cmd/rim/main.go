package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/config"
	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
	"github.com/Guli-Joy/residential-ip-manager/internal/probe"
	"github.com/Guli-Joy/residential-ip-manager/internal/service"
	"github.com/Guli-Joy/residential-ip-manager/internal/source"
	"github.com/Guli-Joy/residential-ip-manager/internal/tunnel"
	"github.com/Guli-Joy/residential-ip-manager/internal/web"
)

var version = "0.1.0-go-alpha"

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		printUsage()
		return nil
	}
	switch args[1] {
	case "serve":
		flags := flag.NewFlagSet("serve", flag.ExitOnError)
		configPath := flags.String("config", "config.json", "path to config file")
		listen := flags.String("listen", "", "override listen address")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		if strings.TrimSpace(*listen) != "" {
			cfg.Server.Listen = *listen
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		return web.NewServer(cfg, logger).ListenAndServe(ctx)
	case "config":
		if len(args) >= 3 && args[2] == "init" {
			flags := flag.NewFlagSet("config init", flag.ExitOnError)
			output := flags.String("output", "config.json", "output config file")
			if err := flags.Parse(args[3:]); err != nil {
				return err
			}
			cfg, err := config.Init(*output)
			if err != nil {
				return err
			}
			fmt.Printf("created %s\n", *output)
			fmt.Printf("web username: %s\n", cfg.Auth.Username)
			fmt.Printf("web password: %s\n", cfg.Auth.Password)
			fmt.Println("store this password securely; it will not be shown again")
			return nil
		}
		printUsage()
		return nil
	case "service":
		if len(args) >= 3 && args[2] == "generate" {
			flags := flag.NewFlagSet("service generate", flag.ExitOnError)
			platform := flags.String("platform", runtimeGOOS(), "target platform: linux, darwin, or windows")
			output := flags.String("output", "service", "output directory")
			binary := flags.String("binary", service.DefaultBinaryPath(), "path to rim binary")
			configPath := flags.String("config", "config.json", "path to config file")
			dataDir := flags.String("data-dir", "", "runtime data directory")
			listen := flags.String("listen", "127.0.0.1:8899", "listen address")
			if err := flags.Parse(args[3:]); err != nil {
				return err
			}
			if err := service.Generate(*platform, service.Options{
				BinaryPath: *binary,
				ConfigPath: *configPath,
				DataDir:    *dataDir,
				Listen:     *listen,
				Version:    version,
			}, *output); err != nil {
				return err
			}
			fmt.Printf("generated service files in %s\n", *output)
			return nil
		}
		printUsage()
		return nil
	case "diagnose":
		if len(args) >= 3 && args[2] == "vpngate" {
			return diagnoseVPNGate(args[3:])
		}
		printUsage()
		return nil
	case "version":
		fmt.Println(version)
		return nil
	default:
		printUsage()
		return nil
	}
}

func printUsage() {
	fmt.Println(`Residential IP Manager Go

Usage:
  rim serve --config config.json [--listen 127.0.0.1:8899]
  rim config init --output config.json
  rim diagnose vpngate [--config config.json]
  rim service generate --platform linux --output service
  rim version`)
}

func diagnoseVPNGate(args []string) error {
	flags := flag.NewFlagSet("diagnose vpngate", flag.ContinueOnError)
	configPath := flags.String("config", "", "optional path to config file")
	maxNodes := flags.Int("max-nodes", 50, "maximum number of VPNGate nodes to inspect")
	attempts := flags.Int("attempts", 5, "maximum connection attempts")
	checkURL := flags.String("check-url", "https://api.ipify.org", "public IP check URL")
	timeout := flags.Duration("timeout", 4*time.Minute, "overall diagnostic timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *maxNodes < 1 || *attempts < 1 || *timeout <= 0 {
		return fmt.Errorf("max-nodes, attempts, and timeout must be positive")
	}

	cfg := config.Default()
	var err error
	if strings.TrimSpace(*configPath) != "" {
		cfg, err = config.Load(*configPath)
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	fmt.Println("[1/4] fetching VPNGate nodes")
	nodes, err := (source.VPNGateSource{APIURL: cfg.VPNGate.APIURL, MaxNodes: *maxNodes}).Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch VPNGate nodes: %w", err)
	}
	fmt.Printf("[2/4] probing %d TCP profiles\n", len(nodes))
	nodes = (probe.TCPProbe{
		Timeout:        time.Duration(cfg.Probe.TimeoutMS) * time.Millisecond,
		MaxConcurrency: cfg.Probe.MaxConcurrency,
	}).Probe(ctx, nodes)
	candidates := diagnosticCandidates(nodes)
	if len(candidates) == 0 {
		return fmt.Errorf("no reachable TCP VPNGate profiles found")
	}
	if len(candidates) > *attempts {
		candidates = candidates[:*attempts]
	}

	controller := tunnel.NewOpenVPNController(cfg.OpenVPN, cfg.DataDir())
	defer controller.Disconnect()
	var failures []string
	for index, node := range candidates {
		fmt.Printf("[3/4] connecting %d/%d to %s:%d (%s)\n", index+1, len(candidates), node.IP, node.RemotePort, node.CountryCode)
		if err := controller.Connect(ctx, node); err != nil {
			fmt.Printf("      handshake failed: %v\n", err)
			failures = append(failures, fmt.Sprintf("%s: %v", node.ID, err))
			continue
		}
		exitIP, err := checkExitIP(ctx, controller, *checkURL)
		if err != nil {
			environment := controller.CheckEnvironment()
			fmt.Printf("      data path failed: %v (packets out/in=%d/%d, bytes out/in=%d/%d)\n",
				err, environment.PacketsOut, environment.PacketsIn, environment.BytesOut, environment.BytesIn)
			failures = append(failures, fmt.Sprintf("%s data path: %v", node.ID, err))
			_ = controller.Disconnect()
			continue
		}
		environment := controller.CheckEnvironment()
		fmt.Println("[4/4] userspace VPN data path passed")
		fmt.Printf("node=%s country=%s tunnel_ip=%s gateway=%s exit_ip=%s\n", node.ID, node.CountryCode, environment.LocalIP, environment.Gateway, exitIP)
		return nil
	}
	return fmt.Errorf("all VPNGate attempts failed: %s", strings.Join(failures, " | "))
}

func diagnosticCandidates(nodes []domain.VpnNode) []domain.VpnNode {
	candidates := make([]domain.VpnNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Protocol == "tcp" && node.Status == domain.NodeAvailable {
			candidates = append(candidates, node)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.LatencyMS != nil && right.LatencyMS != nil && *left.LatencyMS != *right.LatencyMS {
			return *left.LatencyMS < *right.LatencyMS
		}
		return left.Score > right.Score
	})
	return candidates
}

func checkExitIP(ctx context.Context, dialer *tunnel.OpenVPNController, url string) (string, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			DialContext:         dialer.DialContext,
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: 15 * time.Second,
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("IP check returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("IP check returned invalid address/body %q", ip)
	}
	return ip, nil
}

func runtimeGOOS() string {
	if value := os.Getenv("GOOS"); value != "" {
		return value
	}
	return runtime.GOOS
}
