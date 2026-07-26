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
