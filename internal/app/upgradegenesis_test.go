package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chains, as a surface does
)

// UpgradeGenesis builds the one file a handoff runs on: the from-chain's genesis
// with the successor's fork section merged in. It was uncovered, and the thing it
// does was described WRONGLY in a profile comment for as long as nobody checked —
// the note claimed extra_data was the validators' RLP encoding, when the merge
// never carries the wbft genesis's extraData at all.
//
// So what is pinned here is the shape of the merge: what it adds, and what it
// leaves exactly as it found it.

const goldenProfile = "../../profiles/wemix-upgrade.yaml"

// baseGenesis writes a minimal from-chain genesis: the fork keys
// ValidateForks requires, a recognisable extraData, and an alloc.
func baseGenesis(t *testing.T) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"config": map[string]any{
			"chainId": 8285, "homesteadBlock": 0, "eip150Block": 0, "eip155Block": 0,
			"eip158Block": 0, "byzantiumBlock": 0, "constantinopleBlock": 0,
			"petersburgBlock": 0, "istanbulBlock": 0,
		},
		"extraData": "0x77656d69782d62617365", // "wemix-base"
		"alloc":     map[string]any{"0x1111111111111111111111111111111111111111": map[string]any{"balance": "0x1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "base.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func mustUpgradeGenesis(t *testing.T) (UpgradeGenesisOut, map[string]any) {
	t.Helper()
	out, err := UpgradeGenesis(Deps{}, goldenProfile, baseGenesis(t))
	if err != nil {
		t.Fatalf("UpgradeGenesis: %v", err)
	}
	var g map[string]any
	if err := json.Unmarshal(out.Genesis, &g); err != nil {
		t.Fatalf("merged genesis is not valid JSON: %v", err)
	}
	return out, g
}

// TestUpgradeGenesis_AddsTheForkSectionAndNothingElse is the property a wrong
// profile comment rested on. The merge lifts config.<fork> out of the successor's
// genesis and sets <fork>Block; everything else in the file is the from-chain's,
// including its extraData.
func TestUpgradeGenesis_AddsTheForkSectionAndNothingElse(t *testing.T) {
	_, g := mustUpgradeGenesis(t)

	if got := g["extraData"]; got != "0x77656d69782d62617365" {
		t.Errorf("extraData = %v; the merge must keep the from-chain's, not the successor's", got)
	}
	alloc, _ := g["alloc"].(map[string]any)
	if len(alloc) != 1 {
		t.Errorf("alloc has %d entries, want the from-chain's one", len(alloc))
	}

	cfg, _ := g["config"].(map[string]any)
	base := map[string]bool{
		"chainId": true, "homesteadBlock": true, "eip150Block": true, "eip155Block": true,
		"eip158Block": true, "byzantiumBlock": true, "constantinopleBlock": true,
		"petersburgBlock": true, "istanbulBlock": true,
	}
	added := map[string]bool{"croissant": true, "croissantBlock": true}
	for k := range cfg {
		if !base[k] && !added[k] {
			t.Errorf("config gained %q; the merge adds only the fork section and its block", k)
		}
	}
	for k := range added {
		if _, ok := cfg[k]; !ok {
			t.Errorf("config is missing %q", k)
		}
	}
}

// TestUpgradeGenesis_TheForkSectionCarriesTheProfilesValidators: the section is
// data from the successor's own template, substituted with the profile's set —
// not a constant in the upgrade package. A section with no validators would
// produce a chain with nobody to seal after the fork.
func TestUpgradeGenesis_TheForkSectionCarriesTheProfilesValidators(t *testing.T) {
	_, g := mustUpgradeGenesis(t)
	cfg := g["config"].(map[string]any)
	cro, ok := cfg["croissant"].(map[string]any)
	if !ok {
		t.Fatalf("croissant section is not an object: %T", cfg["croissant"])
	}
	init, ok := cro["init"].(map[string]any)
	if !ok {
		t.Fatalf("croissant has no init section: %v", cro)
	}
	vals, _ := init["validators"].([]any)
	keys, _ := init["blsPublicKeys"].([]any)
	if len(vals) == 0 {
		t.Fatal("the fork section declares no validators; nothing would seal after the fork")
	}
	if len(vals) != len(keys) {
		t.Errorf("%d validators but %d BLS keys — a set that passes genesis and fails to sign", len(vals), len(keys))
	}
}

// TestUpgradeGenesis_ProjectsTheNodesProducersFirst: the plan's order is what
// every caller indexes by, and producers come first. A projection that reordered
// them would hand a producer's port set to a successor.
func TestUpgradeGenesis_ProjectsTheNodesProducersFirst(t *testing.T) {
	out, _ := mustUpgradeGenesis(t)
	if out.Plan.From != "wemix" || out.Plan.To != "wbft" || out.Plan.AtFork != "croissant" {
		t.Fatalf("plan = %+v, want the profile's wemix -> wbft at croissant", out.Plan)
	}
	if len(out.Nodes) != out.Plan.Nodes {
		t.Fatalf("%d projected nodes for a plan of %d", len(out.Nodes), out.Plan.Nodes)
	}
	var seenSuccessor bool
	var producers, successors int
	netID := out.Nodes[0].NetworkID
	for _, n := range out.Nodes {
		if n.Producer {
			if seenSuccessor {
				t.Errorf("node%d is a producer after a successor; producers come first", n.Index)
			}
			producers++
		} else {
			seenSuccessor = true
			successors++
		}
		if n.NetworkID != netID {
			t.Errorf("node%d is on network %d, the first is on %d — one handoff is one network",
				n.Index, n.NetworkID, netID)
		}
		if n.P2P == 0 || n.HTTP == 0 {
			t.Errorf("node%d was projected with no ports", n.Index)
		}
	}
	if producers == 0 || successors == 0 {
		t.Errorf("a handoff needs both sides: %d producers, %d successors", producers, successors)
	}
}

func TestUpgradeGenesis_Refusals(t *testing.T) {
	base := baseGenesis(t)

	if _, err := UpgradeGenesis(Deps{}, "/no/such/profile.yaml", base); err == nil {
		t.Error("a missing profile was accepted")
	}

	_, err := UpgradeGenesis(Deps{}, goldenProfile, "/no/such/genesis.json")
	if err == nil || !strings.Contains(err.Error(), "read from-genesis") {
		t.Errorf("a missing from-genesis should be named as such: %v", err)
	}

	// A profile naming a chain nobody registered cannot be planned: the chain is
	// where the genesis template and the consensus family come from.
	raw, rerr := os.ReadFile(goldenProfile)
	if rerr != nil {
		t.Fatal(rerr)
	}
	bogus := filepath.Join(t.TempDir(), "bogus.yaml")
	if werr := os.WriteFile(bogus, []byte(strings.Replace(string(raw), "to: wbft", "to: nope", 1)), 0o600); werr != nil {
		t.Fatal(werr)
	}
	if _, err := UpgradeGenesis(Deps{}, bogus, base); err == nil {
		t.Error("a profile naming an unregistered to-chain was accepted")
	} else if !strings.Contains(err.Error(), "nope") {
		t.Errorf("the refusal should name the chain: %v", err)
	}
}
