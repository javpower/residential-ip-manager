package web

import (
	"sort"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

func connectionCandidates(nodes []domain.VpnNode, requestedID string, maxAttempts int, now time.Time) []domain.VpnNode {
	if requestedID != "" {
		for _, node := range nodes {
			if node.ID == requestedID && connectable(node, now) {
				return []domain.VpnNode{node}
			}
		}
		return nil
	}
	candidates := []domain.VpnNode{}
	for _, node := range nodes {
		if connectable(node, now) {
			candidates = append(candidates, node)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := nodeRank(candidates[i])
		right := nodeRank(candidates[j])
		if left != right {
			return left > right
		}
		leftLatency := latencyRank(candidates[i])
		rightLatency := latencyRank(candidates[j])
		if leftLatency != rightLatency {
			return leftLatency < rightLatency
		}
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].ID < candidates[j].ID
	})
	if maxAttempts <= 0 || maxAttempts > len(candidates) {
		return candidates
	}
	return candidates[:maxAttempts]
}

func connectable(node domain.VpnNode, now time.Time) bool {
	if node.Protocol != "tcp" || node.PurityGrade == domain.PurityRejected {
		return false
	}
	if node.Status == domain.NodeUnavailable {
		return false
	}
	if node.CooldownUntil != nil && now.Before(*node.CooldownUntil) {
		return false
	}
	return true
}

func nodeRank(node domain.VpnNode) int {
	score := 0
	if node.PurityGrade == domain.PurityStrictHome {
		score += 1000
	}
	if node.Status == domain.NodeAvailable {
		score += 200
	}
	score += node.PurityScore
	return score
}

func latencyRank(node domain.VpnNode) int {
	if node.LatencyMS != nil && !node.LatencySuspicious {
		return *node.LatencyMS
	}
	if node.AdvertisedPingMS > 0 {
		return 100_000 + node.AdvertisedPingMS
	}
	return 1_000_000
}

func markNodeFailure(nodes []domain.VpnNode, nodeID string, message string, cooldown time.Duration, now time.Time) []domain.VpnNode {
	result := append([]domain.VpnNode(nil), nodes...)
	for i := range result {
		if result[i].ID != nodeID {
			continue
		}
		result[i].Status = domain.NodeCooldown
		result[i].LastError = message
		checkedAt := now
		result[i].LastCheckedAt = &checkedAt
		if cooldown > 0 {
			until := now.Add(cooldown)
			result[i].CooldownUntil = &until
		}
		break
	}
	return result
}

func preserveNodeCooldowns(nodes, previous []domain.VpnNode, now time.Time) []domain.VpnNode {
	byID := make(map[string]domain.VpnNode, len(previous))
	for _, node := range previous {
		if node.CooldownUntil != nil && now.Before(*node.CooldownUntil) {
			byID[node.ID] = node
		}
	}
	result := append([]domain.VpnNode(nil), nodes...)
	for i := range result {
		old, ok := byID[result[i].ID]
		if !ok {
			continue
		}
		result[i].Status = domain.NodeCooldown
		result[i].CooldownUntil = old.CooldownUntil
		result[i].LastError = old.LastError
	}
	return result
}
