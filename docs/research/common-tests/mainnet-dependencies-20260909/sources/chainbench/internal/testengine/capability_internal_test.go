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
	if !applies(dsl.Spec{}) {
		t.Fatal("empty spec should apply")
	}
	// requires a capability the target provides.
	if !applies(dsl.Spec{Requires: []string{"rpc"}}) {
		t.Fatal("spec requiring rpc should apply against an rpc target")
	}
	// requires a capability the target lacks -> skip.
	if applies(dsl.Spec{Requires: []string{"ws"}}) {
		t.Fatal("spec requiring ws should not apply against an rpc-only target")
	}
	// wrong chain -> skip regardless of capabilities.
	if applies(dsl.Spec{ApplicableChains: "wbft"}) {
		t.Fatal("spec for another chain should not apply")
	}
}
