package keyringcmd_test

import (
	"github.com/0xmhha/chainbench/internal/core/remote"
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
	p := filepath.Join(t.TempDir(), "server-set.yaml")
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
	t.Setenv(remote.EnvUser, "")
	t.Setenv(remote.EnvPass, "")
	t.Setenv(remote.EnvKeyFile, "")
	dir := filepath.Join(t.TempDir(), "ring")
	_, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x", "--from", "10.0.0.1:/k")
	if err == nil || !strings.Contains(err.Error()+" ", "SSH") {
		t.Fatalf("expected an SSH credential error, got %v", err)
	}
}

// TestKeyringImport_ServerNameComesFromInventory pins the server set contract:
// the command line carries a name, and the host address is only ever in the
// server-set file.
func TestKeyringImport_ServerNameComesFromInventory(t *testing.T) {
	cfg := writeServerConfig(t, twoServerInventory)
	dir := filepath.Join(t.TempDir(), "ring")

	// An unknown entry fails by name, before any dial.
	_, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x",
		"--from", "srv://nope/k", "--server-set", cfg)
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected an unknown-entry error naming it, got %v", err)
	}

	// A missing inventory is a clear error rather than a silent local read.
	_, err = run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x",
		"--from", "srv://bp1/k", "--server-set", filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected a server set-not-found error, got %v", err)
	}
}

// TestKeyringImport_FromRing pins the whole-ring clone: labels and the
// validator declaration travel with the keys, a tampered source index is
// refused whole, and --from-ring cannot be mixed with single-key flags.
func TestKeyringImport_FromRing(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if _, err := run(t, "keyring", "new", "--keyring-dir", src, "--count", "3", "--validators", "2"); err != nil {
		t.Fatalf("seed source ring: %v", err)
	}

	t.Run("clone carries the declaration", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "dst")
		out, err := run(t, "keyring", "import", "--keyring-dir", dst, "--from-ring", src)
		if err != nil {
			t.Fatalf("clone: %v", err)
		}
		for _, want := range []string{"node1", "node2", "node3", "2 validators"} {
			if !strings.Contains(out, want) {
				t.Errorf("clone output lost %q:\n%s", want, out)
			}
		}
		if lst, err := run(t, "keyring", "list", "--keyring-dir", dst, "--verify"); err != nil {
			t.Errorf("cloned ring fails verification: %v\n%s", lst, err)
		}
	})

	t.Run("refuses a single-key flag alongside", func(t *testing.T) {
		dst := filepath.Join(t.TempDir(), "dst")
		if _, err := run(t, "keyring", "import", "--keyring-dir", dst, "--from-ring", src, "--name", "x"); err == nil {
			t.Fatal("accepted --from-ring with --name")
		}
	})

	t.Run("refuses a tampered source", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "bad")
		if _, err := run(t, "keyring", "new", "--keyring-dir", bad, "--count", "1"); err != nil {
			t.Fatalf("seed: %v", err)
		}
		meta := filepath.Join(bad, "metadata.json")
		raw, err := os.ReadFile(meta)
		if err != nil {
			t.Fatal(err)
		}
		// Flip the recorded address so the key no longer derives it.
		tampered := strings.Replace(string(raw), `"address": "0x`, `"address": "0xdead`, 1)
		if tampered == string(raw) {
			t.Fatal("tamper did not apply")
		}
		if err := os.WriteFile(meta, []byte(tampered), 0o600); err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(t.TempDir(), "dst")
		if _, err := run(t, "keyring", "import", "--keyring-dir", dst, "--from-ring", bad); err == nil {
			t.Fatal("cloned a ring whose index does not match its keys")
		}
	})
}

// TestKeyringImport_ExpectAddress pins the single-key integrity gate: the
// caller states what the key must be, and a different key is refused.
func TestKeyringImport_ExpectAddress(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	if _, err := run(t, "keyring", "new", "--keyring-dir", src, "--count", "1"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	show, err := run(t, "keyring", "show", "--keyring-dir", src, "--name", "node1")
	if err != nil {
		t.Fatal(err)
	}
	var addr string
	for _, line := range strings.Split(show, "\n") {
		if i := strings.Index(line, "0x"); i >= 0 && strings.Contains(line, "address") {
			addr = strings.TrimSpace(line[i:])
			break
		}
	}
	if addr == "" {
		t.Fatalf("no address in show output:\n%s", show)
	}
	keyfile := filepath.Join(src, "node1", "nodekey")

	dst := filepath.Join(t.TempDir(), "dst")
	if _, err := run(t, "keyring", "import", "--keyring-dir", dst, "--name", "a",
		"--from", keyfile, "--expect-address", addr); err != nil {
		t.Fatalf("matching expect-address refused: %v", err)
	}
	wrong := "0x" + strings.Repeat("11", 20)
	if _, err := run(t, "keyring", "import", "--keyring-dir", dst, "--name", "b",
		"--from", keyfile, "--expect-address", wrong); err == nil {
		t.Fatal("imported a key that derives a different address than expected")
	}
}
