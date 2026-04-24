package stablenet

import (
	"testing"

	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

func TestAdapter_ConsensusRpcNamespace(t *testing.T) {
	if ns := New().ConsensusRpcNamespace(); ns != "istanbul" {
		t.Errorf("got %q, want istanbul", ns)
	}
}

func TestAdapter_ExtraStartFlags_Validator(t *testing.T) {
	want := "--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs --mine"
	if flags := New().ExtraStartFlags(spec.RoleValidator); flags != want {
		t.Errorf("got %q, want %q", flags, want)
	}
}

func TestAdapter_ExtraStartFlags_Endpoint(t *testing.T) {
	want := "--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs"
	if flags := New().ExtraStartFlags(spec.RoleEndpoint); flags != want {
		t.Errorf("got %q, want %q", flags, want)
	}
}
