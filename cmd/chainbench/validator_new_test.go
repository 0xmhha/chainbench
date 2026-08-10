package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func validatorJSON(t *testing.T, args ...string) validatorOut {
	t.Helper()
	out, err := run(t, args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	var v validatorOut
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("%v not JSON: %v\n%s", args, err, out)
	}
	return v
}

func TestValidatorNew_PoaNoBLS(t *testing.T) {
	v := validatorJSON(t, "validator", "new", "--chain", "wemix", "--json")
	if v.Family != "poa" || v.Address == "" {
		t.Fatalf("poa validator: %+v", v)
	}
	if v.BLSPublicKey != "" {
		t.Fatalf("poa validator must have no BLS: %+v", v)
	}
	if v.Note == "" {
		t.Fatalf("poa validator should note bootstrap registration")
	}
}

func TestValidatorNew_WbftNeedsBootnode(t *testing.T) {
	if _, err := run(t, "validator", "new", "--chain", "stablenet"); err == nil {
		t.Fatal("expected error: wbft validator needs --bootnode")
	}
}

func TestValidatorNew_RequiresChain(t *testing.T) {
	if _, err := run(t, "validator", "new"); err == nil {
		t.Fatal("expected error without --chain")
	}
}

func TestValidatorImport_PoaPrivateKey(t *testing.T) {
	gen := keyJSON(t, "keys", "new", "--json")
	v := validatorJSON(t, "validator", "import", "--chain", "wemix", "--private-key", gen["privateKey"], "--json")
	if v.Family != "poa" || v.Address == "" {
		t.Fatalf("import poa validator: %+v", v)
	}
}

// TestValidatorNew_WbftLiveBLS derives a real BLS key via the go-wbft bootnode.
// Gated on BOOTNODE_BIN; skips in CI (no binary).
func TestValidatorNew_WbftLiveBLS(t *testing.T) {
	bn := os.Getenv("BOOTNODE_BIN")
	if bn == "" {
		t.Skip("set BOOTNODE_BIN to the go-wbft bootnode to derive a real validator BLS key")
	}
	v := validatorJSON(t, "validator", "new", "--chain", "stablenet", "--bootnode", bn, "--json")
	if v.Family != "wbft" {
		t.Fatalf("family = %q", v.Family)
	}
	if !strings.HasPrefix(v.BLSPublicKey, "0x") || len(v.BLSPublicKey) < 90 {
		t.Fatalf("BLS public key not derived: %q", v.BLSPublicKey)
	}
	if !strings.HasPrefix(v.BLSPoP, "0x") {
		t.Fatalf("BLS PoP not derived: %q", v.BLSPoP)
	}
}
