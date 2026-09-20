package mcp_test

import (
	"strings"
	"testing"
)

func TestConsensusStatusTool(t *testing.T) {
	srv := rpcMock(map[string]any{
		"eth_blockNumber":        "0x64", // 100
		"eth_chainId":            "0x205b",
		"net_peerCount":          "0x3",
		"eth_syncing":            false,
		"istanbul_getValidators": []any{"0xa", "0xb", "0xc", "0xd"},
	})
	defer srv.Close()
	text, isErr := callText(t, newServer(), "chainbench_consensus_status", map[string]any{
		"chain": "stablenet", "rpc": srv.URL,
	})
	if isErr || !strings.Contains(text, "head=100") || !strings.Contains(text, "peers=3") ||
		!strings.Contains(text, "syncing=false") || !strings.Contains(text, "validators=4") {
		t.Errorf("consensus_status: err=%v text=%s", isErr, text)
	}
}

func TestConsensusHealthTool(t *testing.T) {
	healthy := rpcMock(map[string]any{"eth_blockNumber": "0x5", "eth_syncing": false, "net_peerCount": "0x2"})
	defer healthy.Close()
	text, _ := callText(t, newServer(), "chainbench_consensus_health", map[string]any{"rpc": healthy.URL})
	if !strings.HasPrefix(text, "healthy") {
		t.Errorf("expected healthy, got %s", text)
	}

	// syncing endpoint is unhealthy.
	stuck := rpcMock(map[string]any{"eth_blockNumber": "0x0", "eth_syncing": true})
	defer stuck.Close()
	text, _ = callText(t, newServer(), "chainbench_consensus_health", map[string]any{"rpc": stuck.URL})
	if !strings.HasPrefix(text, "unhealthy") {
		t.Errorf("expected unhealthy, got %s", text)
	}
}

func TestConsensusBlockInfoTool(t *testing.T) {
	srv := rpcMock(map[string]any{
		"eth_getBlockByNumber": map[string]any{
			"number":       "0x64",
			"hash":         "0xabc",
			"miner":        "0xf00",
			"timestamp":    "0x61",
			"gasUsed":      "0x5208",
			"transactions": []any{"0x1", "0x2"},
		},
	})
	defer srv.Close()
	// decimal block tag is normalized to hex before the call.
	text, isErr := callText(t, newServer(), "chainbench_consensus_block_info", map[string]any{
		"rpc": srv.URL, "block": "100",
	})
	if isErr || !strings.Contains(text, "number=100") || !strings.Contains(text, "miner=0xf00") ||
		!strings.Contains(text, "txs=2") || !strings.Contains(text, "gas_used=21000") {
		t.Errorf("block_info: err=%v text=%s", isErr, text)
	}
}
