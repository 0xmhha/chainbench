package origin

import (
	"os"
	"strings"
	"testing"
)

// designDoc holds the priority line this package is the code for. Reading it
// here is what keeps the two from drifting: a rung added to one and not the
// other is the failure this vocabulary exists to end, and it is exactly how the
// three it replaced came to disagree.
const designDoc = "../../../docs/dev/architecture/design-v3/declaration-model-2026-09-22.md"

// TestLadderIsTheLineTheDesignFixed holds the order to the document that fixed
// it.
//
// The document writes the line as prose with four rungs, leaving out the two
// that never compete (a chain's own facts, a key set's material) because prose
// about precedence has no reason to mention a supplier with no rival. Those two
// are checked by position instead: they sit below the declaration, because a
// declaration that names a binary or a key overrides what the chain or the set
// would have supplied.
func TestLadderIsTheLineTheDesignFixed(t *testing.T) {
	raw, err := os.ReadFile(designDoc)
	if err != nil {
		t.Fatalf("read the design: %v", err)
	}
	doc := string(raw)

	// The line as the document draws it.
	const line = "체인 정의  <  환경 선언  <  케이스 override  <  server-set(자원 배정)  <  CLI/MCP"
	if !strings.Contains(doc, line) {
		t.Fatalf("the design no longer draws the line this ladder is built from;\nwanted the row:\n  %s", line)
	}

	for _, c := range []struct{ lower, higher Origin }{
		{FromDefault, FromChain},
		{FromChain, FromDeclaration},
		{FromKeySet, FromDeclaration},
		{FromBlueprint, FromDeclaration},
		{FromDeclaration, FromPlacement},
		{FromPlacement, FromCommand},
	} {
		if c.lower.Rank() >= c.higher.Rank() {
			t.Errorf("%s (%d) should rank below %s (%d)", c.lower, c.lower.Rank(), c.higher, c.higher.Rank())
		}
	}
}

func TestRank(t *testing.T) {
	if got := Origin("nonesuch").Rank(); got != 0 {
		t.Errorf("an unknown origin ranks %d, and it must rank below every known one", got)
	}
	if Origin("nonesuch").Known() {
		t.Error("an unknown origin reports itself known")
	}
	seen := map[int]Origin{}
	for _, o := range Ladder() {
		r := o.Rank()
		if r == 0 {
			t.Errorf("%s is on the ladder and ranks 0", o)
		}
		if other, dup := seen[r]; dup {
			t.Errorf("%s and %s share rank %d", other, o, r)
		}
		seen[r] = o
	}
}

// TestLadderIsNotSharedState: Ladder hands out the order, and a caller that
// sorts or truncates it must not change what everyone else reads.
func TestLadderIsNotSharedState(t *testing.T) {
	got := Ladder()
	if len(got) == 0 {
		t.Fatal("the ladder is empty")
	}
	got[0] = Origin("tampered")
	if Ladder()[0] == Origin("tampered") {
		t.Error("the ladder handed out its own slice")
	}
}
