package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRemoteImport(t *testing.T) {
	cases := []struct {
		in               string
		user, host, path string
		wantErr          bool
	}{
		{"10.0.0.1:/keys/node1.json", "", "10.0.0.1", "/keys/node1.json", false},
		{"ubuntu@host:/k", "ubuntu", "host", "/k", false},
		{"nopath", "", "", "", true},
		{"host:", "", "", "", true},
		{":/path", "", "", "", true},
	}
	for _, c := range cases {
		u, h, p, err := parseRemoteImport(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: expected error", c.in)
			}
			continue
		}
		if err != nil || u != c.user || h != c.host || p != c.path {
			t.Errorf("%q -> (%q,%q,%q,%v), want (%q,%q,%q)", c.in, u, h, p, err, c.user, c.host, c.path)
		}
	}
}

func TestKeysImport_RemoteExactlyOneSource(t *testing.T) {
	// remote-import counts as a source: combining with --private-key is an error.
	if _, err := run(t, "keys", "import", "--remote-import", "h:/k", "--private-key", "0x01"); err == nil {
		t.Fatal("expected exactly-one-source error")
	}
	// malformed remote-import spec.
	if _, err := run(t, "keys", "import", "--remote-import", "bogus"); err == nil {
		t.Fatal("expected malformed remote-import error")
	}
	// well-formed but no SSH creds in env -> credential error (no dial).
	t.Setenv("CHAINBENCH_REMOTE_USER", "")
	t.Setenv("CHAINBENCH_REMOTE_PASS", "")
	t.Setenv("CHAINBENCH_REMOTE_KEY_FILE", "")
	_, err := run(t, "keys", "import", "--remote-import", "10.0.0.1:/k")
	if err == nil || !strings.Contains(err.Error()+" ", "SSH") {
		t.Fatalf("expected SSH credential error, got %v", err)
	}
}

// writeServerConfig writes a minimal inventory and returns its path.
func writeServerConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "remote-server-config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestKeysImport_ServerSource(t *testing.T) {
	cfg := writeServerConfig(t, "version: 1\ndefaults:\n  ssh: {user: ubuntu, password: pw}\nservers:\n  - index: 3\n    kind: remote\n    host: 10.0.0.3\n")

	// --server counts as a source: combining with --private-key is an error.
	if _, err := run(t, "keys", "import", "--server", "3", "--server-config", cfg, "--private-key", "0x01"); err == nil {
		t.Fatal("expected exactly-one-source error")
	}
	// --server without --remote-path errors before any dial.
	if _, err := run(t, "keys", "import", "--server", "3", "--server-config", cfg); err == nil {
		t.Fatal("expected --remote-path error")
	}
	// unknown server index errors (lists available), no dial.
	_, err := run(t, "keys", "import", "--server", "9", "--server-config", cfg, "--remote-path", "/k")
	if err == nil || !strings.Contains(err.Error(), "index 9") {
		t.Fatalf("expected missing-index error, got %v", err)
	}
	// missing config file surfaces a clear error.
	_, err = run(t, "keys", "import", "--server", "3", "--server-config", filepath.Join(t.TempDir(), "nope.yaml"), "--remote-path", "/k")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected config-not-found error, got %v", err)
	}
}
