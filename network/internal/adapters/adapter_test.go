package adapters_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/adapters"
)

func TestLoad_Stablenet(t *testing.T) {
	a, err := adapters.Load("stablenet")
	if err != nil {
		t.Fatalf("Load(stablenet): %v", err)
	}
	if a == nil {
		t.Fatal("Load returned nil adapter")
	}
	if ns := a.ConsensusRpcNamespace(); ns != "istanbul" {
		t.Errorf("stablenet namespace = %q, want istanbul", ns)
	}
}

func TestLoad_WBFT(t *testing.T) {
	a, err := adapters.Load("wbft")
	if err != nil {
		t.Fatalf("Load(wbft): %v", err)
	}
	if ns := a.ConsensusRpcNamespace(); ns != "istanbul" {
		t.Errorf("wbft namespace = %q, want istanbul", ns)
	}
}

func TestLoad_Wemix(t *testing.T) {
	a, err := adapters.Load("wemix")
	if err != nil {
		t.Fatalf("Load(wemix): %v", err)
	}
	if ns := a.ConsensusRpcNamespace(); ns != "wemix" {
		t.Errorf("wemix namespace = %q, want wemix", ns)
	}
}

func TestLoad_UnknownChainType(t *testing.T) {
	_, err := adapters.Load("ethereum")
	if !errors.Is(err, adapters.ErrUnknownChainType) {
		t.Errorf("err = %v, want ErrUnknownChainType", err)
	}
}

// TestSupportedTxTypes pins the fee-delegation support matrix that gates
// node.tx_fee_delegation_send: stablenet and wbft accept the 0x16
// FeeDelegateDynamicFeeTx, wemix does not.
func TestSupportedTxTypes(t *testing.T) {
	cases := []struct {
		chainType   string
		wantFeeDele bool
	}{
		{"stablenet", true},
		{"wbft", true},
		{"wemix", false},
	}
	for _, tc := range cases {
		t.Run(tc.chainType, func(t *testing.T) {
			a, err := adapters.Load(tc.chainType)
			if err != nil {
				t.Fatalf("Load(%s): %v", tc.chainType, err)
			}
			got := slices.Contains(a.SupportedTxTypes(), adapters.FeeDelegateDynamicFeeTxType)
			if got != tc.wantFeeDele {
				t.Errorf("%s supports fee-delegation = %v, want %v", tc.chainType, got, tc.wantFeeDele)
			}
		})
	}
}
