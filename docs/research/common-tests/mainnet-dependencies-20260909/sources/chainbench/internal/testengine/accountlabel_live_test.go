package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestSuite_Live_AccountLabels proves the label chain end to end, with the
// chain as the witness.
//
// A label claims two things at once: that it names an address, and that the key
// behind that name is the one that signs for it. Checking either against our
// own values would only prove we are consistent with ourselves. So the last
// assertion reads the transaction back from the chain and compares the sender
// the NODE recovered from the signature against the address the label resolved
// to. A name bound to the wrong identity cannot produce that sender.
//
// The spec also uses the reserved funder name. Nothing about "faucet" is a new
// account — it is the name of the role, so a test asks for the funded account
// rather than for whichever address happens to hold the money in this key set.
//
// Gated on GSTABLE_BIN; CI skips it and stays green.
func TestSuite_Live_AccountLabels(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the account-label live e2e")
	}

	ws, err := os.MkdirTemp("/tmp", "al")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(ws) })

	keysDir := filepath.Join(repoRoot(t), "keys", "preset")
	set, err := store.LoadPreset(keysDir)
	if err != nil {
		t.Fatalf("load preset: %v", err)
	}
	node1, ok := set.Node(1)
	if !ok {
		t.Fatal("the preset has no node1")
	}
	// An address nobody starts with, so the faucet's effect is visible.
	const fresh = "0x00000000000000000000000000000000000000bb"

	spec := map[string]any{
		"schemaVersion": "1",
		"id":            "account-labels",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"steps": []map[string]any{
			// The funder is named by role, the recipient by address.
			{"faucet": map[string]any{"from": "faucet", "to": fresh, "amount": "1000000000000000000"}},
			// Both ends named: no address appears in this step at all.
			{"sendTx": map[string]any{"from": "node1", "to": "node2", "value": "0x1", "save": "sent"}},
		},
		"assertions": []map[string]any{
			{"assert": "balanceAt", "address": fresh, "compare": "GreaterOrEqual", "expected": "1000000000000000000"},
			{"assert": "txStatus", "hash": "$sent", "expected": "0x1"},
			// The witness: the sender the node recovered from the signature is
			// the address node1 resolved to.
			{"assert": "rpcCall", "method": "eth_getTransactionByHash", "params": []any{"$sent"},
				"select": "from", "compare": "EqualCI", "expected": node1.Address},
		},
	}
	b, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(ws, "account-labels.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	out, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{
		SpecPaths: []string{p}, DataDir: ws, Binary: bin, KeysDir: keysDir, WaitBlocks: 1,
	})
	if err != nil {
		t.Fatalf("RunSuite: %v (setup %v)", err, out.SetupSteps)
	}
	if s := out.Summary.Summary; s.Pass != 1 || s.Fail > 0 || s.Blocked > 0 {
		t.Fatalf("summary %+v, want the label spec passing (session %s)", s, out.SessionRoot)
	}
	t.Logf("node1 resolved to %s and the chain agrees it signed; session at %s", node1.Address, out.SessionRoot)
}
