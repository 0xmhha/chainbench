package registry_test

import (
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// TestFamilyByName_ResolvesWhatIsLinkedIn is the switch this replaced.
//
// A chain names its family as a string, so something turns that string into an
// implementation. That was a switch in the package that loads a project-supplied
// manifest: adding a family meant editing a function in a package that has
// nothing to do with it, and that package had to import every family to name
// them. A family registers itself now, so the set is whatever the build linked.
func TestFamilyByName_ResolvesWhatIsLinkedIn(t *testing.T) {
	for _, id := range []string{"wbft", "poa"} {
		f, err := registry.FamilyByName(id)
		if err != nil {
			t.Errorf("family %q must resolve: %v", id, err)
			continue
		}
		if f.ID() != id {
			t.Errorf("family %q resolved to %q", id, f.ID())
		}
	}
}

// TestFamilyByName_SaysWhatIsAvailable: a family that is not there may be a
// typo or may simply not be in this build, and the caller cannot tell the two
// apart without being told what is.
func TestFamilyByName_SaysWhatIsAvailable(t *testing.T) {
	_, err := registry.FamilyByName("nosuchfamily")
	if err == nil {
		t.Fatal("an unknown family must be an error")
	}
	for _, want := range []string{"wbft", "poa"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name the available family %q", err, want)
		}
	}
}

// TestEveryChainsFamilyIsRegistered is the rule that keeps the two registries
// agreeing: a chain names a family in its manifest, and that name has to
// resolve or the chain cannot be composed at all.
//
// It is checked here rather than at each chain because the failure is not a
// chain's: it happens when a family stops registering itself, and then every
// chain that composes it breaks at once.
func TestEveryChainsFamilyIsRegistered(t *testing.T) {
	for _, id := range registry.Names() {
		p, err := registry.Get(id)
		if err != nil {
			t.Errorf("chain %q: %v", id, err)
			continue
		}
		named := p.Manifest().ConsensusFamily
		fam, err := registry.FamilyByName(named)
		if err != nil {
			t.Errorf("chain %q names family %q: %v", id, named, err)
			continue
		}
		// And the plugin's own family must be the one it names, or a manifest
		// says one thing while the code does another.
		if got := p.Family().ID(); got != named {
			t.Errorf("chain %q: manifest says family %q, plugin composes %q", id, named, got)
		}
		if fam.ID() != named {
			t.Errorf("family %q resolved to %q", named, fam.ID())
		}
	}
}

// TestValidatorsCarryBLS_IsTheFamilysAnswer keeps a key derivation from being
// decided by comparing a family's name.
//
// The app layer asked "is the family called wbft" to choose between deriving a
// BLS key and not. A family registered after that line was written falls into
// the else branch and produces identities without the material its own genesis
// then asks for — nothing fails at derivation, and the chain will not seal.
func TestValidatorsCarryBLS_IsTheFamilysAnswer(t *testing.T) {
	for name, want := range map[string]bool{"wbft": true, "poa": false} {
		f, err := registry.FamilyByName(name)
		if err != nil {
			t.Fatalf("family %q: %v", name, err)
		}
		if got := f.ValidatorsCarryBLS(); got != want {
			t.Errorf("family %q carries BLS = %v, want %v", name, got, want)
		}
	}
}
