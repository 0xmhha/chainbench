package engine_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyreg"
	"github.com/0xmhha/chainbench/internal/engine"
)

// presetDir is the repository's shipped key set, used as a realistic fixture.
const presetDir = "../../keys/preset"

// derivingDeps derives an address deterministically from the key bytes so the
// identity check can be exercised without real crypto.
func derivingDeps(addrFor func(priv []byte) (string, error)) keyreg.Deps {
	return keyreg.Deps{DeriveAddress: addrFor}
}

func TestPresetKeySource_LoadsAndChecksCapacity(t *testing.T) {
	src := engine.PresetKeySource{Path: presetDir}
	if src.Dir() != presetDir {
		t.Fatalf("Dir() = %s, want %s", src.Dir(), presetDir)
	}

	ks, err := src.Ensure(context.Background(), 4)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if len(ks.Preset.Validators) == 0 {
		t.Error("preset carries no validators")
	}
	if _, ok := ks.Preset.Node(1); !ok {
		t.Error("preset carries no node1 identity")
	}

	// Asking for more nodes than the set has must fail at composition time, not
	// when a node later launches without an identity.
	if _, err := src.Ensure(context.Background(), len(ks.Preset.Nodes)+1); err == nil {
		t.Error("want an error when the key set is smaller than the topology")
	}
}

// TestGeneratedKeySource_NeedsNoExternalBinary is the K1 gate at the engine
// level. Generating a set used to require the go-wbft bootnode tool, which is
// what made the committed preset the only practical way to start a network.
// PATH is emptied so a surviving shell-out fails instead of quietly finding a
// binary the developer happens to have.
func TestGeneratedKeySource_NeedsNoExternalBinary(t *testing.T) {
	t.Setenv("PATH", "")
	const nodes = 2
	src := engine.GeneratedKeySource{Path: t.TempDir()}
	ks, err := src.Ensure(context.Background(), nodes)
	if err != nil {
		t.Fatalf("Ensure with no PATH: %v", err)
	}
	if len(ks.Preset.Nodes) != nodes {
		t.Fatalf("got %d identities, want %d", len(ks.Preset.Nodes), nodes)
	}
	// A wbft-family genesis reads the validator set out of extra-data, which is
	// derived from the BLS keys at genesis time — so what the generated set
	// must carry is the BLS material, not a precomputed extra-data.
	if len(ks.Preset.BLSKeys) != nodes {
		t.Errorf("generated set has %d BLS keys, want %d", len(ks.Preset.BLSKeys), nodes)
	}
	if ks.Preset.ExtraData != "" {
		t.Errorf("generated set stored a derived extraData: %q", ks.Preset.ExtraData)
	}
}

func TestGeneratedKeySource_ReusesAnExistingSet(t *testing.T) {
	// A directory that already holds a key set is loaded, not regenerated:
	// regenerating would hand the run different identities than the genesis it
	// may already have produced. No bootnode is configured, so a generation
	// attempt would fail and this test would catch it.
	dir := t.TempDir()
	copyPreset(t, presetDir, dir)

	src := engine.GeneratedKeySource{Path: dir}
	ks, err := src.Ensure(context.Background(), 4)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if ks.Dir != dir {
		t.Errorf("Dir = %s, want %s", ks.Dir, dir)
	}
	if len(ks.Preset.Nodes) == 0 {
		t.Error("reused set carries no node identities")
	}
}

func TestRegisterIdentities_RegistersEachNode(t *testing.T) {
	ks, err := engine.PresetKeySource{Path: presetDir}.Ensure(context.Background(), 4)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	// Address derivation is stubbed to return exactly what the metadata declares,
	// so this test covers registration; the mismatch path is the next test.
	byKey := map[string]string{}
	for _, n := range ks.Preset.Nodes {
		byKey[n.Nodekey.Hex()] = n.Address
	}
	reg := keyreg.New(t.TempDir(), derivingDeps(func(priv []byte) (string, error) {
		return byKey[hexOf(priv)], nil
	}))

	if err := engine.RegisterIdentities(context.Background(), reg, ks, 4); err != nil {
		t.Fatalf("RegisterIdentities: %v", err)
	}
	for i := 1; i <= 4; i++ {
		name := nodeName(i)
		k, ok := reg.Get(name)
		if !ok {
			t.Fatalf("%s was not registered", name)
		}
		want, _ := ks.Preset.Node(i)
		if !strings.EqualFold(k.Address, want.Address) {
			t.Errorf("%s address = %s, want %s", name, k.Address, want.Address)
		}
	}
}

func TestRegisterIdentities_RejectsDriftedIdentity(t *testing.T) {
	ks, err := engine.PresetKeySource{Path: presetDir}.Ensure(context.Background(), 4)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	// A key set whose declared address no longer matches its key material would
	// launch nodes signing as one address while the genesis registers another.
	reg := keyreg.New(t.TempDir(), derivingDeps(func([]byte) (string, error) {
		return "0x00000000000000000000000000000000deadbeef", nil
	}))

	err = engine.RegisterIdentities(context.Background(), reg, ks, 4)
	if err == nil {
		t.Fatal("want an error when a declared identity does not match its key")
	}
	if !strings.Contains(err.Error(), "node1") {
		t.Errorf("error should name the offending node, got: %v", err)
	}
}

func TestRegisterIdentities_NilRegistryIsANoOp(t *testing.T) {
	ks, err := engine.PresetKeySource{Path: presetDir}.Ensure(context.Background(), 4)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if err := engine.RegisterIdentities(context.Background(), nil, ks, 4); err != nil {
		t.Errorf("nil registry should be a no-op, got: %v", err)
	}
}

// nodeName is the registry name for a node identity.
func nodeName(i int) string { return fmt.Sprintf("node%d", i) }

// hexOf renders key bytes the way the metadata stores them.
func hexOf(b []byte) string { return hex.EncodeToString(b) }

// copyPreset seeds to with the metadata a key set is loaded from.
func copyPreset(t *testing.T, from, to string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(from, "metadata.json"))
	if err != nil {
		t.Fatalf("read preset metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(to, "metadata.json"), raw, 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}
