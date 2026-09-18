package mcp_test

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/mcp"
)

// TestReadOnly_ReachesTheToolList is rule 3 of surface-unification-design §4.4:
// an agent asks for the tools and learns which ones are safe to call while it
// is exploring.
//
// It is carried as MCP's own readOnlyHint rather than a chainbench-specific
// field. This server declares the 2024-11-05 protocol and the annotations block
// arrived in 2025-03-26, so a client that does not know the field ignores it
// and one that does gets the right answer — which beats a name only chainbench
// would understand.
func TestReadOnly_ReachesTheToolList(t *testing.T) {
	s := mcp.NewServer("test", "0")
	s.Register(mcp.Tool{Name: "look", ReadOnly: true, Handler: nil})
	s.Register(mcp.Tool{Name: "change", Handler: nil})

	raw := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	var resp struct {
		Result struct {
			Tools []struct {
				Name        string         `json:"name"`
				Annotations map[string]any `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode tools/list: %v\n%s", err, raw)
	}
	if len(resp.Result.Tools) != 2 {
		t.Fatalf("got %d tools, want 2", len(resp.Result.Tools))
	}
	for _, tool := range resp.Result.Tools {
		hint, has := tool.Annotations["readOnlyHint"]
		switch tool.Name {
		case "look":
			if !has || hint != true {
				t.Errorf("the read-only tool carries annotations %v, want readOnlyHint true", tool.Annotations)
			}
		case "change":
			if has {
				t.Errorf("a tool that writes carries %v — an agent would call it while exploring", tool.Annotations)
			}
		}
	}
}

// TestReadOnlyTools_ReadsTheSameDeclaration keeps the list and the hint from
// coming apart: both are the tool's own field, so a caller that asks the server
// and a client that reads tools/list cannot get different answers.
func TestReadOnlyTools_ReadsTheSameDeclaration(t *testing.T) {
	s := mcp.NewServer("test", "0")
	s.Register(mcp.Tool{Name: "a", ReadOnly: true})
	s.Register(mcp.Tool{Name: "b"})
	s.Register(mcp.Tool{Name: "c", ReadOnly: true})
	got := s.ReadOnlyTools()
	if strings.Join(got, ",") != "a,c" {
		t.Errorf("ReadOnlyTools = %v, want [a c] in registration order", got)
	}
}

// TestReadOnly_CoversNoToolThatWrites is the same sanity check the CLI applies
// to its declarations. The property cannot be proven from the code — the method
// chainbench_node_rpc calls is an argument — but a tool named for a verb that
// plainly writes is a mistake worth catching before an agent is told it is safe.
func TestReadOnly_CoversNoToolThatWrites(t *testing.T) {
	writes := []string{
		"_new", "_add", "_import", "_send", "_deploy", "_start", "_stop", "_rm",
		"_init", "_up", "_run", "_resume", "_restart", "_attach", "_detach",
		"_keys", "_genesis", "_config", "_build", "_place", "_faucet", "_rpc",
		"_hardfork", "_upgrade",
	}
	names := mcp.Default("test", "0").ReadOnlyTools()
	if len(names) == 0 {
		t.Fatal("no tool declares itself read-only, so this test asserts nothing")
	}
	sort.Strings(names)
	for _, n := range names {
		for _, w := range writes {
			if strings.HasSuffix(n, w) {
				t.Errorf("%q declares itself read-only, but %q is a verb that writes — check the declaration", n, w)
			}
		}
	}
	t.Logf("%d of the registered tools declare themselves read-only", len(names))
}
