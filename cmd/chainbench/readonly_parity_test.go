package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/mcp"
)

// cliToMCP maps a CLI path to the MCP tool that is the same feature.
//
// It is written out rather than derived, because the two surfaces name things
// differently on purpose: `chain status` is chainbench_chain_status, but the
// top-level `status` is chainbench_status and `log` is chainbench_log. A
// generated mapping would have to encode those exceptions anyway, and encoding
// them here at least puts them where a reader looking for the pairing finds
// them.
//
// Only pairs that exist on both surfaces belong here. A feature one surface
// does not offer is not a disagreement.
var cliToMCP = map[string]string{
	"account state":    "chainbench_account_state",
	"chain health":     "chainbench_chain_health",
	"chain logs":       "chainbench_chain_logs",
	"chain show":       "chainbench_chain_show",
	"chain status":     "chainbench_chain_status",
	"chains":           "chainbench_chains",
	"consensus":        "chainbench_consensus",
	"contract call":    "chainbench_contract_call",
	"keyring list":     "chainbench_keyring_list",
	"keyring show":     "chainbench_keyring_show",
	"log":              "chainbench_log",
	"report":           "chainbench_report",
	"resource plan":    "chainbench_resource_plan",
	"resource pool":    "chainbench_resource_pool",
	"status":           "chainbench_status",
	"tx wait":          "chainbench_tx_wait",
	"verify":           "chainbench_verify",
	"validator roster": "",
}

// TestReadOnly_TheTwoSurfacesAgree: a feature that is safe on one surface must
// be safe on the other.
//
// The property is declared twice because the surfaces have no shared registry
// yet — the CLI reads a cobra annotation and MCP reads a struct field. Two
// declarations of one fact is exactly the shape that drifts, so the pairing is
// held here: mark a tool read-only and leave its command unmarked and this
// fails, and the reverse fails too.
func TestReadOnly_TheTwoSurfacesAgree(t *testing.T) {
	cli := map[string]bool{}
	for _, p := range ReadOnlyPaths(newRootCmd()) {
		if !strings.HasPrefix(p, "query ") {
			cli[p] = true
		}
	}
	tools := map[string]bool{}
	for _, n := range mcp.Default("test", "0").ReadOnlyTools() {
		tools[n] = true
	}
	if len(cli) == 0 || len(tools) == 0 {
		t.Fatal("one of the surfaces declared nothing, so this test asserts nothing")
	}

	var disagree []string
	for path, tool := range cliToMCP {
		if tool == "" {
			continue // no MCP counterpart
		}
		if cli[path] != tools[tool] {
			disagree = append(disagree, path+" is "+yesNo(cli[path])+" on the CLI but "+
				tool+" is "+yesNo(tools[tool])+" on MCP")
		}
	}
	sort.Strings(disagree)
	for _, d := range disagree {
		t.Errorf("the surfaces disagree about what is safe: %s", d)
	}
	t.Logf("%d paired features, %d read-only commands, %d read-only tools", len(cliToMCP), len(cli), len(tools))
}

func yesNo(b bool) string {
	if b {
		return "read-only"
	}
	return "not read-only"
}
