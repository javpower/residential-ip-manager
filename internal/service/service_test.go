package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateLinuxUnit(t *testing.T) {
	dir := t.TempDir()
	if err := Generate("linux", Options{BinaryPath: "/usr/bin/rim", ConfigPath: "/etc/rim/config.json", DataDir: "/var/lib/rim", Listen: "127.0.0.1:9988"}, dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "residential-ip-manager.service"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{"/usr/bin/rim", "/etc/rim/config.json", "/var/lib/rim", "127.0.0.1:9988"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in unit: %s", want, got)
		}
	}
}

func TestGenerateWindowsScripts(t *testing.T) {
	dir := t.TempDir()
	if err := Generate("windows", Options{}, dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"install-service.ps1", "uninstall-service.ps1"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}
