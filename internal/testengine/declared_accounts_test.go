package testengine

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// A composed run prepares and funds its declared accounts before any spec
// runs, and the attach engine it runs through then sees them in its ring. The
// per-spec wrapper must leave them alone: it has no funded account to send
// from, so a second funding was a transaction from the empty address.
func TestDeclaredAccountsAlreadyPreparedAreNotFundedAgain(t *testing.T) {
	ring := store.NewKeySet(t.TempDir())
	if _, err := ring.Add(context.Background(), keyring.Label("dev1"), keyring.RandomSource{}, derive.AccountOnly); err != nil {
		t.Fatal(err)
	}
	ran := false
	inner := func(context.Context, dsl.Spec, session.Environment, session.TestRecord) (session.TestStatus, error) {
		ran = true
		return session.StatusPass, nil
	}
	// Nothing listens here: reaching the network at all is the failure.
	eps := []node.RPCEndpoint{{RPCURL: "http://127.0.0.1:1"}}
	spec := dsl.Spec{EnvAccounts: map[string]dsl.AccountV2{"dev1": {Fund: "10"}}}
	status, err := withDeclaredAccounts(inner, ring, eps)(context.Background(), spec, nil, nil)
	if err != nil || status != session.StatusPass || !ran {
		t.Fatalf("status %v, err %v, ran %v", status, err, ran)
	}
}
