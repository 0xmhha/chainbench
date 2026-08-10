package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func keyJSON(t *testing.T, args ...string) map[string]string {
	t.Helper()
	out, err := run(t, args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("%v output not JSON: %v\n%s", args, err, out)
	}
	return m
}

func TestKeysNew_StoreKeystore(t *testing.T) {
	dir := t.TempDir()
	m := keyJSON(t, "keys", "new", "--out", dir, "--name", "alice", "--store", "keystore", "--password", "pw", "--json")
	stored := m["stored"]
	if stored != filepath.Join(dir, "alice.json") {
		t.Fatalf("stored = %q", stored)
	}
	data, err := os.ReadFile(stored)
	if err != nil || len(data) == 0 || data[0] != '{' {
		t.Fatalf("keystore file not written as JSON: %v", err)
	}
}

func TestKeysNew_StoreKeystoreNeedsPassword(t *testing.T) {
	if _, err := run(t, "keys", "new", "--out", t.TempDir(), "--store", "keystore"); err == nil {
		t.Fatal("expected error: keystore storage needs a password")
	}
}

func TestKeysImport_PrivateKeyRoundTrip(t *testing.T) {
	gen := keyJSON(t, "keys", "new", "--json")
	imp := keyJSON(t, "keys", "import", "--private-key", gen["privateKey"], "--json")
	if imp["publicKey"] != gen["publicKey"] {
		t.Fatalf("round-trip mismatch: %s vs %s", imp["publicKey"], gen["publicKey"])
	}
}

func TestKeysImport_MnemonicCoinType(t *testing.T) {
	const m = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	eth := keyJSON(t, "keys", "import", "--mnemonic", m, "--json")
	chain := keyJSON(t, "keys", "import", "--mnemonic", m, "--hd-coin-type", "8283", "--json")
	if eth["publicKey"] == chain["publicKey"] {
		t.Fatalf("coin type had no effect: both %s", eth["publicKey"])
	}
}

func TestKeysImport_FileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gen := keyJSON(t, "keys", "new", "--out", dir, "--name", "k", "--store", "file", "--json")
	imp := keyJSON(t, "keys", "import", "--import", filepath.Join(dir, "k.key"), "--json")
	if imp["publicKey"] != gen["publicKey"] {
		t.Fatalf("file import mismatch: %s vs %s", imp["publicKey"], gen["publicKey"])
	}
}

func TestKeysImport_RequiresExactlyOneSource(t *testing.T) {
	if _, err := run(t, "keys", "import"); err == nil {
		t.Fatal("expected error with no source")
	}
	if _, err := run(t, "keys", "import", "--private-key", "0x01", "--mnemonic", "x"); err == nil {
		t.Fatal("expected error with two sources")
	}
	if !strings.Contains(mustErr(t, "keys", "import"), "exactly one") {
		t.Fatal("error should name the constraint")
	}
}

func mustErr(t *testing.T, args ...string) string {
	t.Helper()
	out, err := run(t, args...)
	if err == nil {
		t.Fatalf("%v expected error", args)
	}
	return err.Error() + out
}
