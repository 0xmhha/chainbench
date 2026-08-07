package engine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/testspec"
)

func TestApplicableTo(t *testing.T) {
	cases := []struct {
		name    string
		chains  string
		target  string
		applies bool
	}{
		{"empty applies to all", "", "stablenet", true},
		{"listed applies", "stablenet,wbft", "stablenet", true},
		{"space separated", "wbft stablenet", "stablenet", true},
		{"not listed", "wbft,wemix", "stablenet", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := applicableTo(tc.target)(testspec.Spec{ApplicableChains: tc.chains})
			if got != tc.applies {
				t.Fatalf("applicableTo(%q)(%q) = %v, want %v", tc.target, tc.chains, got, tc.applies)
			}
		})
	}
}

func TestValidatorReqs(t *testing.T) {
	reqs := validatorReqs(4)(testspec.Spec{})
	if len(reqs) != 4 {
		t.Fatalf("reqs = %d, want 4", len(reqs))
	}
	for i, r := range reqs {
		if r.Role != node.RoleValidator {
			t.Fatalf("req %d role = %q, want validator", i, r.Role)
		}
		if r.Name == "" {
			t.Fatalf("req %d has no name", i)
		}
	}
}
