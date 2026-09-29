package testengine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestSatisfies(t *testing.T) {
	cases := []struct {
		name     string
		required []string
		provided []string
		want     bool
	}{
		{"no requirements", nil, []string{"rpc"}, true},
		{"all present", []string{"rpc", "ws"}, []string{"rpc", "ws", "consensus"}, true},
		{"one missing", []string{"rpc", "consensus"}, []string{"rpc", "ws"}, false},
		{"none provided", []string{"rpc"}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := satisfies(tc.required, tc.provided); got != tc.want {
				t.Fatalf("satisfies(%v,%v) = %v, want %v", tc.required, tc.provided, got, tc.want)
			}
		})
	}
}

func TestApplicableWithCaps(t *testing.T) {
	applies := applicableWithCaps("stablenet", []string{"rpc"})
	// chain matches, no requirements -> applies.
	if !applies(dsl.Spec{}).Runs {
		t.Fatal("empty spec should apply")
	}
	// requires a capability the target provides.
	if !applies(dsl.Spec{Requires: []string{"rpc"}}).Runs {
		t.Fatal("spec requiring rpc should apply against an rpc target")
	}
	// requires a capability the target lacks -> skip.
	if applies(dsl.Spec{Requires: []string{"ws"}}).Runs {
		t.Fatal("spec requiring ws should not apply against an rpc-only target")
	}
	// wrong chain -> skip regardless of capabilities.
	if applies(dsl.Spec{ApplicableChains: "wbft"}).Runs {
		t.Fatal("spec for another chain should not apply")
	}
}

// TestApplicableWithCaps_SkipsOnTurnsASkipIntoADecision.
//
// Without it a skip is silent, and a run that skipped a third of its cases
// reports no failure — measured on the common set against go-wemix. With it
// the spec says where it expects to be skipped and the gate says whether that
// held.
func TestApplicableWithCaps_SkipsOnTurnsASkipIntoADecision(t *testing.T) {
	applies := applicableWithCaps("wemix", []string{"rpc"})
	needsWS := []string{"ws"}

	// Declared nothing: every skip is foreseen, which is how it behaved before
	// the field existed.
	if got := applies(dsl.Spec{Requires: needsWS}); got.Runs || !got.Foreseen || got.Stale {
		t.Errorf("a spec with no skipsOn gave %+v, want a foreseen skip", got)
	}
	// Declared this chain: the skip is the one it named.
	if got := applies(dsl.Spec{Requires: needsWS, SkipsOn: []string{"wemix"}}); got.Runs || !got.Foreseen || got.Stale {
		t.Errorf("a declared skip gave %+v, want a foreseen skip", got)
	}
	// Declared another chain: skipping here is not what it said.
	if got := applies(dsl.Spec{Requires: needsWS, SkipsOn: []string{"wbft"}}); got.Runs || got.Foreseen {
		t.Errorf("an undeclared skip gave %+v, want Runs=false Foreseen=false", got)
	}
	// Declared it must run everywhere, and it cannot here.
	if got := applies(dsl.Spec{Requires: needsWS, SkipsOn: []string{}}); got.Runs || got.Foreseen {
		t.Errorf("an empty skipsOn gave %+v, want the skip unforeseen", got)
	}
	// Declared a skip that does not happen: the declaration is stale.
	if got := applies(dsl.Spec{Requires: []string{"rpc"}, SkipsOn: []string{"wemix"}}); !got.Runs || !got.Stale {
		t.Errorf("a stale declaration gave %+v, want Runs=true Stale=true", got)
	}
	// Declared a skip elsewhere and runs here: nothing to report.
	if got := applies(dsl.Spec{Requires: []string{"rpc"}, SkipsOn: []string{"wbft"}}); !got.Runs || got.Stale {
		t.Errorf("a spec that runs here gave %+v, want Runs=true Stale=false", got)
	}
}
