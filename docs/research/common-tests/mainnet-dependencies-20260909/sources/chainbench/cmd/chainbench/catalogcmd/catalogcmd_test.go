package catalogcmd_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/catalogcmd"
	"github.com/0xmhha/chainbench/internal/mcp"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(catalogcmd.NewChains(), catalogcmd.NewCapabilities())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

func runMCP(t *testing.T, tool string, args map[string]any) string {
	t.Helper()
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := mcp.Default("chainbench", "test").Handle(context.Background(), req)
	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("%s: bad response: %v (%s)", tool, err, raw)
	}
	if len(resp.Result.Content) == 0 || resp.Result.IsError {
		t.Fatalf("%s failed: %s", tool, raw)
	}
	return resp.Result.Content[0].Text
}

// TestParity_Chains: both surfaces answer "which chains does this bench know"
// from the same registry, so they must name the same chains.
//
// The renderings differ and are allowed to: the CLI writes a column per chain
// for a person to read, MCP writes a line for a program. What has to match is
// the set of names, which is the answer.
func TestParity_Chains(t *testing.T) {
	cli, err := runCLI(t, "chains")
	if err != nil {
		t.Fatalf("CLI: %v\n%s", err, cli)
	}
	mcpOut := runMCP(t, "chainbench_chains", map[string]any{})

	known := []string{"stablenet", "wemix", "wbft"}
	var seen int
	for _, name := range known {
		inCLI, inMCP := strings.Contains(cli, name), strings.Contains(mcpOut, name)
		if inCLI != inMCP {
			t.Errorf("the surfaces disagree about %q: CLI=%v MCP=%v\n  CLI: %s\n  MCP: %s",
				name, inCLI, inMCP, cli, mcpOut)
		}
		if inCLI {
			seen++
		}
	}
	if seen == 0 {
		t.Fatalf("neither surface named a chain, so agreeing proves nothing.\n  CLI: %s\n  MCP: %s", cli, mcpOut)
	}
}

// TestCapabilities_ListsSomething: the catalog is read from the registry, so an
// empty answer means the plugins were never registered rather than that the
// bench supports nothing.
func TestCapabilities_ListsSomething(t *testing.T) {
	out, err := runCLI(t, "capabilities")
	if err != nil {
		t.Fatalf("capabilities: %v\n%s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("capabilities printed nothing; the chain plugins are not registered")
	}
}

// TestCapabilitiesCall_RefusesAnUnknownName: a typo has to be told apart from a
// capability that exists but is unsupported on this chain.
func TestCapabilitiesCall_RefusesAnUnknownName(t *testing.T) {
	out, err := runCLI(t, "capabilities", "call", "v1.stablenet.no-such-capability")
	if err == nil {
		t.Fatalf("an unknown capability was accepted:\n%s", out)
	}
}
