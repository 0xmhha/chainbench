package genesis_test

import (
	"context"
	"encoding/json"
	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
	src := genesis.PresetSource{KeysDir: dir}

	gen, err := src.Genesis(context.Background(), wbftTestPlugin(), genesis.Request{Validators: 0}) // whole preset
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
	gen, err := genesis.PresetSource{KeysDir: dir}.Genesis(context.Background(), wbftTestPlugin(), genesis.Request{Validators: 1})
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
	// The composition must apply ConfigOverrides and Overlay to whatever the
	// family's source produced. petersburgBlock is included so the
	// post-transform fork validation passes on the minimal test template.
	gen, err := genesis.Compose(context.Background(), wbftTestPlugin(),
		genesis.Request{Validators: 0},
		genesis.Config{
			KeysDir:         dir,
			ConfigOverrides: map[string]string{"petersburgBlock": "0", "bohoBlock": "10"},
			Overlay:         []byte(`{"alloc":{"00000000000000000000000000000000000000ff":{"balance":"0x2a"}}}`),
		})
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
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
	src := genesis.PresetSource{KeysDir: t.TempDir()} // no metadata.json
	if _, err := src.Genesis(context.Background(), wbftTestPlugin(), genesis.Request{Validators: 0}); err == nil {
		t.Fatal("expected error when preset metadata is absent")
	}
}

// TestPresetGenesisSource_ProducersFromPlacement is the E1 defect reproduction:
// for the topology node1=EN, node2=BP, node3=PN, node4=BP the genesis validator
// set must be the producers (node2, node4), not the first two nodes of the ring.
func TestPresetGenesisSource_ProducersFromPlacement(t *testing.T) {
	// Use the real committed ring: the preset loader verifies each identity
	// derives from its nodekey, so a fabricated one is rejected.
	presetDir := filepath.Join("..", "..", "..", "..", "keys", "preset")
	preset, err := store.LoadPreset(presetDir)
	if err != nil {
		t.Skipf("preset fixture unavailable: %v", err)
	}
	n2, ok2 := preset.Node(2)
	n4, ok4 := preset.Node(4)
	if !ok2 || !ok4 {
		t.Skip("preset ring has fewer than 4 nodes")
	}

	m, err := node.NewMap([]node.Placement{
		{Index: 1, Label: node.LabelFor(1), Role: node.RoleEN, Host: "10.0.0.1", Ports: node.Endpoints{P2P: 30301}},
		{Index: 2, Label: node.LabelFor(2), Role: node.RoleBP, Host: "10.0.0.2", Ports: node.Endpoints{P2P: 30301}},
		{Index: 3, Label: node.LabelFor(3), Role: node.RolePN, Host: "10.0.0.3", Ports: node.Endpoints{P2P: 30301}},
		{Index: 4, Label: node.LabelFor(4), Role: node.RoleBP, Host: "10.0.0.4", Ports: node.Endpoints{P2P: 30301}},
	})
	if err != nil {
		t.Fatalf("NewMap: %v", err)
	}

	gen, err := genesis.PresetSource{KeysDir: presetDir}.Genesis(context.Background(), wbftTestPlugin(),
		genesis.Request{Validators: 2, Nodes: m})
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}

	var doc struct {
		Validators []string `json:"validators"`
	}
	if err := json.Unmarshal(gen.Genesis, &doc); err != nil {
		t.Fatalf("parse genesis: %v\n%s", err, gen.Genesis)
	}
	want := []string{n2.Address, n4.Address}
	if !reflect.DeepEqual(doc.Validators, want) {
		t.Fatalf("genesis validators = %v, want the producers node2,node4 %v", doc.Validators, want)
	}
	// And the endpoint's identity must NOT be a validator.
	n1, _ := preset.Node(1)
	for _, v := range doc.Validators {
		if v == n1.Address {
			t.Fatalf("node1 (EN) %s wrongly seated as a validator", n1.Address)
		}
	}
}
