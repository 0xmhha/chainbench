package testengine

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// upgradeEnv builds an env declaring a hardfork, with extra fields folded in.
func upgradeEnv(extra string) string {
	return `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"wemix",
	  "binaries":{"default":"gwemix","to":{"binary":"gwbft","chain":"wbft"}},
	  "topology":{"nodes":[
	    {"index":1,"role":"en","binary":"to"},
	    {"index":2,"role":"bp"}
	  ]},
	  "upgrade":{"from":"default"` + extra + `}}`
}

// TestUpgradeDecl_TheForkMustBeOneTheChainKnows.
//
// A misspelled fork is the quietest error the grammar allows. The genesis
// writes <name>Block for whatever name it is given, no chain reaches a fork by
// that name, and the run reports what the pre-fork build did — a pass for a
// change that was never exercised.
//
// The chain asked is the one the POST-fork binary runs, not the network's:
// wemix does not know croissant, and the wbft build that takes over from it
// does. Asking the network's chain would refuse every handoff there is.
//
// This replaces holding the declaration against a second document. Two
// documents agreeing said nothing about whether either was right, and only the
// declarations that named a second document were checked at all.
func TestUpgradeDecl_TheForkMustBeOneTheChainKnows(t *testing.T) {
	cases := []struct {
		name, extra, want string
	}{
		{"the successor's own fork", `,"fork":"croissant","at":20`, ""},
		{"its own block", `,"fork":"croissant","at":120`, ""},
		{"a fork nobody has", `,"fork":"croisant","at":20`, `wbft does not know it`},
		{"the producer's fork, not the successor's", `,"fork":"brioche","at":20`, ""},
		{"another chain's fork", `,"fork":"boho","at":20`, `wbft does not know it`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir("../..")
			spec := caseWithEnv(t, upgradeEnv(tc.extra))
			_, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("a correct declaration was refused: %v", err)
			case tc.want == "":
			case err == nil:
				t.Fatalf("a fork the chain does not know was accepted")
			case !strings.Contains(err.Error(), tc.want):
				t.Fatalf("the refusal should name the chain and the fork: %v", err)
			}
		})
	}
}

// TestUpgradeDecl_ARestartComposesAsAForkThatMovesTheWholeNetwork.
//
// The ordinary hardfork of one chain: every node runs the pre-fork build, and
// at the fork every one of them is relaunched on the build that knows what
// happens there. The declaration reaches the composition as a fork that moves
// the whole network rather than one that hands production to another set.
func TestUpgradeDecl_ARestartComposesAsAForkThatMovesTheWholeNetwork(t *testing.T) {
	// The fork is one the build taking over knows. A restart moves the whole
	// network onto that build, so the same rule applies to it as to a handoff.
	spec := caseWithEnv(t, upgradeEnv(`,"style":"restart","fork":"croissant","at":200`))
	t.Chdir("../..")
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("a restart was refused: %v", err)
	}
	f := comp.up.GenesisFork
	if f == nil || !f.Restart {
		t.Fatalf("the composition does not know the fork moves the whole network: %+v", f)
	}
}

// TestUpgradeDecl_RefusesADeclarationThatContradictsItself.
func TestUpgradeDecl_RefusesADeclarationThatContradictsItself(t *testing.T) {
	cases := []struct{ name, extra, want string }{
		{"no fork or block", ``, `says which fork it crosses and at which block`},
		{"a fork with no block", `,"fork":"croissant"`, `says which fork it crosses and at which block`},
		{"the retired profile spelling", `,"fork":"croissant","at":20,"profile":"b.yaml"`, "profile"},
		{"an unknown style", `,"fork":"croissant","at":20,"style":"rolling"`, "unknown upgrade style"},
		{"one side twice", `,"fork":"croissant","at":20,"from":"to"`, "on both sides"},
		{"a side no binary declares", `,"fork":"croissant","at":20,"from":"old"`, "binaries.old is missing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := `{"schemaVersion":"2","kind":"case","id":"c","chainPreset":` + upgradeEnv(tc.extra) + `,
			  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
			_, err := dsl.Parse([]byte(raw))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want a refusal saying %q, got %v", tc.want, err)
			}
		})
	}
}
