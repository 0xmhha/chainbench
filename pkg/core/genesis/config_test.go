package genesis_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/genesis"
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
