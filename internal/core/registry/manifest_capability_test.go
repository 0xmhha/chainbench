package registry_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// manifestFor builds a manifest carrying only what the derivation reads.
func manifestFor() registry.Manifest {
	m := registry.Manifest{
		ConsensusFamily: "wbft",
		TxTypes:         []string{"0x00", "0x16"},
		SystemContracts: map[string]string{
			"govMinter":  "0x0000000000000000000000000000000000001003",
			"govCouncil": "0x0000000000000000000000000000000000001004",
		},
	}
	m.Genesis.Hardforks = []string{"istanbul", "boho"}
	m.Genesis.EngineField = "anzeon"
	return m
}

// TestDerivedCapabilities_SaysWhatTheManifestAlreadyKnows.
//
// The facts are in the manifest already. Deriving them is what lets a case ask
// for a govMinter rather than name the chain it was written on, without the
// manifest carrying the same fact twice.
func TestDerivedCapabilities_SaysWhatTheManifestAlreadyKnows(t *testing.T) {
	got := manifestFor().DerivedCapabilities()

	for _, want := range []string{
		"contract:govMinter", "contract:govCouncil",
		"fork:istanbul", "fork:boho",
		"engine:anzeon", "family:wbft",
		"tx:0x00", "tx:0x16",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %q in %v", want, got)
		}
	}
	if !slices.IsSorted(got) {
		t.Errorf("the advertised set must be stable across runs, so it is sorted: %v", got)
	}
	// The chain's name is deliberately absent: a case gating on a name has to be
	// edited to meet a second chain.
	for _, c := range got {
		if c == "stablenet" || c == "chain:stablenet" {
			t.Errorf("the chain name must not be a capability: %v", got)
		}
	}
}

// TestDerivedCapabilities_EmptyManifestClaimsNothing: a chain that declares no
// contracts, forks or engine must not be credited with any. wemix declares no
// contracts because it deploys them at run time, and a contract requirement
// skipping there is the right answer.
func TestDerivedCapabilities_EmptyManifestClaimsNothing(t *testing.T) {
	if got := (registry.Manifest{}).DerivedCapabilities(); len(got) != 0 {
		t.Fatalf("an empty manifest derived %v", got)
	}
}

// TestMalformedCapability_TellsATypoFromAnUnmetRequirement.
//
// A requirement a chain does not provide is a SKIP, which is the point of
// gating. A requirement with a typo in its prefix would be that same silent SKIP
// on every chain, and a case that never runs anywhere looks exactly like one
// that is correctly gated out. Only the second is reported here.
func TestMalformedCapability_TellsATypoFromAnUnmetRequirement(t *testing.T) {
	for _, ok := range []string{
		"rpc", "consensus", "account-extra", "delayed-boho",
		"contract:govMinter", "fork:boho", "engine:anzeon", "family:poa", "tx:0x16",
	} {
		if why := registry.MalformedCapability(ok); why != "" {
			t.Errorf("%q must be accepted: %s", ok, why)
		}
	}
	for _, bad := range []string{"", "   ", "contract:", "fork: ", "contracts:govMinter", "chain:stablenet"} {
		why := registry.MalformedCapability(bad)
		if why == "" {
			t.Errorf("%q must be refused", bad)
			continue
		}
		if strings.Contains(bad, ":") && !strings.Contains(why, ":") {
			t.Errorf("the refusal of %q must name what it wanted: %s", bad, why)
		}
	}
}
