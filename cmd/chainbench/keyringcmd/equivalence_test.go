package keyringcmd_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/mcp"
)

// U3 routed the keyring group through app, which is what makes these possible:
// both surfaces now call the same entry point, so a difference between them is
// a defect rather than the expected consequence of two implementations
// (worklist §1l).

// callMCP invokes a tool and returns its text content.
func callMCP(t *testing.T, tool string, args map[string]any) string {
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
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("%s: bad response: %v (%s)", tool, err, raw)
	}
	if resp.Error != nil {
		t.Fatalf("%s: %s", tool, resp.Error.Message)
	}
	if len(resp.Result.Content) == 0 {
		t.Fatalf("%s: no content in %s", tool, raw)
	}
	if resp.Result.IsError {
		t.Fatalf("%s failed: %s", tool, resp.Result.Content[0].Text)
	}
	return resp.Result.Content[0].Text
}

// jsonIn pulls the JSON document out of a surface's answer.
//
// The CLI announces which key set it read before printing it — an operator who
// did not name one needs to know which was chosen — so its answer is a line of
// prose followed by the document. That preamble is a rendering choice, not a
// difference in what was reported, so it is skipped rather than compared.
func jsonIn(t *testing.T, what, s string) any {
	t.Helper()
	i := strings.IndexAny(s, "[{")
	if i < 0 {
		t.Fatalf("%s answered with no JSON at all:\n%s", what, s)
	}
	var v any
	if err := json.Unmarshal([]byte(s[i:]), &v); err != nil {
		t.Fatalf("%s answer is not JSON: %v\n%s", what, err, s)
	}
	return v
}

// entriesOf takes the identities out of a whole-key-set record.
func entriesOf(t *testing.T, v any) any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	e, ok := m["entries"]
	if !ok {
		t.Fatalf("the record carries no identities: %+v", m)
	}
	return e
}

// size reports how many entries a document carries, so a comparison that
// succeeded on two empty answers can be caught.
func size(v any) int {
	switch t := v.(type) {
	case []any:
		return len(t)
	case map[string]any:
		return len(t)
	}
	return 0
}

// TestEquivalence_KeyringListNamesTheSameIdentities: one key set, two surfaces
// reading it, so the identities they report must match — labels, addresses and
// public keys alike.
//
// A key set is referenced by the material inside it: a genesis alloc, a running
// datadir, a test's declaration. Two surfaces disagreeing about what a set
// holds is therefore not a display bug, it is a network that comes up with the
// wrong validators.
func TestEquivalence_KeyringListNamesTheSameIdentities(t *testing.T) {
	dir := newRing(t, "--with-bls", "--validators", "2")

	cli, err := run(t, "keyring", "list", "--keyring-dir", dir, "--json")
	if err != nil {
		t.Fatalf("CLI keyring list: %v\n%s", err, cli)
	}
	mcpOut := callMCP(t, "chainbench_keyring_list", map[string]any{"keyringDir": dir})

	// The tool answers with the whole record — where the set is, how that was
	// chosen, and the identities — while the CLI puts the first two in its
	// preamble and prints the identities. Same information, laid out for
	// different readers, so the identities are what is compared.
	got, want := jsonIn(t, "CLI", cli), entriesOf(t, jsonIn(t, "MCP", mcpOut))
	if size(got) == 0 {
		t.Fatal("the CLI listed no identities, so agreeing about them proves nothing")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the two surfaces disagree about the key set.\n  CLI: %+v\n  MCP: %+v", got, want)
	}
}

// TestEquivalence_KeyringShowRevealsTheSameIdentity: the same for one entry,
// which is the read a caller uses when it needs an address to fund or a public
// key to place in a genesis.
func TestEquivalence_KeyringShowRevealsTheSameIdentity(t *testing.T) {
	dir := newRing(t, "--with-bls", "--validators", "2")

	cli, err := run(t, "keyring", "show", "--keyring-dir", dir, "--name", "node1", "--json")
	if err != nil {
		t.Fatalf("CLI keyring show: %v\n%s", err, cli)
	}
	mcpOut := callMCP(t, "chainbench_keyring_show", map[string]any{"keyringDir": dir, "name": "node1"})

	got, want := jsonIn(t, "CLI", cli), jsonIn(t, "MCP", mcpOut)
	if size(got) == 0 {
		t.Fatal("the CLI showed nothing, so agreeing about it proves nothing")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the two surfaces disagree about node1.\n  CLI: %+v\n  MCP: %+v", got, want)
	}
}

// TestEquivalence_ImportingTheSameKeyLandsTheSameIdentity: importing is where
// the two surfaces used to read the operator's request separately — the CLI
// built a key source from its flags while the module built one from an import
// request. Both readings had to agree about what "exactly one origin" meant,
// and nothing checked that they did.
func TestEquivalence_ImportingTheSameKeyLandsTheSameIdentity(t *testing.T) {
	const key = "0xeb47b675926a348755d89dfaca9ba5a2c02a192fd54e7e78475f15443ddf8c21" // betterleaks:allow — throwaway signing key for this test only; its derived address appears in no genesis, config or keystore
	cliDir := filepath.Join(t.TempDir(), "cli")
	mcpDir := filepath.Join(t.TempDir(), "mcp")

	if out, err := run(t, "keyring", "new", "--keyring-dir", cliDir, "--count", "1"); err != nil {
		t.Fatalf("cli ring: %v\n%s", err, out)
	}
	callMCP(t, "chainbench_keyring_new", map[string]any{"keyringDir": mcpDir, "count": float64(1)})

	cli, err := run(t, "keyring", "import", "--keyring-dir", cliDir,
		"--name", "imported", "--private-key", key, "--json")
	if err != nil {
		t.Fatalf("CLI keyring import: %v\n%s", err, cli)
	}
	mcpOut := callMCP(t, "chainbench_keyring_import", map[string]any{
		"keyringDir": mcpDir, "name": "imported", "privateKey": key,
	})

	// The same key must derive the same address on both sides. That is the
	// whole of what an import promises.
	cliAddr, mcpAddr := addressIn(t, "CLI", cli), addressIn(t, "MCP", mcpOut)
	if cliAddr == "" {
		t.Fatal("the CLI import reported no address, so agreeing about it proves nothing")
	}
	if !strings.EqualFold(cliAddr, mcpAddr) {
		t.Errorf("the same key landed as different addresses.\n  CLI: %s\n  MCP: %s", cliAddr, mcpAddr)
	}
}

// addressIn pulls the address out of whichever shape a surface answers with.
func addressIn(t *testing.T, what, s string) string {
	t.Helper()
	doc := jsonIn(t, what, s)
	v, ok := doc.(map[string]any)
	if !ok {
		if arr, isArr := doc.([]any); isArr && len(arr) > 0 {
			v, _ = arr[len(arr)-1].(map[string]any)
		}
	}
	if a, ok := v["address"].(string); ok {
		return a
	}
	if entries, ok := v["entries"].([]any); ok {
		for _, e := range entries {
			if m, ok := e.(map[string]any); ok {
				if m["label"] == "imported" {
					if a, ok := m["address"].(string); ok {
						return a
					}
				}
			}
		}
	}
	t.Fatalf("%s answer carries no address: %s", what, s)
	return ""
}
