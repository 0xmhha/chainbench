package preflight_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/preflight"
)

// alive is a liveness probe that says every node answers, so these cases turn on
// the paper comparison alone.
func alive(context.Context, preflight.Node) (bool, string) { return true, "" }

func composed() (preflight.Have, preflight.Want) {
	have := preflight.Have{
		Chain: "stablenet", Binary: "gstable", KeysDir: "keys/preset", Validators: 4,
		GenesisDeclared: "aaaa", Started: true,
		Nodes: []preflight.Node{{Index: 1, PID: 11}, {Index: 2, PID: 12}, {Index: 3, PID: 13}, {Index: 4, PID: 14}},
	}
	want := preflight.Want{
		Chain: "stablenet", Binary: "gstable", KeysDir: "keys/preset", Validators: 4,
		GenesisDeclared: "aaaa",
	}
	return have, want
}

// TestCheck_ADifferentGenesisIsADifferentChain is the case the comparison could
// not see. Everything cheaper agrees — chain, binary, keys, counts, every node
// alive — and only the declared genesis differs, which is exactly the shape two
// specs take when one of them enables a fork. Reusing here runs a test against a
// chain it did not declare.
func TestCheck_ADifferentGenesisIsADifferentChain(t *testing.T) {
	have, want := composed()
	want.GenesisDeclared = "bbbb"
	d := preflight.Check(context.Background(), have, want, alive)
	if d.Verdict != preflight.RebuildAll {
		t.Fatalf("verdict = %v, want RebuildAll (reasons: %v)", d.Verdict, d.Reasons)
	}
	var named bool
	for _, r := range d.Reasons {
		if r == "genesis differs" {
			named = true
		}
	}
	if !named {
		t.Errorf("the decision does not say why: %v", d.Reasons)
	}
}

// TestCheck_TheSameGenesisStillReuses keeps the fix from becoming "always
// rebuild": the digest matching is the ordinary case and must not cost a rebuild.
func TestCheck_TheSameGenesisStillReuses(t *testing.T) {
	have, want := composed()
	if d := preflight.Check(context.Background(), have, want, alive); d.Verdict != preflight.Reuse {
		t.Fatalf("verdict = %v, want Reuse (reasons: %v)", d.Verdict, d.Reasons)
	}
}

// TestCheck_AnUnknownGenesisOnEitherSideDoesNotForceARebuild pins the deliberate
// gap. A workspace composed before the request was recorded has no genesis to
// digest, and making that a rebuild would tear down every network composed by an
// older build. It is a known blind spot, not an oversight, and it is the reason
// the guard checks both sides for emptiness.
func TestCheck_AnUnknownGenesisOnEitherSideDoesNotForceARebuild(t *testing.T) {
	for name, mutate := range map[string]func(*preflight.Have, *preflight.Want){
		"the composition predates the recorded request": func(h *preflight.Have, _ *preflight.Want) { h.GenesisDeclared = "" },
		"the request declares no genesis at all":        func(_ *preflight.Have, w *preflight.Want) { w.GenesisDeclared = "" },
	} {
		t.Run(name, func(t *testing.T) {
			have, want := composed()
			want.GenesisDeclared = "bbbb"
			mutate(&have, &want)
			if d := preflight.Check(context.Background(), have, want, alive); d.Verdict != preflight.Reuse {
				t.Fatalf("verdict = %v, want Reuse (reasons: %v)", d.Verdict, d.Reasons)
			}
		})
	}
}
