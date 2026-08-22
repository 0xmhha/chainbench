package engine_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/engine"
)

// wbftTestPlugin is a StaticPlugin with the real wbft family and a minimal
// template exercising the consensus-critical placeholders.
func wbftTestPlugin() registry.ChainPlugin {
	tmpl := `{"config":{"chainId":__CHAIN_ID__},` +
		`"validators":"__VALIDATORS_JSON__",` +
		`"blsKeys":"__BLS_PUBLIC_KEYS_JSON__",` +
		`"extraData":"__EXTRA_DATA__",` +
		`"alloc":__ALLOC_JSON__}`
	return registry.StaticPlugin{
		M:    registry.Manifest{ID: "wbfttest", Binary: "go-wbft", ChainID: 1337, ConsensusFamily: "wbft"},
		Fam:  wbftfam.New(),
		Tmpl: []byte(tmpl),
	}
}

// writePreset writes a metadata.json preset into a fresh dir and returns it.
func writePreset(t *testing.T, metadata string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(metadata), 0o600); err != nil {
		t.Fatalf("write preset: %v", err)
	}
	return dir
}

func TestPresetGenesisSource_BuildsGenesis(t *testing.T) {
	dir := writePreset(t, `{
		"password": "x",
		"validators": ["0xaaaa", "0xbbbb"],
		"blsPublicKeys": ["0x01", "0x02"],
		"extraData": "0xdeadbeef",
		"alloc": {}
	}`)
	src := engine.PresetGenesisSource{KeysDir: dir}

	gen, err := src.Genesis(context.Background(), wbftTestPlugin(), engine.GenesisRequest{Validators: 0}) // whole preset
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	out := string(gen.Genesis)
	for _, want := range []string{"1337", "0xaaaa", "0xbbbb", "0xdeadbeef"} {
		if !strings.Contains(out, want) {
			t.Fatalf("genesis missing %q:\n%s", want, out)
		}
	}
}

func TestPresetGenesisSource_TakeLimitsValidators(t *testing.T) {
	// Full-length values: narrowing the set drops the preset's extra-data, so
	// the builder derives a fresh one and needs real 20-byte addresses and
	// 48-byte BLS keys to do it.
	dir := writePreset(t, `{
		"validators": ["0xaaaa000000000000000000000000000000000001", "0xbbbb000000000000000000000000000000000002"],
		"blsPublicKeys": [
			"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		],
		"extraData": "0xdeadbeef",
		"alloc": {}
	}`)
	gen, err := engine.PresetGenesisSource{KeysDir: dir}.Genesis(context.Background(), wbftTestPlugin(), engine.GenesisRequest{Validators: 1})
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	out := string(gen.Genesis)
	if !strings.Contains(out, "0xaaaa0000") || strings.Contains(out, "0xbbbb0000") {
		t.Fatalf("Take(1) should keep only the first validator:\n%s", out)
	}
	// The preset's stored extra-data described both validators, so it must not
	// survive into a one-validator genesis.
	if strings.Contains(out, "0xdeadbeef") {
		t.Fatalf("narrowed genesis reused the full set's extraData:\n%s", out)
	}
}

func TestPresetGenesisSource_AppliesOverridesAndOverlay(t *testing.T) {
	dir := writePreset(t, `{
		"validators": ["0xaaaa"],
		"blsPublicKeys": ["0x01"],
		"extraData": "0xdeadbeef",
		"alloc": {}
	}`)
	// The source must thread ConfigOverrides and Overlay into BuildNetwork.
	// petersburgBlock is included so the post-transform fork validation passes on
	// the minimal test template.
	src := engine.PresetGenesisSource{
		KeysDir:         dir,
		ConfigOverrides: map[string]string{"petersburgBlock": "0", "bohoBlock": "10"},
		Overlay:         []byte(`{"alloc":{"00000000000000000000000000000000000000ff":{"balance":"0x2a"}}}`),
	}
	gen, err := src.Genesis(context.Background(), wbftTestPlugin(), engine.GenesisRequest{Validators: 0})
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	out := string(gen.Genesis)
	if !strings.Contains(out, `"bohoBlock":10`) {
		t.Errorf("config override not applied:\n%s", out)
	}
	if !strings.Contains(out, "00000000000000000000000000000000000000ff") {
		t.Errorf("overlay not merged:\n%s", out)
	}
}

func TestPresetGenesisSource_MissingPreset(t *testing.T) {
	src := engine.PresetGenesisSource{KeysDir: t.TempDir()} // no metadata.json
	if _, err := src.Genesis(context.Background(), wbftTestPlugin(), engine.GenesisRequest{Validators: 0}); err == nil {
		t.Fatal("expected error when preset metadata is absent")
	}
}
