package genesis_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/genesis"
)

func TestSetAndExtractConfigSection(t *testing.T) {
	base := []byte(`{"config":{"chainId":8285,"petersburgBlock":0,"croissantBlock":20},"alloc":{}}`)
	// section content is arbitrary data supplied by the caller (no hardcoding).
	section := json.RawMessage(`{"wBFT":{"epochLength":100},"init":{"validators":["0xaaa"]}}`)

	out, err := genesis.SetConfigSection(base, "croissant", section)
	if err != nil {
		t.Fatal(err)
	}
	// merged, and unrelated fields preserved
	if !strings.Contains(string(out), `"croissant"`) || !strings.Contains(string(out), `"chainId":8285`) {
		t.Errorf("merge lost data:\n%s", out)
	}

	got, err := genesis.ExtractConfigSection(out, "croissant")
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		Init struct{ Validators []string } `json:"init"`
	}
	if err := json.Unmarshal(got, &g); err != nil || len(g.Init.Validators) != 1 {
		t.Errorf("extracted section wrong: %s (%v)", got, err)
	}
	if s, _ := genesis.ExtractConfigSection(out, "nope"); s != nil {
		t.Errorf("missing key should be nil, got %s", s)
	}
}

func TestSetConfigSection_Rejects(t *testing.T) {
	base := []byte(`{"config":{}}`)
	if _, err := genesis.SetConfigSection(base, "", json.RawMessage(`{}`)); err == nil {
		t.Error("empty key should error")
	}
	if _, err := genesis.SetConfigSection(base, "x", json.RawMessage(`not json`)); err == nil {
		t.Error("invalid section JSON should error")
	}
}

func TestApplyConfigOverrides(t *testing.T) {
	base := []byte(`{"config":{"petersburgBlock":0,"londonBlock":0},"alloc":{}}`)

	// Empty overrides return the input unchanged (same bytes).
	if same, err := genesis.ApplyConfigOverrides(base, nil); err != nil || string(same) != string(base) {
		t.Fatalf("empty overrides changed genesis: %s (%v)", same, err)
	}

	out, err := genesis.ApplyConfigOverrides(base, map[string]string{"bohoBlock": "10", "engineVer": "v2"})
	if err != nil {
		t.Fatalf("ApplyConfigOverrides: %v", err)
	}
	var g struct {
		Config map[string]json.RawMessage `json:"config"`
		Alloc  json.RawMessage            `json:"alloc"`
	}
	if err := json.Unmarshal(out, &g); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	// A numeric-looking value stays a JSON number; a bare string is quoted.
	if got := string(g.Config["bohoBlock"]); got != "10" {
		t.Errorf("bohoBlock = %s, want numeric 10", got)
	}
	if got := string(g.Config["engineVer"]); got != `"v2"` {
		t.Errorf("engineVer = %s, want quoted string \"v2\"", got)
	}
	// Untouched keys and sibling objects survive.
	if _, ok := g.Config["petersburgBlock"]; !ok {
		t.Error("petersburgBlock dropped by override")
	}
	if string(g.Alloc) != "{}" {
		t.Errorf("alloc changed: %s", g.Alloc)
	}

	// Deterministic: same overrides produce identical bytes.
	out2, _ := genesis.ApplyConfigOverrides(base, map[string]string{"engineVer": "v2", "bohoBlock": "10"})
	if string(out) != string(out2) {
		t.Errorf("ApplyConfigOverrides not deterministic:\n%s\n%s", out, out2)
	}
}

func TestMergeOverride(t *testing.T) {
	base := []byte(`{"config":{"petersburgBlock":0,"anzeon":{"systemContracts":{"govCouncil":{"params":{"quorum":"2"}}}}},"alloc":{"0xaaa":{"balance":"0x1"}}}`)

	// Empty overlay is a no-op.
	if out, err := genesis.MergeOverride(base, []byte("  ")); err != nil || string(out) != string(base) {
		t.Fatalf("empty overlay changed genesis: %s (%v)", out, err)
	}

	overlay := []byte(`{
		"alloc": {"0xbbb": {"balance":"0x2","extra":"0x4000000000000000"}},
		"config": {"anzeon": {"systemContracts": {"govCouncil": {"params": {"authorizedAddresses": ["0xbbb"]}}}}}
	}`)
	out, err := genesis.MergeOverride(base, overlay)
	if err != nil {
		t.Fatalf("MergeOverride: %v", err)
	}
	var g struct {
		Alloc  map[string]map[string]string `json:"alloc"`
		Config struct {
			Petersburg json.RawMessage `json:"petersburgBlock"`
			Anzeon     struct {
				SystemContracts struct {
					GovCouncil struct {
						Params map[string]json.RawMessage `json:"params"`
					} `json:"govCouncil"`
				} `json:"systemContracts"`
			} `json:"anzeon"`
		} `json:"config"`
	}
	if err := json.Unmarshal(out, &g); err != nil {
		t.Fatalf("parse merged: %v", err)
	}
	// New alloc entry added; existing one preserved (deep merge, not replace).
	if g.Alloc["0xbbb"]["extra"] != "0x4000000000000000" || g.Alloc["0xaaa"]["balance"] != "0x1" {
		t.Errorf("alloc merge wrong: %+v", g.Alloc)
	}
	// New nested param added; sibling param (quorum) and sibling fork survive.
	params := g.Config.Anzeon.SystemContracts.GovCouncil.Params
	if string(params["authorizedAddresses"]) != `["0xbbb"]` || string(params["quorum"]) != `"2"` {
		t.Errorf("nested param merge wrong: %v", params)
	}
	if len(g.Config.Petersburg) == 0 {
		t.Error("petersburgBlock lost in merge")
	}

	// A non-object overlay value replaces (array/scalar), not merges.
	out2, err := genesis.MergeOverride([]byte(`{"alloc":{"0xaaa":{"balance":"0x1"}}}`), []byte(`{"alloc":{"0xaaa":"0x9"}}`))
	if err != nil {
		t.Fatalf("MergeOverride replace: %v", err)
	}
	if !strings.Contains(string(out2), `"0xaaa":"0x9"`) {
		t.Errorf("scalar overlay should replace object value: %s", out2)
	}
}

func TestValidateForks(t *testing.T) {
	if err := genesis.ValidateForks([]byte(`{"config":{"croissantBlock":0,"croissant":{}}}`)); err == nil {
		t.Error("missing petersburg should error")
	}
	if err := genesis.ValidateForks([]byte(`{"config":{"petersburgBlock":0,"croissantBlock":20}}`)); err == nil {
		t.Error("croissantBlock without section should error")
	}
	if err := genesis.ValidateForks([]byte(`{"config":{"petersburgBlock":0,"croissant":{}}}`)); err == nil {
		t.Error("croissant section without block should error")
	}
	if err := genesis.ValidateForks([]byte(`{"config":{"petersburgBlock":0}}`)); err != nil {
		t.Errorf("pure wemix genesis should pass: %v", err)
	}
}
