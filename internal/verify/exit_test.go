package verify

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Guli-Joy/residential-ip-manager/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestExitVerifierPassesMatchingExit(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"status":"success","query":"203.0.113.10","country":"Japan","countryCode":"JP","isp":"Example ISP","as":"AS12345 Example ISP"}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	node := domain.VpnNode{CountryCode: "JP", ASN: "AS12345 Example ISP"}
	got, err := ExitVerifier{HTTPClient: client}.Verify(context.Background(), node)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Passed {
		t.Fatalf("expected verification to pass: %#v", got)
	}
}

func TestExitVerifierRejectsASNAndHostingMismatch(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"status":"success","query":"198.51.100.20","country":"United States","countryCode":"US","isp":"Cloud Host","as":"AS999 Cloud Host","hosting":true}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})}
	node := domain.VpnNode{CountryCode: "JP", ASN: "AS12345 Example ISP"}
	got, err := ExitVerifier{HTTPClient: client}.Verify(context.Background(), node)
	if err != nil {
		t.Fatal(err)
	}
	if got.Passed {
		t.Fatalf("expected verification to fail: %#v", got)
	}
	if !strings.Contains(got.Message, "ASN 不匹配") || !strings.Contains(got.Message, "国家不匹配") {
		t.Fatalf("expected mismatch reasons, got %q", got.Message)
	}
}
