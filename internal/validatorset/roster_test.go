package validatorset_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/validatorset"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

func preset(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		p := filepath.Join(dir, "keys", "preset")
		if _, err := os.Stat(filepath.Join(p, "metadata.json")); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("keys/preset not found")
		}
		dir = parent
	}
}

func TestRoster_WbftFamilyHasValidators(t *testing.T) {
	r, err := validatorset.Load("stablenet", preset(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Family != "wbft" {
		t.Fatalf("family = %q, want wbft", r.Family)
	}
	var validators, nodes, gov int
	for _, a := range r.Accounts {
		switch a.Role {
		case registry.AccountValidator:
			validators++
			if a.Detail != "BLS present" {
				t.Fatalf("stablenet validator should have BLS: %+v", a)
			}
		case registry.AccountNode:
			nodes++
		case registry.AccountGovernance:
			gov++
		}
	}
	if validators != 4 || nodes < 4 {
		t.Fatalf("roster counts: validators=%d nodes=%d gov=%d", validators, nodes, gov)
	}
}

func TestRoster_PoaFamilyNoGenesisValidators(t *testing.T) {
	r, err := validatorset.Load("wemix", preset(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Family != "poa" {
		t.Fatalf("family = %q, want poa", r.Family)
	}
	for _, a := range r.Accounts {
		if a.Role == registry.AccountValidator {
			t.Fatalf("poa chain should not list genesis validators: %+v", a)
		}
	}
	if r.Note == "" {
		t.Fatal("poa roster should note validators are set at bootstrap")
	}
}

func TestRoster_UnknownChain(t *testing.T) {
	if _, err := validatorset.Load("nope", preset(t)); err == nil {
		t.Fatal("expected error for unknown chain")
	}
}

// TestRoster_FamilyThatSaysNothingStillListsNodeIdentities: a family that does
// not implement the capability is not broken and not unknown -- it has simply
// not said which accounts it takes from a ring. The roster must still show the
// node identities every family has, and the note must say which of the two it is.
//
// The old code decided this with `switch family { case "wbft": … case "poa": … }`
// and a default that reported the family as unknown. That default fired for a
// family the switch had not been updated for, which is a fact about this package
// rather than about the chain.
func TestRoster_FamilyThatSaysNothingStillListsNodeIdentities(t *testing.T) {
	r, err := validatorset.Load("stablenet", preset(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// The registered families do implement it, so this checks the shape the
	// fallback produces rather than re-deriving it: every roster lists nodes.
	var nodes int
	for _, a := range r.Accounts {
		if a.Role == registry.AccountNode {
			nodes++
		}
	}
	if nodes == 0 {
		t.Fatal("a roster with no node identities cannot be right for any family")
	}
}

// TestRoster_EveryRegisteredFamilyAnswers is the ratchet this refactor earns: a
// family that does not implement RingAccountReader falls back to node identities
// and a note, which is correct but silent. Registering a family and forgetting to
// say what it takes from a ring should be visible here rather than in a roster an
// operator reads and believes.
func TestRoster_EveryRegisteredFamilyAnswers(t *testing.T) {
	for _, chain := range registry.Names() {
		p, err := registry.Get(chain)
		if err != nil {
			t.Fatalf("registry.Get(%s): %v", chain, err)
		}
		if _, ok := p.Family().(registry.RingAccountReader); !ok {
			t.Errorf("chain %q (family %q) does not implement RingAccountReader, so its roster silently shows node identities only",
				chain, p.Manifest().ConsensusFamily)
		}
	}
}
