package classify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

const ipAPIBatchURL = "http://ip-api.com/batch?fields=status,message,query,country,countryCode,regionName,city,isp,org,as,asname,proxy,hosting,mobile"
const ipAPIBatchSize = 100

var allowKeywords = []string{
	"ais fibre", "ais-fibre", "biglobe", "cable", "chubu telecommunications", "cnci",
	"comcast", "community network center", "fios", "frontier", "k-opticom", "kddi",
	"korea telecom", "kornet", "lg powercomm", "ntt", "open computer network", "optage",
	"qtnet", "residential", "rostelecom", "sk broadband", "softbank", "sony network",
	"telecom", "verizon", "xfinity", "xpeed",
}

var denyKeywords = []string{
	"backbone", "cloud", "colo", "colocation", "data center", "datacenter", "digitalocean",
	"hosting", "internet initiative japan", "leaseweb", "linode", "proxy", "server",
	"servers", "transit", "vps", "vpn", "amazon", "aws", "azure", "google cloud",
	"microsoft", "oracle", "ovh", "hetzner",
}

type StrictClassifier struct {
	HTTPClient     *http.Client
	Timeout        time.Duration
	MaxConcurrency int
}

type ipAPIRecord struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Query       string `json:"query"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	AS          string `json:"as"`
	ASName      string `json:"asname"`
	Proxy       bool   `json:"proxy"`
	Hosting     bool   `json:"hosting"`
	Mobile      bool   `json:"mobile"`
}

func (c StrictClassifier) Classify(ctx context.Context, nodes []domain.VpnNode) []domain.VpnNode {
	result := append([]domain.VpnNode(nil), nodes...)
	if len(result) == 0 {
		return result
	}
	records, err := c.fetchIPAPI(ctx, result)
	if err != nil {
		now := time.Now()
		for i := range result {
			result[i].Evidence = append(result[i].Evidence, domain.ResidentialEvidence{
				Provider:  "ip-api",
				Passed:    false,
				Summary:   "分类源不可用: " + err.Error(),
				CheckedAt: now,
			})
			if result[i].PurityGrade == "" {
				result[i].PurityGrade = domain.PurityCandidate
			}
		}
		return result
	}
	byIP := map[string]ipAPIRecord{}
	for _, record := range records {
		byIP[record.Query] = record
	}
	now := time.Now()
	for i := range result {
		record, ok := byIP[result[i].IP]
		if !ok || record.Status != "success" {
			result[i].PurityGrade = domain.PurityCandidate
			result[i].Evidence = append(result[i].Evidence, domain.ResidentialEvidence{
				Provider:  "ip-api",
				Passed:    false,
				Summary:   "未返回可用分类结果",
				CheckedAt: now,
			})
			continue
		}
		result[i].Country = valueOr(record.Country, result[i].Country)
		result[i].CountryCode = valueOr(record.CountryCode, result[i].CountryCode)
		result[i].City = valueOr(record.City, result[i].City)
		result[i].ISP = valueOr(record.ISP, result[i].ISP)
		result[i].ASN = valueOr(record.AS, result[i].ASN)
		identity := strings.ToLower(strings.Join([]string{record.ISP, record.Org, record.AS, record.ASName, result[i].ReverseDNS}, " "))
		denied := record.Proxy || record.Hosting || record.Mobile || containsAny(identity, denyKeywords)
		allowed := containsAny(identity, allowKeywords)
		switch {
		case denied:
			result[i].PurityGrade = domain.PurityRejected
			result[i].PurityScore = 0
			result[i].Evidence = append(result[i].Evidence, domain.ResidentialEvidence{
				Provider:  "ip-api",
				Passed:    false,
				Summary:   fmt.Sprintf("拒绝证据 proxy=%v hosting=%v mobile=%v identity=%s", record.Proxy, record.Hosting, record.Mobile, compact(identity)),
				CheckedAt: now,
			})
		case allowed:
			result[i].PurityGrade = domain.PurityStrictHome
			result[i].PurityScore = 90
			result[i].Evidence = append(result[i].Evidence, domain.ResidentialEvidence{
				Provider:  "ip-api",
				Passed:    true,
				Summary:   "命中家庭宽带运营商证据: " + compact(identity),
				CheckedAt: now,
			})
		default:
			result[i].PurityGrade = domain.PurityCandidate
			result[i].PurityScore = 40
			result[i].Evidence = append(result[i].Evidence, domain.ResidentialEvidence{
				Provider:  "ip-api",
				Passed:    false,
				Summary:   "无明确家庭宽带证据: " + compact(identity),
				CheckedAt: now,
			})
		}
	}
	return result
}

func (c StrictClassifier) fetchIPAPI(ctx context.Context, nodes []domain.VpnNode) ([]ipAPIRecord, error) {
	client := c.HTTPClient
	if client == nil {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	ips := make([]string, 0, len(nodes))
	seen := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if _, ok := seen[node.IP]; ok {
			continue
		}
		seen[node.IP] = struct{}{}
		ips = append(ips, node.IP)
	}
	records := make([]ipAPIRecord, 0, len(ips))
	for start := 0; start < len(ips); start += ipAPIBatchSize {
		end := start + ipAPIBatchSize
		if end > len(ips) {
			end = len(ips)
		}
		batch, err := c.fetchIPAPIBatch(ctx, client, ips[start:end])
		if err != nil {
			return nil, err
		}
		records = append(records, batch...)
	}
	return records, nil
}

func (c StrictClassifier) fetchIPAPIBatch(ctx context.Context, client *http.Client, ips []string) ([]ipAPIRecord, error) {
	body, err := json.Marshal(ips)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ipAPIBatchURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "residential-ip-manager-go/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ip-api returned HTTP %d", resp.StatusCode)
	}
	var records []ipAPIRecord
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}
	return records, nil
}

func containsAny(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}

func compact(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 180 {
		return value[:180] + "..."
	}
	return value
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
