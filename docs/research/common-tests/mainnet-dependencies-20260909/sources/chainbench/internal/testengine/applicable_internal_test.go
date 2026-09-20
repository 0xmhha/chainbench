package testengine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
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
			got := applicableTo(tc.target)(dsl.Spec{ApplicableChains: tc.chains})
			if got != tc.applies {
				t.Fatalf("applicableTo(%q)(%q) = %v, want %v", tc.target, tc.chains, got, tc.applies)
			}
		})
	}
}
