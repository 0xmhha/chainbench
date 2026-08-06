package wbft

import (
	"encoding/json"
	"strings"
	"testing"
)

const tmpl = `{
  "config": { "chainId": __CHAIN_ID__, "anzeon": {
    "init": { "validators": "__VALIDATORS_JSON__", "blsPublicKeys": "__BLS_PUBLIC_KEYS_JSON__" }
  } },
  "extraData": "__EXTRA_DATA__"
}`

func TestBuildGenesis_SubstitutesAndValidates(t *testing.T) {
	out, err := BuildGenesis([]byte(tmpl), GenesisParams{
		ChainID:    8283,
		Validators: []string{"0xaaa", "0xbbb"},
		BLSKeys:    []string{"0x111", "0x222"},
		ExtraData:  "0xdeadbeef",
	})
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}

	var g struct {
		Config struct {
			ChainID int64 `json:"chainId"`
			Anzeon  struct {
				Init struct {
					Validators []string `json:"validators"`
					BLSKeys    []string `json:"blsPublicKeys"`
				} `json:"init"`
			} `json:"anzeon"`
		} `json:"config"`
		ExtraData string `json:"extraData"`
	}
	if err := json.Unmarshal(out, &g); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	if g.Config.ChainID != 8283 {
		t.Errorf("chainId: %d", g.Config.ChainID)
	}
	if len(g.Config.Anzeon.Init.Validators) != 2 || g.Config.Anzeon.Init.Validators[0] != "0xaaa" {
		t.Errorf("validators: %v", g.Config.Anzeon.Init.Validators)
	}
	if len(g.Config.Anzeon.Init.BLSKeys) != 2 || g.Config.Anzeon.Init.BLSKeys[1] != "0x222" {
		t.Errorf("bls keys: %v", g.Config.Anzeon.Init.BLSKeys)
	}
	if g.ExtraData != "0xdeadbeef" {
		t.Errorf("extraData: %s", g.ExtraData)
	}
	// No placeholders should remain.
	if strings.Contains(string(out), "__") {
		t.Errorf("unsubstituted placeholder remains: %s", out)
	}
}

func TestBuildGenesis_Validation(t *testing.T) {
	if _, err := BuildGenesis([]byte(tmpl), GenesisParams{ChainID: 0}); err == nil {
		t.Error("expected error for chainId 0")
	}
	if _, err := BuildGenesis([]byte(tmpl), GenesisParams{
		ChainID: 1, Validators: []string{"0xa"}, BLSKeys: []string{},
	}); err == nil {
		t.Error("expected error for validator/bls length mismatch")
	}
}
