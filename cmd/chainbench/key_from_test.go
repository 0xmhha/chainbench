package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeServerConfig writes a minimal inventory and returns its path.
func writeServerConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "remote-server-config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

const twoServerInventory = "version: 1\n" +
	"defaults:\n  ssh: {user: ubuntu, password: pw}\n" +
	"servers:\n" +
	"  - index: 3\n    name: bp1\n    kind: remote\n    host: 10.0.0.3\n"

func TestKeyFrom_ExactlyOneOrigin(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"from plus private key", []string{"--from", "/k", "--private-key", "0x01"}},
		{"from plus mnemonic", []string{"--from", "/k", "--mnemonic", "a b c"}},
		{"no origin at all", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := run(t, append([]string{"keys", "import"}, tc.args...)...); err == nil {
				t.Fatal("expected an exactly-one-origin error")
			}
		})
	}
}

// TestKeyFrom_RejectsMalformedPaths keeps a typo from being read as a local
// path: "bogus" naming a file that does not exist must fail, not be dialled.
func TestKeyFrom_RejectsMalformedPaths(t *testing.T) {
	for _, bad := range []string{"srv://", "srv://bp1", "user@:/k", "ssh://"} {
		t.Run(bad, func(t *testing.T) {
			if _, err := run(t, "keys", "import", "--from", bad); err == nil {
				t.Errorf("accepted malformed path %q", bad)
			}
		})
	}
}

// TestKeyFrom_DirectHostNeedsCredentials checks the [user@]host:path form still
// works and still refuses to dial without credentials.
func TestKeyFrom_DirectHostNeedsCredentials(t *testing.T) {
	t.Setenv("CHAINBENCH_REMOTE_USER", "")
	t.Setenv("CHAINBENCH_REMOTE_PASS", "")
	t.Setenv("CHAINBENCH_REMOTE_KEY_FILE", "")
	_, err := run(t, "keys", "import", "--from", "10.0.0.1:/k")
	if err == nil || !strings.Contains(err.Error()+" ", "SSH") {
		t.Fatalf("expected an SSH credential error, got %v", err)
	}
}

// TestKeyFrom_ServerNameComesFromInventory is the K7 gate: the command line
// carries a name, and the host address is only ever in the inventory file.
func TestKeyFrom_ServerNameComesFromInventory(t *testing.T) {
	cfg := writeServerConfig(t, twoServerInventory)

	// An unknown entry fails by name, before any dial.
	_, err := run(t, "keys", "import", "--from", "srv://nope/k", "--server-config", cfg)
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected an unknown-entry error naming it, got %v", err)
	}

	// A missing inventory is a clear error rather than a silent local read.
	_, err = run(t, "keys", "import",
		"--from", "srv://bp1/k", "--server-config", filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected an inventory-not-found error, got %v", err)
	}

	// A known entry resolves its credentials from the file and gets as far as
	// dialling — which fails here, since 10.0.0.3 is not listening.
	_, err = run(t, "keys", "import", "--from", "srv://bp1/k", "--server-config", cfg)
	if err == nil {
		t.Fatal("expected a dial failure against the inventory host")
	}
	if strings.Contains(err.Error(), "inventory") {
		t.Fatalf("entry was not resolved: %v", err)
	}
}

// TestKeyFrom_SupersededFlagsStillWork keeps existing scripts running: the four
// old spellings must reach the same code as --from.
func TestKeyFrom_SupersededFlagsStillWork(t *testing.T) {
	cfg := writeServerConfig(t, twoServerInventory)

	// --server + --remote-path is translated into srv://<name>/<path>, so an
	// unknown index still fails on the index rather than on a path.
	_, err := run(t, "keys", "import", "--server", "9", "--server-config", cfg, "--remote-path", "/k")
	if err == nil || !strings.Contains(err.Error(), "index 9") {
		t.Fatalf("expected a missing-index error, got %v", err)
	}

	// --server without --remote-path errors before any dial.
	if _, err := run(t, "keys", "import", "--server", "3", "--server-config", cfg); err == nil {
		t.Fatal("expected a --remote-path error")
	}

	// --remote-import still counts as an origin.
	if _, err := run(t, "keys", "import", "--remote-import", "h:/k", "--private-key", "0x01"); err == nil {
		t.Fatal("expected an exactly-one-origin error")
	}
}
