package testengine_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"

	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestAttachEngine_EmitsChainstate proves the collector is wired into the engine:
// an attach run against a mock RPC node publishes a chainstate snapshot to the
// bus, so the dashboard sees the live network state. No chain binary is needed.
func TestAttachEngine_EmitsChainstate(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_chainId":     "0x539", // 1337
		"eth_blockNumber": "0x10",  // 16
		"net_peerCount":   "0x3",
		"eth_getBlockByNumber": map[string]any{
			"number": "0x10", "hash": "0xhead", "miner": "0xA",
		},
	})

	bus := collector.NewBus()
	sub := bus.Subscribe()

	eng, err := testengine.NewAttachEngine(testengine.AttachConfig{
		Chain:        "stablenet",
		RPCURLs:      []string{srv.URL},
		ArtifactRoot: t.TempDir(),
		Bus:          bus,
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewAttachEngine: %v", err)
	}

	spec, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "collect-smoke",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 1337}},
	})
	if _, err := eng.Run(context.Background(), [][]byte{spec}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The teardown publishes a final chainstate snapshot before Run returns;
	// drain the bus and require one reflecting the sampled height.
	bus.Close()
	var sawHeight bool
	for e := range sub {
		if e.Message == "chainstate" {
			if h, ok := e.Fields["heights"].(map[string]uint64); ok && h["node1"] == 16 {
				sawHeight = true
			}
		}
	}
	if !sawHeight {
		t.Fatal("expected a chainstate event with node1 height 16")
	}
}
