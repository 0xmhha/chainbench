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

// TestUpgradeDecl_TheCaseIsHeldToTheForkItNames.
//
// The name of the fork and the block it sits on are different kinds of fact,
// and only one of them is the preset's.
//
// The name says which change is under test. A case wrong about that exercises
// one fork and reports a pass for another, so it is refused.
//
// The block is a schedule this run chooses. A case that has to write state
// while the pre-fork build is still sealing needs the fork far enough out to
// finish — with the preset's block 20 the chain reaches the fork and stops
// during bring-up — so it sets its own and is not contradicted.
func TestUpgradeDecl_TheCaseIsHeldToTheForkItNames(t *testing.T) {
	cases := []struct {
		name, extra, want string
	}{
		{"the right fork and block", `,"preset":"wemix-upgrade","fork":"croissant","at":20`, ""},
		{"saying nothing is fine", `,"preset":"wemix-upgrade"`, ""},
		{"the wrong fork", `,"preset":"wemix-upgrade","fork":"boho"`, `says it tests the "boho" fork`},
		{"its own block", `,"preset":"wemix-upgrade","at":120`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A named preset resolves under the working directory, the same
			// rule the default key set follows, so the check reads the real
			// file rather than a fixture that could drift from it.
			t.Chdir("../..")
			spec := caseWithEnv(t, upgradeEnv(tc.extra))
			_, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("a correct declaration was refused: %v", err)
			case tc.want == "":
			case err == nil:
				t.Fatalf("a declaration that disagrees with the preset was accepted")
			case !strings.Contains(err.Error(), tc.want):
				t.Fatalf("the refusal should say what disagrees: %v", err)
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
	spec := caseWithEnv(t, upgradeEnv(`,"style":"restart","fork":"boho","at":200`))
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
		{"no preset", ``, `needs a "preset"`},
		{"the retired profile spelling", `,"preset":"a","profile":"b.yaml"`, "profile"},
		{"an unknown style", `,"preset":"a","style":"rolling"`, "unknown upgrade style"},
		{"one side twice", `,"preset":"a","from":"to"`, "on both sides"},
		{"a side no binary declares", `,"preset":"a","from":"old"`, "binaries.old is missing"},
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
