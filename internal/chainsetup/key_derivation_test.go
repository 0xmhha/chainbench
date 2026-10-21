package chainsetup

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
)

// TestKeyDerivation_FollowsEveryChainTheNetworkRuns covers a fork that hands
// the network to another family. The keys are made once, before genesis, and
// the genesis of the chain after the fork is built from that same ring — so a
// wemix network that becomes wbft needs BLS keys although wemix itself does not.
func TestKeyDerivation_FollowsEveryChainTheNetworkRuns(t *testing.T) {
	for _, tc := range []struct {
		name         string
		chain        string
		binaryChains map[string]string
		want         derive.Derivation
	}{
		{"poa alone", "wemix", nil, derive.AccountOnly},
		{"wbft alone", "wbft", nil, derive.WithBLS},
		{"poa handing over to wbft", "wemix", map[string]string{"next": "wbft"}, derive.WithBLS},
		{"poa binary that stays poa", "wemix", map[string]string{"old": "wemix"}, derive.AccountOnly},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{state: State{Chain: tc.chain, BinaryChains: tc.binaryChains}}
			got, err := w.keyDerivation()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("keyDerivation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestKeyDerivation_RejectsAnUnknownHandoverChain(t *testing.T) {
	w := &Workspace{state: State{Chain: "wemix", BinaryChains: map[string]string{"next": "no-such-chain"}}}
	if _, err := w.keyDerivation(); err == nil {
		t.Fatal("keyDerivation() accepted a fork target that resolves to no chain")
	}
}
