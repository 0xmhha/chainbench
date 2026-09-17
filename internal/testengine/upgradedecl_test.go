package testengine

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// upgradeEnv builds an env declaring a handoff, with extra fields folded in.
func upgradeEnv(extra string) string {
	return `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"from":"gwemix","to":"gwbft"},
	  "upgrade":{"template":"t.json"` + extra + `}}`
}

// TestUpgradeDecl_TheCaseIsHeldToWhatItSaysAboutTheFork.
//
// The preset decides which fork and which block. A case may repeat them, and a
// repetition that disagrees is a case testing something other than what it
// claims — which reads as a pass, against the wrong fork.
func TestUpgradeDecl_TheCaseIsHeldToWhatItSaysAboutTheFork(t *testing.T) {
	cases := []struct {
		name, extra, want string
	}{
		{"the right fork and block", `,"preset":"wemix-upgrade","fork":"croissant","at":20`, ""},
		{"saying nothing is fine", `,"preset":"wemix-upgrade"`, ""},
		{"the wrong fork", `,"preset":"wemix-upgrade","fork":"boho"`, `says it tests the "boho" fork`},
		{"the wrong block", `,"preset":"wemix-upgrade","at":99`, "at block 99"},
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

// TestUpgradeDecl_RestartIsRefusedByName: the ordinary hardfork, where every
// node is relaunched on the post-fork binary, has a place in the grammar and no
// implementation. Accepting it and quietly running the other style is the shape
// this track keeps removing.
func TestUpgradeDecl_RestartIsRefusedByName(t *testing.T) {
	raw := `{"schemaVersion":"2","kind":"case","id":"c","env":` +
		upgradeEnv(`,"preset":"wemix-upgrade","style":"restart"`) + `,
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	_, err := dsl.Parse([]byte(raw))
	if err == nil {
		t.Fatal("an unbuilt style was accepted")
	}
	for _, want := range []string{"restart", "not built yet"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal should name it and say why: %v", err)
		}
	}
}

// TestUpgradeDecl_RefusesADeclarationThatContradictsItself.
func TestUpgradeDecl_RefusesADeclarationThatContradictsItself(t *testing.T) {
	cases := []struct{ name, extra, want string }{
		{"no preset and no profile", ``, `needs a "preset" or a "profile"`},
		{"both", `,"preset":"a","profile":"b.yaml"`, "name one"},
		{"an unknown style", `,"preset":"a","style":"rolling"`, "unknown upgrade style"},
		{"one side twice", `,"preset":"a","from":"to"`, "on both sides"},
		{"a side no binary declares", `,"preset":"a","from":"old"`, "binaries.old is missing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := `{"schemaVersion":"2","kind":"case","id":"c","env":` + upgradeEnv(tc.extra) + `,
			  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
			_, err := dsl.Parse([]byte(raw))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want a refusal saying %q, got %v", tc.want, err)
			}
		})
	}
}
