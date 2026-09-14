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

// TestRegister_RefusesAHalfWiredPlugin: a chain is four choices, and a plugin
// missing one of them is a wiring mistake that used to surface much later.
//
// Each chain hand-wrote a four-method type, so the choices were made in four
// places per chain and nothing checked they were all made. Composing one
// literal makes an omission possible in a new way — a zero field — so
// registration refuses it where the mistake is, rather than as a nil
// dereference in the middle of a composition.
func TestRegister_RefusesAHalfWiredPlugin(t *testing.T) {
	m := registry.MustParseManifest([]byte(`{
		"id":"probe-only","binary":"gx","dialect":"geth114","chain_id":1,"network_id":1,
		"miner_recommit":"duration","bootstrap":{"type":"static"},"consensus_family":"wbft"}`))

	for name, p := range map[string]registry.ChainPlugin{
		"no family":   registry.StaticPlugin{M: m},
		"no protocol": registry.StaticPlugin{M: m, Fam: mustFamily(t, "wbft")},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("registering a plugin with %s must panic", name)
				}
			}()
			registry.Register(p)
		})
	}
}

func mustFamily(t *testing.T, id string) registry.ConsensusFamily {
	t.Helper()
	f, err := registry.FamilyByName(id)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestEveryChainComposesAllFourChoices reads the built-in chains the way an
// operator reads a chain's folder: what consensus, what accounts protocol, what
// flag vocabulary, what constants.
func TestEveryChainComposesAllFourChoices(t *testing.T) {
	for _, id := range registry.Names() {
		p, err := registry.Get(id)
		if err != nil {
			t.Fatalf("chain %q: %v", id, err)
		}
		m := p.Manifest()
		switch {
		case p.Family() == nil:
			t.Errorf("chain %q composes no consensus family", id)
		case p.Protocol().Name == "":
			t.Errorf("chain %q composes no accounts protocol", id)
		case m.Dialect == "":
			t.Errorf("chain %q names no flag vocabulary", id)
		case m.Binary == "":
			t.Errorf("chain %q names no binary", id)
		}
	}
}
