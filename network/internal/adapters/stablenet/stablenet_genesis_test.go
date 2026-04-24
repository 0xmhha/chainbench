package stablenet_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
	"github.com/0xmhha/chainbench/network/internal/adapters/stablenet"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return data
}

func loadTemplate(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "templates", "genesis.template.json"))
	if err != nil {
		t.Fatalf("load genesis template: %v", err)
	}
	return data
}

func decodeGenesisInput(t *testing.T) spec.GenesisInput {
	t.Helper()
	var profile spec.Profile
	if err := json.Unmarshal(loadFixture(t, "profile.json"), &profile); err != nil {
		t.Fatalf("profile decode: %v", err)
	}
	var metadata spec.Metadata
	if err := json.Unmarshal(loadFixture(t, "metadata.json"), &metadata); err != nil {
		t.Fatalf("metadata decode: %v", err)
	}
	return spec.GenesisInput{
		Profile:       profile,
		Metadata:      metadata,
		TemplateBytes: loadTemplate(t),
		NumValidators: 2,
		BaseP2P:       30303,
	}
}

func TestGenerateGenesis_Happy(t *testing.T) {
	in := decodeGenesisInput(t)
	in.OutputPath = filepath.Join(t.TempDir(), "sub", "genesis.json")

	if err := stablenet.New().GenerateGenesis(context.Background(), in); err != nil {
		t.Fatalf("GenerateGenesis: %v", err)
	}

	raw, err := os.ReadFile(in.OutputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	cfg, ok := got["config"].(map[string]any)
	if !ok {
		t.Fatalf("config missing / wrong type: %T", got["config"])
	}
	if cid, _ := cfg["chainId"].(float64); cid != 8283 {
		t.Errorf("chainId = %v, want 8283", cfg["chainId"])
	}

	anzeon, ok := cfg["anzeon"].(map[string]any)
	if !ok {
		t.Fatalf("config.anzeon missing: %T", cfg["anzeon"])
	}
	wbft, ok := anzeon["wbft"].(map[string]any)
	if !ok {
		t.Fatalf("config.anzeon.wbft missing")
	}
	if v, _ := wbft["requestTimeoutSeconds"].(float64); v != 2 {
		t.Errorf("requestTimeoutSeconds = %v, want 2", wbft["requestTimeoutSeconds"])
	}
	if v, _ := wbft["blockPeriodSeconds"].(float64); v != 1 {
		t.Errorf("blockPeriodSeconds = %v, want 1", wbft["blockPeriodSeconds"])
	}
	if v, _ := wbft["epochLength"].(float64); v != 140 {
		t.Errorf("epochLength = %v, want 140", wbft["epochLength"])
	}
	if wbft["maxRequestTimeoutSeconds"] != nil {
		t.Errorf("maxRequestTimeoutSeconds = %v, want null", wbft["maxRequestTimeoutSeconds"])
	}

	init, ok := anzeon["init"].(map[string]any)
	if !ok {
		t.Fatalf("config.anzeon.init missing")
	}
	validators, ok := init["validators"].([]any)
	if !ok || len(validators) != 2 {
		t.Fatalf("validators wrong shape: %v", init["validators"])
	}
	if validators[0] != "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" {
		t.Errorf("validators[0] = %v", validators[0])
	}

	sysContracts, ok := anzeon["systemContracts"].(map[string]any)
	if !ok {
		t.Fatalf("systemContracts missing")
	}
	gov, ok := sysContracts["govValidator"].(map[string]any)
	if !ok {
		t.Fatalf("govValidator missing")
	}
	gvParams, ok := gov["params"].(map[string]any)
	if !ok {
		t.Fatalf("govValidator.params missing")
	}
	wantCSV := "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA,0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	if gvParams["validators"] != wantCSV {
		t.Errorf("govValidator.params.validators = %v, want %q", gvParams["validators"], wantCSV)
	}
	if gvParams["members"] != wantCSV {
		t.Errorf("govValidator.params.members = %v, want %q", gvParams["members"], wantCSV)
	}
	if gvParams["blsPublicKeys"] != "0xaaaa,0xbbbb" {
		t.Errorf("govValidator.params.blsPublicKeys = %v", gvParams["blsPublicKeys"])
	}
	minter, ok := sysContracts["govMinter"].(map[string]any)
	if !ok {
		t.Fatalf("govMinter missing")
	}
	if mp, _ := minter["params"].(map[string]any); mp["members"] != wantCSV {
		t.Errorf("govMinter.params.members = %v", mp["members"])
	}

	alloc, ok := got["alloc"].(map[string]any)
	if !ok {
		t.Fatalf("alloc missing")
	}
	// Node default-balance entries keyed by stripped (0x-less) address;
	// case is preserved from the metadata fixture.
	for _, k := range []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"} {
		entry, ok := alloc[k].(map[string]any)
		if !ok {
			t.Errorf("alloc[%q] missing", k)
			continue
		}
		if entry["balance"] != "0x84595161401484a000000" {
			t.Errorf("alloc[%q].balance = %v", k, entry["balance"])
		}
	}
	// User-specified override kept (lowercased).
	override, ok := alloc["1234567890abcdef1234567890abcdef12345678"].(map[string]any)
	if !ok {
		t.Fatalf("override alloc entry missing")
	}
	if override["balance"] != "0xDEADBEEF" {
		t.Errorf("override balance = %v, want 0xDEADBEEF", override["balance"])
	}
	// CREATE2 proxy appended (preserves original mixed casing).
	c2, ok := alloc["4e59b44847b379578588920cA78FbF26c0B4956C"].(map[string]any)
	if !ok {
		lowered, lok := alloc["4e59b44847b379578588920ca78fbf26c0b4956c"].(map[string]any)
		if !lok {
			t.Fatalf("CREATE2 proxy alloc entry missing")
		}
		c2 = lowered
	}
	if code, _ := c2["code"].(string); !strings.HasPrefix(code, "0x") {
		t.Errorf("CREATE2 code missing or malformed: %v", c2["code"])
	}
	if c2["balance"] != "0x0" {
		t.Errorf("CREATE2 balance = %v, want 0x0", c2["balance"])
	}

	// extraData — metadata wins over overrides.
	if ed, _ := got["extraData"].(string); ed != "0xfeedface" {
		t.Errorf("extraData = %q, want 0xfeedface", ed)
	}
}

func TestGenerateGenesis_EmptyTemplate(t *testing.T) {
	in := spec.GenesisInput{
		TemplateBytes: nil,
		OutputPath:    filepath.Join(t.TempDir(), "out.json"),
		NumValidators: 1,
	}
	if err := stablenet.New().GenerateGenesis(context.Background(), in); err == nil {
		t.Fatal("expected error for empty template")
	}
}

func TestGenerateGenesis_ZeroValidators(t *testing.T) {
	in := decodeGenesisInput(t)
	in.NumValidators = 0
	in.OutputPath = filepath.Join(t.TempDir(), "g.json")
	if err := stablenet.New().GenerateGenesis(context.Background(), in); err == nil {
		t.Fatal("expected error for zero num_validators")
	}
}

func TestGenerateGenesis_ExtraDataFallback(t *testing.T) {
	// Metadata without extraData => override wins ("0x00" in fixture).
	in := decodeGenesisInput(t)
	delete(in.Metadata, "extraData")
	in.OutputPath = filepath.Join(t.TempDir(), "g.json")
	if err := stablenet.New().GenerateGenesis(context.Background(), in); err != nil {
		t.Fatalf("GenerateGenesis: %v", err)
	}
	raw, err := os.ReadFile(in.OutputPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if ed, _ := got["extraData"].(string); ed != "0x00" {
		t.Errorf("extraData = %q, want 0x00", ed)
	}
}
