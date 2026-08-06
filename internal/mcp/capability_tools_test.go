package mcp_test

import (
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/mcp"
)

// serverWithCaps builds a Default server; features/all is imported above so the
// capability tools are registered.
func TestCapabilityToolsGenerated(t *testing.T) {
	s := mcp.Default("chainbench", "test")

	// The discovery tool + hierarchical per-capability tools are exposed.
	text, isErr := callText(t, s, "chainbench.capabilities", map[string]any{"chain": "stablenet"})
	if isErr {
		t.Fatalf("capabilities discovery errored: %s", text)
	}
	for _, want := range []string{
		"chainbench.v1.common.chains.list",
		"chainbench.v1.common.chains.info",
		"chainbench.v1.stablenet.governance.propose_mint",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("discovery missing %q:\n%s", want, text)
		}
	}

	// A common capability is callable.
	out, isErr := callText(t, s, "chainbench.v1.common.chains.list", map[string]any{})
	if isErr || !strings.Contains(out, "stablenet") {
		t.Errorf("chains.list: err=%v out=%s", isErr, out)
	}

	// A chain-specific capability produces calldata.
	cd, isErr := callText(t, s, "chainbench.v1.stablenet.governance.propose_mint", map[string]any{
		"beneficiary": "0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
		"amount":      "1000000000000000000",
		"timestamp":   "1700000000",
	})
	if isErr || !strings.HasPrefix(cd, "0x") {
		t.Errorf("propose_mint: err=%v out=%s", isErr, cd)
	}
}

// TestDiscoveryCoversAllBuiltinTools guards that every built-in flat tool
// (chainbench_*) is folded into the capability catalog, so discovery never
// silently omits a tool a future change adds.
func TestDiscoveryCoversAllBuiltinTools(t *testing.T) {
	s := mcp.Default("chainbench", "test")

	list := call(t, s, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	var flat []string
	for _, tt := range list["result"].(map[string]any)["tools"].([]any) {
		name := tt.(map[string]any)["name"].(string)
		if strings.HasPrefix(name, "chainbench_") {
			flat = append(flat, name)
		}
	}
	if len(flat) < 30 {
		t.Fatalf("expected >= 30 built-in tools, got %d", len(flat))
	}

	discovery, isErr := callText(t, s, "chainbench.capabilities", map[string]any{})
	if isErr {
		t.Fatalf("discovery errored: %s", discovery)
	}
	for _, name := range flat {
		if !strings.Contains(discovery, name) {
			t.Errorf("built-in tool %q is missing from capability discovery", name)
		}
	}
}

// TestCapabilitiesFilterByChain confirms a chain sees common + its own only.
func TestCapabilitiesFilterByChain(t *testing.T) {
	s := mcp.Default("chainbench", "test")
	// wbft has no chain-specific capability registered in this slice; it should
	// still see the common ones but not stablenet's.
	text, _ := callText(t, s, "chainbench.capabilities", map[string]any{"chain": "wbft"})
	if !strings.Contains(text, "chainbench.v1.common.chains.list") {
		t.Errorf("wbft should see common capabilities:\n%s", text)
	}
	if strings.Contains(text, "stablenet.governance") {
		t.Errorf("wbft must not see stablenet-specific capabilities:\n%s", text)
	}
}
