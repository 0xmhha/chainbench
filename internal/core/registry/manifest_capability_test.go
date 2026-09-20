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
	// A template that switches both on, so this test keeps asking what it asked:
	// that a manifest's own facts become capabilities.
	tmpl := []byte(`{"config":{"istanbulBlock":0,"bohoBlock":0}}`)
	got := manifestFor().DerivedCapabilities(tmpl)

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
	if got := (registry.Manifest{}).DerivedCapabilities(nil); len(got) != 0 {
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

// TestDerivedCapabilities_ForkOnlyWhenTheGenesisTurnsItOn pins the half of the
// rule that was missing: Genesis.Hardforks lists the forks a chain KNOWS, and a
// capability says what a network HAS.
//
// stablenet is the case that showed the difference. Its manifest names applepie
// and boho and its template switches on neither, so the gate let a case that
// requires fork:boho run on a network without it — and the case failed on an
// assertion about govMinter v2 rather than skipping, which reads as a broken
// chain instead of an ineligible network.
//
// CapPrecompile already took this care, for this same fork. Both halves of the
// rule live here now.
func TestDerivedCapabilities_ForkOnlyWhenTheGenesisTurnsItOn(t *testing.T) {
	m := manifestFor() // knows istanbul and boho

	onlyIstanbul := []byte(`{"config":{"istanbulBlock":0,"anzeon":{}}}`)
	got := m.DerivedCapabilities(onlyIstanbul)
	if !slices.Contains(got, "fork:istanbul") {
		t.Errorf("the template switches istanbul on, so it is a capability: %v", got)
	}
	if slices.Contains(got, "fork:boho") {
		t.Errorf("the template does not switch boho on, so no network composed from it answers for boho: %v", got)
	}

	// An engine section counts as switching a fork on: that is how the chains
	// carry the ones whose configuration travels as a block rather than a number.
	if got := m.DerivedCapabilities([]byte(`{"config":{"boho":{"systemContracts":{}}}}`)); !slices.Contains(got, "fork:boho") {
		t.Errorf("a boho section is boho: %v", got)
	}

	// No static template means the family writes the genesis, and what it turns
	// on cannot be read here — the declared list is the best available answer.
	if got := m.DerivedCapabilities(nil); !slices.Contains(got, "fork:boho") {
		t.Errorf("without a template the declared forks stand: %v", got)
	}
}

// TestDerivedCapabilities_EveryChainAdvertisesOnlyForksItsTemplateHas runs the
// rule over the manifests that ship, so a chain whose template stops carrying a
// fork it declares — or whose declaration grows one the template lacks — is
// caught here rather than by a case failing on an assertion in a live run.
func TestDerivedCapabilities_EveryChainAdvertisesOnlyForksItsTemplateHas(t *testing.T) {
	for _, id := range registry.Names() {
		p, err := registry.Get(id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		tmpl := p.GenesisTemplate()
		if len(tmpl) == 0 {
			continue // no static template: nothing to check it against
		}
		for _, c := range p.Manifest().DerivedCapabilities(tmpl) {
			fork, ok := strings.CutPrefix(c, registry.CapFork)
			if !ok {
				continue
			}
			if !registry.ForkActiveIn(tmpl, fork) {
				t.Errorf("%s advertises %q but its genesis template does not switch it on", id, c)
			}
		}
	}
}
