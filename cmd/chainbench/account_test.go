package main

import (
	"path/filepath"
	"testing"
)

func TestAccountNew_And_Import_ShareModel(t *testing.T) {
	// account new prints a keypair.
	gen := keyJSON(t, "account", "new", "--json")
	if gen["privateKey"] == "" || gen["address"] == "" {
		t.Fatalf("account new: %v", gen)
	}
	// account import --private-key round-trips to the same address.
	imp := keyJSON(t, "account", "import", "--private-key", gen["privateKey"], "--json")
	if imp["address"] != gen["address"] {
		t.Fatalf("account import round-trip mismatch: %s vs %s", imp["address"], gen["address"])
	}
}

func TestAccountNew_StoreKeystoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gen := keyJSON(t, "account", "new", "--out", dir, "--name", "alice", "--store", "keystore", "--password", "pw", "--json")
	if gen["stored"] != filepath.Join(dir, "alice.json") {
		t.Fatalf("stored = %q", gen["stored"])
	}
	// Import it back from the keystore file.
	imp := keyJSON(t, "account", "import", "--from", gen["stored"], "--password", "pw", "--json")
	if imp["address"] != gen["address"] {
		t.Fatalf("keystore import mismatch: %s vs %s", imp["address"], gen["address"])
	}
}

func TestAccountImport_MnemonicCoinType(t *testing.T) {
	const m = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	a := keyJSON(t, "account", "import", "--mnemonic", m, "--json")
	b := keyJSON(t, "account", "import", "--mnemonic", m, "--hd-coin-type", "8283", "--json")
	if a["address"] == b["address"] {
		t.Fatalf("coin type had no effect")
	}
}
