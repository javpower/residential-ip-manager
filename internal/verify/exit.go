package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

var asnPattern = regexp.MustCompile(`(?i)\bAS\s*(\d+)\b`)

type ExitVerifier struct {
	APIURL     string
	HTTPClient *http.Client
	Timeout    time.Duration
}

type Result struct {
	Passed      bool   `json:"passed"`
	ExitIP      string `json:"exit_ip"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	ISP         string `json:"isp"`
	ASN         string `json:"asn"`
	Proxy       bool   `json:"proxy"`
	Hosting     bool   `json:"hosting"`
	Mobile      bool   `json:"mobile"`
	Message     string `json:"message"`
}

type ipAPIResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Query       string `json:"query"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	ISP         string `json:"isp"`
	AS          string `json:"as"`
	ASName      string `json:"asname"`
	Proxy       bool   `json:"proxy"`
	Hosting     bool   `json:"hosting"`
	Mobile      bool   `json:"mobile"`
}

func (v ExitVerifier) Verify(ctx context.Context, node domain.VpnNode) (Result, error) {
	apiURL := v.APIURL
	if apiURL == "" {
		apiURL = "http://ip-api.com/json/?fields=status,message,query,country,countryCode,isp,as,asname,proxy,hosting,mobile"
	}
	client := v.HTTPClient
	if client == nil {
		timeout := v.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "residential-ip-manager-go/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("exit verifier returned HTTP %d", resp.StatusCode)
	}
	var payload ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Result{}, err
	}
	if payload.Status != "success" {
		return Result{}, fmt.Errorf("exit verifier failed: %s", payload.Message)
	}
	result := Result{
		ExitIP:      payload.Query,
		Country:     payload.Country,
		CountryCode: strings.ToUpper(payload.CountryCode),
		ISP:         payload.ISP,
		ASN:         payload.AS,
		Proxy:       payload.Proxy,
		Hosting:     payload.Hosting,
		Mobile:      payload.Mobile,
	}
	result.Passed, result.Message = compare(node, result)
	return result, nil
}

func compare(node domain.VpnNode, result Result) (bool, string) {
	reasons := []string{}
	if result.Proxy || result.Hosting || result.Mobile {
		reasons = append(reasons, fmt.Sprintf("出口情报包含代理/托管/移动标记 proxy=%v hosting=%v mobile=%v", result.Proxy, result.Hosting, result.Mobile))
	}
	if node.CountryCode != "" && result.CountryCode != "" && !strings.EqualFold(node.CountryCode, result.CountryCode) {
		reasons = append(reasons, fmt.Sprintf("国家不匹配：节点 %s，出口 %s", node.CountryCode, result.CountryCode))
	}
	nodeASN := asnNumber(node.ASN)
	exitASN := asnNumber(result.ASN)
	if nodeASN != "" && exitASN != "" && nodeASN != exitASN {
		reasons = append(reasons, fmt.Sprintf("ASN 不匹配：节点 AS%s，出口 AS%s", nodeASN, exitASN))
	}
	if len(reasons) > 0 {
		return false, strings.Join(reasons, "；")
	}
	return true, "出口 IP、国家和 ASN 复核通过"
}

func asnNumber(value string) string {
	match := asnPattern.FindStringSubmatch(value)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}
