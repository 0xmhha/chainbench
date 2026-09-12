package chainsetup_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// TestGenesisDeclared_SeparatesRequestsThatWantDifferentChains is the test the
// preflight genesis comparison never had. The field, the comparison and the
// guard all existed; nothing filled the want side, so two environments differing
// only in their overlay reused one network and the second test ran against the
// first one's chain.
//
// Each case below is a pair that must NOT digest the same, because each would
// produce a different genesis document.
func TestGenesisDeclared_SeparatesRequestsThatWantDifferentChains(t *testing.T) {
	base := chainsetup.NetUpIn{Chain: "stablenet", Binary: "gstable", ChainID: 8283}
	cases := map[string]chainsetup.NetUpIn{
		"an overlay at all": func() chainsetup.NetUpIn {
			in := base
			in.OverlayPath = "/ws/env-genesis-overlay-c710e207.json"
			return in
		}(),
		"a different overlay": func() chainsetup.NetUpIn {
			in := base
			in.OverlayPath = "/ws/env-genesis-overlay-deadbeef.json"
			return in
		}(),
		"a dot-path set": func() chainsetup.NetUpIn {
			in := base
			in.GenesisSet = []string{"config.applepieBlock=0"}
			return in
		}(),
		"a different chain id": func() chainsetup.NetUpIn {
			in := base
			in.ChainID = 8284
			return in
		}(),
		"a finished genesis used verbatim": func() chainsetup.NetUpIn {
			in := base
			in.GenesisExisting = "/ws/given-genesis.json"
			return in
		}(),
		"a different template": func() chainsetup.NetUpIn {
			in := base
			in.TemplatePath = "/src/genesis-template.json"
			return in
		}(),
		"a different manifest": func() chainsetup.NetUpIn {
			in := base
			in.ManifestPath = "/src/chain.yaml"
			return in
		}(),
	}
	want := chainsetup.GenesisDeclared(base)
	seen := map[string]string{"the base request": want}
	for name, in := range cases {
		got := chainsetup.GenesisDeclared(in)
		if got == want {
			t.Errorf("%s digests the same as the base request — preflight would reuse a network built for a different chain", name)
		}
		if prev, dup := seen[got]; dup {
			t.Errorf("%s digests the same as %s", name, prev)
		}
		seen[got] = name
	}
}

// TestGenesisDeclared_OrderOfTheSetMatters pins a property that is easy to lose
// to a "tidy" sort: the dot-path set is applied in order and a later key wins,
// so two orders can produce two different genesis documents and must not be
// mistaken for one request.
func TestGenesisDeclared_OrderOfTheSetMatters(t *testing.T) {
	a := chainsetup.NetUpIn{GenesisSet: []string{"config.x=1", "config.x=2"}}
	b := chainsetup.NetUpIn{GenesisSet: []string{"config.x=2", "config.x=1"}}
	if chainsetup.GenesisDeclared(a) == chainsetup.GenesisDeclared(b) {
		t.Error("two orders of the same overrides digest alike; the last key wins, so they are different genesis documents")
	}
}

// TestGenesisDeclared_IsStableAndIgnoresWhatIsComparedElsewhere keeps the digest
// from becoming a second, noisier copy of the whole request. Keys and counts have
// their own comparisons that name themselves in the decision's reasons; folding
// them in here would report "genesis differs" for a difference the reader can
// already see named.
func TestGenesisDeclared_IsStableAndIgnoresWhatIsComparedElsewhere(t *testing.T) {
	in := chainsetup.NetUpIn{Chain: "stablenet", ChainID: 8283, GenesisSet: []string{"config.applepieBlock=0"}}
	first := chainsetup.GenesisDeclared(in)
	if second := chainsetup.GenesisDeclared(in); first != second {
		t.Fatalf("not stable: %s then %s", first, second)
	}
	for name, mutate := range map[string]func(*chainsetup.NetUpIn){
		"keys dir":   func(i *chainsetup.NetUpIn) { i.KeysDir = "keys/other" },
		"validators": func(i *chainsetup.NetUpIn) { i.Validators = 15 },
		"binary":     func(i *chainsetup.NetUpIn) { i.Binary = "/other/gstable" },
		"peering":    func(i *chainsetup.NetUpIn) { i.Peering = "proxied" },
	} {
		other := in
		mutate(&other)
		if got := chainsetup.GenesisDeclared(other); got != first {
			t.Errorf("%s changed the genesis digest; it is compared on its own and should not report as a genesis difference", name)
		}
	}
}

// TestGenesisDeclared_LengthPrefixesTheParts guards the concatenation: without a
// length prefix, an overlay path ending where a template path begins would digest
// the same as the pair shifted by one character, and two different chains would
// look like one.
func TestGenesisDeclared_LengthPrefixesTheParts(t *testing.T) {
	a := chainsetup.NetUpIn{OverlayPath: "ab", TemplatePath: "c"}
	b := chainsetup.NetUpIn{OverlayPath: "a", TemplatePath: "bc"}
	if chainsetup.GenesisDeclared(a) == chainsetup.GenesisDeclared(b) {
		t.Error("parts are concatenated without a separator: (ab, c) and (a, bc) collide")
	}
}
