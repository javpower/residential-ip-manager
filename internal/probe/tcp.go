package probe

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

type TCPProbe struct {
	Timeout        time.Duration
	MaxConcurrency int
	Samples        int
}

const minimumTrustedPublicLatencyMS = 5
const batchAnomalyLatencyMS = 15

func (p TCPProbe) Probe(ctx context.Context, nodes []domain.VpnNode) []domain.VpnNode {
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	limit := p.MaxConcurrency
	if limit <= 0 {
		limit = 32
	}
	samples := p.Samples
	if samples <= 0 {
		samples = 3
	}
	result := append([]domain.VpnNode(nil), nodes...)
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i := range result {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			now := time.Now()
			result[index].LastCheckedAt = &now
			result[index].ProbeAttempts = samples
			result[index].ProbeSuccesses = 0
			result[index].LatencyMS = nil
			result[index].LatencySuspicious = false
			if !strings.EqualFold(result[index].Protocol, "tcp") {
				result[index].Status = domain.NodeUnknown
				result[index].ProbeAttempts = 0
				result[index].LastError = "UDP 节点不适用 TCP 建连探测"
				return
			}
			address := net.JoinHostPort(result[index].RemoteHost, strconv.Itoa(result[index].RemotePort))
			latencies := make([]int, 0, samples)
			var lastErr error
			for sample := 0; sample < samples; sample++ {
				start := time.Now()
				var dialer net.Dialer
				dialCtx, cancel := context.WithTimeout(ctx, timeout)
				conn, err := dialer.DialContext(dialCtx, "tcp", address)
				cancel()
				if err != nil {
					lastErr = err
					continue
				}
				_ = conn.Close()
				latencies = append(latencies, ceilMilliseconds(time.Since(start)))
			}
			result[index].ProbeSuccesses = len(latencies)
			if len(latencies) == 0 {
				result[index].Status = domain.NodeUnavailable
				if lastErr != nil {
					result[index].LastError = lastErr.Error()
				} else {
					result[index].LastError = "TCP 建连探测失败"
				}
				return
			}
			sort.Ints(latencies)
			median := latencies[len(latencies)/2]
			result[index].LatencyMS = &median
			result[index].Status = domain.NodeAvailable
			result[index].LastError = ""
			if suspiciousPublicLatency(result[index].RemoteHost, median) {
				result[index].LatencySuspicious = true
				result[index].LastError = "TCP 建连耗时异常低，疑似被本机代理或网络扩展接管"
			}
		}(i)
	}
	wg.Wait()
	markBatchLatencyAnomaly(result)
	return result
}

func ceilMilliseconds(duration time.Duration) int {
	if duration <= 0 {
		return 1
	}
	return int((duration + time.Millisecond - 1) / time.Millisecond)
}

func suspiciousPublicLatency(host string, latencyMS int) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && latencyMS < minimumTrustedPublicLatencyMS
}

func markBatchLatencyAnomaly(nodes []domain.VpnNode) {
	measured := 0
	implausiblyFast := 0
	countries := map[string]struct{}{}
	for _, node := range nodes {
		if node.LatencyMS == nil || !isPublicIP(node.RemoteHost) {
			continue
		}
		measured++
		if *node.LatencyMS < batchAnomalyLatencyMS {
			implausiblyFast++
		}
		if node.CountryCode != "" {
			countries[node.CountryCode] = struct{}{}
		}
	}
	if measured < 10 || len(countries) < 2 || implausiblyFast*100 < measured*60 {
		return
	}
	for i := range nodes {
		if nodes[i].LatencyMS == nil || !isPublicIP(nodes[i].RemoteHost) {
			continue
		}
		nodes[i].LatencySuspicious = true
		nodes[i].LastError = "多个国家的公网节点同时出现异常低耗时，探测疑似被本机代理或网络扩展接管"
	}
}

func isPublicIP(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback()
}
