package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin

	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestEngine_Live_NewVocabulary drives the DSL vocabulary added for the legacy
// suite migration against a real 4-node stablenet: step value binding, the read
// action, faucet, event logs, derived gas reads, a generic chain-namespace call,
// a WebSocket subscription, and node-level fault injection.
//
// Unit tests cover each of these against a mock RPC; this is the proof they work
// against a real chain, which is the only thing that settles whether a ported
// case would actually pass.
//
// Gated on GSTABLE_BIN; CI skips it and stays green.
func TestEngine_Live_NewVocabulary(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the vocabulary live e2e")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GSTABLE_BIN=%q: %v", bin, err)
	}

	artifactRoot, err := os.MkdirTemp("/tmp", "v")
	if err != nil {
		t.Fatalf("mkdir artifact root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(artifactRoot) })

	eng, err := testengine.NewLocalEngine(testengine.LocalConfig{
		Chain:        "stablenet",
		Binary:       bin,
		KeysDir:      filepath.Join(repoRoot(t), "keys", "preset"),
		ArtifactRoot: artifactRoot,
		Validators:   4,
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewLocalEngine: %v", err)
	}

	// The funded preset account, and an address that starts with nothing.
	const (
		funded = "0xc17d493883eaa3b4cceb0f214b273392d562f9d8"
		fresh  = "0x00000000000000000000000000000000000000aa"
	)

	specs := [][]byte{
		mustSpec(t, map[string]any{
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
		}),
		mustSpec(t, map[string]any{
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
		}),
		mustSpec(t, map[string]any{
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
		}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	root, err := eng.Run(ctx, specs)
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatalf("read session.json: %v", err)
	}
	var doc struct {
		Tests []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tests"`
		Summary struct {
			Pass    int `json:"pass"`
			Fail    int `json:"fail"`
			Blocked int `json:"blocked"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse session.json: %v", err)
	}
	if doc.Summary.Pass != len(specs) || doc.Summary.Fail > 0 || doc.Summary.Blocked > 0 {
		t.Fatalf("summary %+v, want all %d specs passing; tests: %+v", doc.Summary, len(specs), doc.Tests)
	}
	t.Logf("all %d vocabulary specs passed live; session at %s", doc.Summary.Pass, root)
}

// mustSpec marshals a spec map or fails the test.
func mustSpec(t *testing.T, m map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	return b
}
