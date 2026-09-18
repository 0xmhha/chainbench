package all_test

import (
	"testing"

	acct "github.com/0xmhha/accounts/tx"

	"github.com/0xmhha/chainbench/internal/accounts"
	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// TestAllChainsRegistered verifies the G0 vertical slice: every chain is
// registered, its manifest consensus_family matches the family plugin it
// composes, and its accounts protocol agrees with the manifest.
func TestAllChainsRegistered(t *testing.T) {
	want := map[string]struct {
		family    string
		namespace string
	}{
		"stablenet": {family: "wbft", namespace: "istanbul"},
		"wbft":      {family: "wbft", namespace: "istanbul"},
		"wemix":     {family: "poa", namespace: "wemix"},
	}

	got := registry.Names()
	if len(got) != len(want) {
		t.Fatalf("registered chains: got %v, want %d", got, len(want))
	}

	for id, exp := range want {
		p, err := registry.Get(id)
		if err != nil {
			t.Fatalf("Get(%q): %v", id, err)
		}
		m := p.Manifest()
		if m.ConsensusFamily != exp.family {
			t.Errorf("%s: manifest family %q, want %q", id, m.ConsensusFamily, exp.family)
		}
		if fam := p.Family(); fam.ID() != exp.family {
			t.Errorf("%s: family plugin id %q, want %q", id, fam.ID(), exp.family)
		}
		if ns := p.Family().RPCNamespace(); ns != exp.namespace {
			t.Errorf("%s: namespace %q, want %q", id, ns, exp.namespace)
		}
		// Manifest namespace and family plugin namespace must agree.
		if m.Consensus.RPCNamespace != p.Family().RPCNamespace() {
			t.Errorf("%s: manifest namespace %q != family namespace %q",
				id, m.Consensus.RPCNamespace, p.Family().RPCNamespace())
		}
		// Protocol must match the chain id.
		if p.Protocol().Name != id {
			t.Errorf("%s: protocol name %q", id, p.Protocol().Name)
		}
	}
}

// TestAccountProvider_FeeDelegationEverywhere pins that the AccountProvider
// boundary reports 0x16 support for all chains (a measured baseline) and Extra only for
// stablenet.
func TestAccountProvider_FeeDelegationEverywhere(t *testing.T) {
	for _, id := range registry.Names() {
		ap, err := accounts.ForChain(id)
		if err != nil {
			t.Fatalf("ForChain(%q): %v", id, err)
		}
		if !ap.SupportsTxType(acct.FeeDelegateDynamicFeeTxType) {
			t.Errorf("%s: expected 0x16 fee-delegation support", id)
		}
		wantExtra := id == "stablenet"
		if ap.HasAccountExtra() != wantExtra {
			t.Errorf("%s: HasAccountExtra=%v, want %v", id, ap.HasAccountExtra(), wantExtra)
		}
	}
}
