package main

import (
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chains/stablenet/govbind"
)

// TestStablenetGovernance_DSL_E2E verifies that a stablenet governance scenario
// runs through the redesign DSL engine with no core changes: it builds the
// real proposals(uint256) calldata via govbind, serves a canned ABI-encoded
// status from a mock node, and runs a spec whose `call` assertion reads it —
// proving the DSL expresses and the engine executes stablenet ACL reads.
func TestStablenetGovernance_DSL_E2E(t *testing.T) {
	data := govbind.ProposalsCall(big.NewInt(1))

	// A proposals() return is 10 static words; the status is the last byte of
	// word 9. Encode status = 2 (approved).
	ret := "0x" + strings.Repeat("00", 319) + "02"
	if status, ok := govbind.DecodeProposalStatus(ret); !ok || status != 2 {
		t.Fatalf("canned return is not a valid status: status=%d ok=%v", status, ok)
	}

	srv := mockRPCNode(t, map[string]any{
		"eth_chainId": "0x205b", // 8283
		"eth_call":    ret,
	})
	spec := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "gov-proposal-status",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": 8283},
			{"assert": "call", "to": "0x0000000000000000000000000000000000000100", "data": data, "expected": ret},
		},
	})

	out, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), spec)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "pass=1") {
		t.Fatalf("expected the governance spec to pass:\n%s", out)
	}
}
