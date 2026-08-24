package main

import (
	"encoding/json"
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

// TestValidatorNew_WbftDerivesBLSWithoutABinary guards the in-process derivation at the CLI. A
// wbft validator used to be refused without --bootnode; the material is now
// derived in process, so the command must succeed with nothing on PATH.
func TestValidatorNew_WbftDerivesBLSWithoutABinary(t *testing.T) {
	t.Setenv("PATH", "")
	v := validatorJSON(t, "validator", "new", "--chain", "stablenet", "--json")
	if v.Family != "wbft" {
		t.Fatalf("family = %q", v.Family)
	}
	if !strings.HasPrefix(v.BLSPublicKey, "0x") || len(v.BLSPublicKey) != blsPubKeyHexLen {
		t.Errorf("BLS public key not derived: %q", v.BLSPublicKey)
	}
	if !strings.HasPrefix(v.BLSPoP, "0x") || len(v.BLSPoP) != blsPoPHexLen {
		t.Errorf("BLS PoP not derived: %q", v.BLSPoP)
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

// BLS material is fixed-width: a compressed G1 point and a compressed G2
// signature, both 0x-prefixed.
const (
	blsPubKeyHexLen = 2 + 48*2
	blsPoPHexLen    = 2 + 96*2
)
