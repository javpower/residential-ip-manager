package tunnel

import (
	"strings"
	"testing"
)

func TestSanitizeOpenVPNConfigRemovesExecutableHooks(t *testing.T) {
	input := "client\nremote 1.2.3.4 443 tcp\nscript-security 2\nup evil.sh\nplugin evil.dll\n"
	got, err := SanitizeOpenVPNConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"script-security", "up evil.sh", "plugin evil.dll"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("sanitized config still contains %q: %s", forbidden, got)
		}
	}
	if !strings.Contains(got, "remote 1.2.3.4 443 tcp") {
		t.Fatalf("safe remote line was removed: %s", got)
	}
}

func TestRenderOpenVPNConfigAddsWindowsDNSProtection(t *testing.T) {
	got, err := RenderOpenVPNConfig("client\nremote 8.8.8.8 443 tcp\n", RenderOpenVPNOptions{
		AuthPath:  "C:/rim/auth.txt",
		UseProxy:  true,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7890,
		BypassIPs: []string{"8.8.8.8"},
		GOOS:      "windows",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"socks-proxy 127.0.0.1 7890", "block-outside-dns", "route 8.8.8.8 255.255.255.255 net_gateway"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered config missing %q: %s", want, got)
		}
	}
}

func TestRenderOpenVPNConfigDoesNotRequireProxy(t *testing.T) {
	got, err := RenderOpenVPNConfig("client\nremote 8.8.8.8 443 tcp\n", RenderOpenVPNOptions{
		AuthPath:  "/tmp/rim/auth.txt",
		BypassIPs: []string{"8.8.8.8"},
		GOOS:      "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "socks-proxy") {
		t.Fatalf("rendered config should not contain socks proxy by default: %s", got)
	}
	if !strings.Contains(got, "auth-user-pass") {
		t.Fatalf("rendered config missing managed auth path: %s", got)
	}
}
