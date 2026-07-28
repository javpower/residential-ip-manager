package classify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestStrictClassifierMarksHomeISPStrict(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `[{"status":"success","query":"8.8.8.8","country":"United States","countryCode":"US","isp":"Comcast Cable","org":"Comcast Cable","as":"AS7922 Comcast Cable","asname":"COMCAST"}]`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	nodes := []domain.VpnNode{{ID: "n1", IP: "8.8.8.8", PurityGrade: domain.PurityCandidate}}
	got := StrictClassifier{HTTPClient: client}.Classify(context.Background(), nodes)
	if got[0].PurityGrade != domain.PurityStrictHome {
		t.Fatalf("expected strict home, got %s", got[0].PurityGrade)
	}
	if got[0].ISP != "Comcast Cable" {
		t.Fatalf("expected ISP to be filled, got %q", got[0].ISP)
	}
}

func TestStrictClassifierRejectsHosting(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `[{"status":"success","query":"1.1.1.1","isp":"Cloud Hosting","org":"Cloud Hosting","as":"AS13335 Cloudflare","hosting":true}]`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	nodes := []domain.VpnNode{{ID: "n1", IP: "1.1.1.1", PurityGrade: domain.PurityCandidate}}
	got := StrictClassifier{HTTPClient: client}.Classify(context.Background(), nodes)
	if got[0].PurityGrade != domain.PurityRejected {
		t.Fatalf("expected rejected, got %s", got[0].PurityGrade)
	}
}

func TestStrictClassifierBatchesAndDeduplicatesRequests(t *testing.T) {
	var mu sync.Mutex
	batchSizes := []int{}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var ips []string
		if err := json.NewDecoder(req.Body).Decode(&ips); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		batchSizes = append(batchSizes, len(ips))
		mu.Unlock()
		records := make([]ipAPIRecord, 0, len(ips))
		for _, ip := range ips {
			records = append(records, ipAPIRecord{Status: "success", Query: ip, ISP: "Comcast Cable"})
		}
		body, err := json.Marshal(records)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: http.Header{}}, nil
	})}
	nodes := make([]domain.VpnNode, 0, 206)
	for i := 1; i <= 205; i++ {
		ip := "198.51." + strconv.Itoa(i/255) + "." + strconv.Itoa(i%255)
		nodes = append(nodes, domain.VpnNode{ID: strconv.Itoa(i), IP: ip, PurityGrade: domain.PurityCandidate})
	}
	nodes = append(nodes, domain.VpnNode{ID: "duplicate", IP: nodes[0].IP, PurityGrade: domain.PurityCandidate})
	got := StrictClassifier{HTTPClient: client}.Classify(context.Background(), nodes)
	if len(got) != len(nodes) {
		t.Fatalf("expected %d nodes, got %d", len(nodes), len(got))
	}
	mu.Lock()
	defer mu.Unlock()
	want := []int{100, 100, 5}
	if len(batchSizes) != len(want) {
		t.Fatalf("expected batch sizes %v, got %v", want, batchSizes)
	}
	for i := range want {
		if batchSizes[i] != want[i] {
			t.Fatalf("expected batch sizes %v, got %v", want, batchSizes)
		}
	}
}
