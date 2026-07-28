package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitGeneratesRuntimeCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	created, err := Init(path)
	if err != nil {
		t.Fatal(err)
	}
	if created.Auth.Password == "" || created.Auth.Password == Default().Auth.Password {
		t.Fatal("admin password was not randomized")
	}
	if created.Server.SessionSecret == Default().Server.SessionSecret {
		t.Fatal("session secret was not randomized")
	}
	if created.Subscription.Token == Default().Subscription.Token {
		t.Fatal("subscription token was not randomized")
	}
	if created.Subscription.UUID == Default().Subscription.UUID {
		t.Fatal("VMESS UUID was not randomized")
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Auth.Password != created.Auth.Password {
		t.Fatal("returned password does not match the stored config")
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config permissions are too broad: %o", info.Mode().Perm())
	}
}

func TestInitRefusesToOverwriteConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(path); err == nil {
		t.Fatal("expected existing config to be preserved")
	}
}

func TestDefaultConfigRequiresGeneratedSecrets(t *testing.T) {
	if err := Default().Validate(); err == nil {
		t.Fatal("default config must not be valid before secrets are generated")
	}
}
