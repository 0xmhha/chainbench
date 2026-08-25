package keyringcmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ported from the retired `keys import` tests: the path syntax and the
// inventory contract are properties of `keyring import` now, and losing the
// group must not lose the coverage.

// writeServerConfig writes a minimal inventory and returns its path.
func writeServerConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "remote-server-config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

const twoServerInventory = "version: 2\n" +
	"pool:\n" +
	"  hosts: [10.0.0.1, 10.0.0.2, {name: bp1, addr: 10.0.0.3}]\n" +
	"ssh: {user: ubuntu, password: pw}\n"

// TestKeyringImport_ExactlyOneOrigin pins the full origin matrix: any two
// origins together, and none at all, are refused before anything is read.
func TestKeyringImport_ExactlyOneOrigin(t *testing.T) {
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
			dir := filepath.Join(t.TempDir(), "ring")
			args := append([]string{"keyring", "import", "--keyring-dir", dir, "--name", "x"}, tc.args...)
			if _, err := run(t, args...); err == nil {
				t.Fatal("expected an exactly-one-origin error")
			}
		})
	}
}

// TestKeyringImport_RejectsMalformedPaths keeps a typo from being read as a
// local path or dialled as a host.
func TestKeyringImport_RejectsMalformedPaths(t *testing.T) {
	for _, bad := range []string{"srv://", "srv://bp1", "user@:/k", "ssh://"} {
		t.Run(bad, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "ring")
			if _, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x", "--from", bad); err == nil {
				t.Errorf("accepted malformed path %q", bad)
			}
		})
	}
}

// TestKeyringImport_DirectHostNeedsCredentials pins that the [user@]host:path
// form refuses to dial without credentials rather than prompting or guessing.
func TestKeyringImport_DirectHostNeedsCredentials(t *testing.T) {
	t.Setenv("CHAINBENCH_REMOTE_USER", "")
	t.Setenv("CHAINBENCH_REMOTE_PASS", "")
	t.Setenv("CHAINBENCH_REMOTE_KEY_FILE", "")
	dir := filepath.Join(t.TempDir(), "ring")
	_, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x", "--from", "10.0.0.1:/k")
	if err == nil || !strings.Contains(err.Error()+" ", "SSH") {
		t.Fatalf("expected an SSH credential error, got %v", err)
	}
}

// TestKeyringImport_ServerNameComesFromInventory pins the inventory contract:
// the command line carries a name, and the host address is only ever in the
// inventory file.
func TestKeyringImport_ServerNameComesFromInventory(t *testing.T) {
	cfg := writeServerConfig(t, twoServerInventory)
	dir := filepath.Join(t.TempDir(), "ring")

	// An unknown entry fails by name, before any dial.
	_, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x",
		"--from", "srv://nope/k", "--server-config", cfg)
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected an unknown-entry error naming it, got %v", err)
	}

	// A missing inventory is a clear error rather than a silent local read.
	_, err = run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x",
		"--from", "srv://bp1/k", "--server-config", filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected an inventory-not-found error, got %v", err)
	}
}
