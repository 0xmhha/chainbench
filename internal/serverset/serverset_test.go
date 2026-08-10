package serverset_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/serverset"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "remote-server-config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

const sample = `
user: ubuntu
port: 22
password: filepass
servers:
  - index: 1
    host: 10.0.0.1
  - index: 7
    host: 10.0.0.7
    user: root
    port: 2222
`

func TestServer_DefaultsAndOverrides(t *testing.T) {
	cfg, err := serverset.Load(write(t, sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// index 1 inherits all file-level defaults.
	s1, err := cfg.Server(1)
	if err != nil {
		t.Fatalf("Server(1): %v", err)
	}
	c1, err := s1.Credentials(func(string) string { return "" })
	if err != nil {
		t.Fatalf("creds(1): %v", err)
	}
	if c1.User != "ubuntu" || c1.Host != "10.0.0.1" || c1.Port != 22 || c1.Password != "filepass" {
		t.Fatalf("index 1 creds wrong: %+v", c1)
	}

	// index 7 overrides user and port; password still inherited.
	s7, _ := cfg.Server(7)
	c7, err := s7.Credentials(func(string) string { return "" })
	if err != nil {
		t.Fatalf("creds(7): %v", err)
	}
	if c7.User != "root" || c7.Port != 2222 || c7.Password != "filepass" {
		t.Fatalf("index 7 creds wrong: %+v", c7)
	}
}

func TestServer_EnvOverridesFile(t *testing.T) {
	cfg, err := serverset.Load(write(t, sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	s1, _ := cfg.Server(1)
	env := map[string]string{"CHAINBENCH_REMOTE_USER": "deploy", "CHAINBENCH_REMOTE_PASS": "envpass"}
	c, err := s1.Credentials(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("creds: %v", err)
	}
	if c.User != "deploy" || c.Password != "envpass" {
		t.Fatalf("env did not override file: %+v", c)
	}
}

func TestServer_NotFoundListsIndexes(t *testing.T) {
	cfg, _ := serverset.Load(write(t, sample))
	if _, err := cfg.Server(99); err == nil {
		t.Fatal("expected error for missing index")
	}
}

func TestServer_NoAuthErrors(t *testing.T) {
	cfg, err := serverset.Load(write(t, "user: ubuntu\nservers:\n  - index: 1\n    host: h\n"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	s, _ := cfg.Server(1)
	if _, err := s.Credentials(func(string) string { return "" }); err == nil {
		t.Fatal("expected no-auth error")
	}
}

func TestLoad_Errors(t *testing.T) {
	if _, err := serverset.Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected not-found error")
	}
	// unknown field is rejected (typo protection).
	if _, err := serverset.Load(write(t, "servers:\n  - index: 1\n    host: h\n    passwrd: x\n")); err == nil {
		t.Fatal("expected unknown-field error")
	}
	// duplicate index.
	if _, err := serverset.Load(write(t, "servers:\n  - index: 1\n    host: a\n  - index: 1\n    host: b\n")); err == nil {
		t.Fatal("expected duplicate-index error")
	}
	// no host.
	if _, err := serverset.Load(write(t, "servers:\n  - index: 1\n")); err == nil {
		t.Fatal("expected no-host error")
	}
	// empty inventory.
	if _, err := serverset.Load(write(t, "user: ubuntu\n")); err == nil {
		t.Fatal("expected empty-inventory error")
	}
}
