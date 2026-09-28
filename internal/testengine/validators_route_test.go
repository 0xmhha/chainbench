package testengine_test

import (
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chains, as a run does
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// TestWemixValidatorsDoNotGoThroughItsManifestMethod.
//
// go-wemix serves no wemix_getValidators — its "wemix" namespace holds the
// brioche reward calls and nothing else — yet the wemix manifest declares that
// method. A run that read the manifest string called it and got -32601 back,
// which read as "this chain cannot answer" about a chain that answers fine
// through its governance contract. Measured 2026-09-28 on the common-set sweep
// against go-wemix.
//
// What keeps that from coming back is not the manifest value but the route:
// the poa family supplies its own reader, so RunningValidators never reaches
// the method. This asserts the route, because that is the part a run depends
// on.
func TestWemixValidatorsDoNotGoThroughItsManifestMethod(t *testing.T) {
	p, err := registry.Get("wemix")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Family().(registry.RuntimeValidatorReader); !ok {
		t.Fatalf("the %s family supplies no runtime reader, so RunningValidators would fall back to %q — a method go-wemix does not serve",
			p.Family().ID(), p.Manifest().Consensus.ValidatorsMethod)
	}
}

// TestChainsWithNoRuntimeReaderDeclareAMethod: the other half of the same
// choice. A chain that does not read its own validators must name a method,
// or RunningValidators has nothing left to try.
func TestChainsWithNoRuntimeReaderDeclareAMethod(t *testing.T) {
	for _, chain := range []string{"stablenet", "wbft", "wemix"} {
		p, err := registry.Get(chain)
		if err != nil {
			t.Fatalf("%s: %v", chain, err)
		}
		if _, ok := p.Family().(registry.RuntimeValidatorReader); ok {
			continue
		}
		if p.Manifest().Consensus.ValidatorsMethod == "" {
			t.Errorf("%s has neither a runtime reader nor a declared method, so its validators cannot be read at all", chain)
		}
	}
}
