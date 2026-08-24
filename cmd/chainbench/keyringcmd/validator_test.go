package keyringcmd_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidatorRoster_WbftFamily(t *testing.T) {
	out, err := run(t, "validator", "roster", "--chain", "stablenet", "--keys", "../../../keys/preset")
	if err != nil {
		t.Fatalf("validator roster: %v\n%s", err, out)
	}
	for _, want := range []string{"family: wbft", "validator", "BLS present", "governance-member"} {
		if !strings.Contains(out, want) {
			t.Fatalf("roster missing %q:\n%s", want, out)
		}
	}
}

func TestValidatorRoster_PoaFamilyNote(t *testing.T) {
	out, err := run(t, "validator", "roster", "--chain", "wemix", "--keys", "../../../keys/preset")
	if err != nil {
		t.Fatalf("validator roster wemix: %v\n%s", err, out)
	}
	if !strings.Contains(out, "family: poa") || !strings.Contains(out, "bootstrap") {
		t.Fatalf("wemix roster should be poa + bootstrap note:\n%s", out)
	}
	if strings.Contains(out, "\nvalidator ") {
		t.Fatalf("poa roster must not list genesis validators:\n%s", out)
	}
}

func TestValidatorRoster_RequiresChain(t *testing.T) {
	if _, err := run(t, "validator", "roster", "--keys", "../../../keys/preset"); err == nil {
		t.Fatal("expected error without --chain")
	}
}

// validatorView mirrors the command's JSON output from the outside — these
// tests assert the surface, not the internal type.
type validatorView struct {
	Chain        string `json:"chain"`
	Family       string `json:"family"`
	Address      string `json:"address"`
	BLSPublicKey string `json:"blsPublicKey"`
	BLSPoP       string `json:"blsPoP"`
	Stored       string `json:"stored"`
	Note         string `json:"note"`
}

func validatorJSON(t *testing.T, args ...string) validatorView {
	t.Helper()
	out, err := run(t, args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	var v validatorView
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
	ring := newRing(t)
	v := validatorJSON(t, "validator", "import", "--chain", "wemix", "--private-key", exportKey(t, ring, "node1"), "--json")
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
