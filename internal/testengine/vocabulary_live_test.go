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
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestSuite_Live_NewVocabulary drives the DSL vocabulary added for the legacy
// suite migration against a real 4-node stablenet composed through the suite
// path: step value binding, the read action, faucet, event logs, derived gas
// reads, a generic chain-namespace call, a WebSocket subscription, and
// node-level fault injection (which since R4 acts through the workspace's own
// node verbs).
//
// Unit tests cover each of these against a mock RPC; this is the proof they
// work against a real chain.
//
// Gated on GSTABLE_BIN; CI skips it and stays green.
func TestSuite_Live_NewVocabulary(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the vocabulary live e2e")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GSTABLE_BIN=%q: %v", bin, err)
	}

	ws, err := os.MkdirTemp("/tmp", "v")
	if err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(ws) })

	// The funded preset account, and an address that starts with nothing.
	const (
		funded = "0xc17d493883eaa3b4cceb0f214b273392d562f9d8"
		fresh  = "0x00000000000000000000000000000000000000aa"
	)

	specs := []map[string]any{
		{
			"schemaVersion": "1",
			"id":            "vocabulary-reads",
			"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
			"steps": []map[string]any{
				// read + save, then compare a later read against it (cross-call).
				{"read": map[string]any{"source": "blockNumber", "save": "head"}},
				{"read": map[string]any{"source": "balanceAt", "address": funded, "save": "funderBalance"}},
			},
			"assertions": []map[string]any{
				{"assert": "blockNumber", "compare": "GreaterOrEqual", "expected": "$head"},
				{"assert": "balanceAt", "address": funded, "compare": "Equal", "expected": "$funderBalance"},
				// A chain namespace read through the generic call.
				{"assert": "rpcCall", "method": "eth_chainId", "compare": "Equal", "expected": "0x205b"},
				// Derived gas: the suggested price is at least the base fee.
				{"assert": "gasPrice", "compare": "GreaterOrEqual", "expected": "0"},
				// The WebSocket transport actually carries heads.
				{"assert": "wsSubscribe", "on": "bp1", "event": "newHeads", "count": 1, "timeout": "30s"},
			},
		},
		{
			"schemaVersion": "1",
			"id":            "vocabulary-funding",
			"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
			"steps": []map[string]any{
				{"faucet": map[string]any{"from": funded, "to": fresh, "amount": "1000000000000000000", "save": "funding"}},
			},
			"assertions": []map[string]any{
				{"assert": "txStatus", "hash": "$funding", "expected": "0x1"},
				{"assert": "balanceAt", "address": fresh, "compare": "GreaterOrEqual", "expected": "1000000000000000000"},
				// The funding transaction is on chain, so a log query over the
				// range returns without error (a native transfer emits none).
				{"assert": "logs", "fromBlock": "0x0", "toBlock": "latest", "compare": "GreaterOrEqual", "expected": "0"},
			},
		},
		{
			"schemaVersion": "1",
			"id":            "vocabulary-fault",
			"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
			"steps": []map[string]any{
				// Stop one of four validators: 3/4 still meets quorum, so the
				// chain must keep producing.
				{"stopNode": map[string]any{"on": "bp4"}},
			},
			"assertions": []map[string]any{
				{"assert": "blockAdvance", "on": "bp1", "timeout": "60s"},
			},
			"postActions": []map[string]any{
				{"startNode": map[string]any{"on": "bp4"}},
			},
		},
	}
	paths := make([]string, 0, len(specs))
	for _, m := range specs {
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal spec: %v", err)
		}
		p := filepath.Join(ws, m["id"].(string)+".json")
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	out, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{
		SpecPaths:  paths,
		DataDir:    ws,
		Binary:     bin,
		KeysDir:    filepath.Join(repoRoot(t), "keys", "preset"),
		WaitBlocks: 1,
	})
	if err != nil {
		t.Fatalf("RunSuite: %v (setup %v)", err, out.SetupSteps)
	}
	s := out.Summary.Summary
	if s.Pass != len(specs) || s.Fail > 0 || s.Blocked > 0 {
		t.Fatalf("summary %+v, want all %d specs passing (session %s)", s, len(specs), out.SessionRoot)
	}
	t.Logf("all %d vocabulary specs passed live; session at %s", s.Pass, out.SessionRoot)
}
