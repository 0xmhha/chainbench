package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// These hold the one decision every run starts with. It was written three times
// — the CLI's switch, the MCP tool's, and AttachRun's own branch — and the
// three did not agree, so what is checked here is that there is now one answer
// and that both surfaces get it in their own words.

func attaching(t *testing.T, rpc []string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "case", "id": "a",
		"env": map[string]any{
			"schemaVersion": "2", "kind": "env", "id": "e", "chain": "wbft",
			"attach": map[string]any{"rpc": rpc, "provides": []string{"short-expiry"}},
		},
		"steps": []any{map[string]any{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestStartForResolvesTheFourWays(t *testing.T) {
	for _, c := range []struct {
		name string
		in   Named
		want lifecycle.Status
	}{
		{"a workspace to compose in",
			Named{WorkspaceDir: "/w"}, lifecycle.ChainOpenWorkspace},
		{"a workspace whose network is up",
			Named{Attach: true, WorkspaceDir: "/w"}, lifecycle.AdoptChainByWorkspace},
		{"endpoints somebody typed",
			Named{RPCURLs: []string{"http://x"}}, lifecycle.AdoptChainByRPC},
		{"a network the specs declare",
			Named{Specs: func() ([][]byte, []string, error) {
				return [][]byte{attaching(t, []string{"http://x"})}, []string{"a.json"}, nil
			}}, lifecycle.AdoptChainByDeclaration},
	} {
		c.in.Spell = CLISpelling
		got, err := StartFor(c.in)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got.At != c.want {
			t.Errorf("%s: %s, want %s", c.name, got.At, c.want)
		}
		if (got.Declared != nil) != (c.want == lifecycle.AdoptChainByDeclaration) {
			t.Errorf("%s: the declaration is carried only by the state that came from one", c.name)
		}
		if got.Adopts() != (c.want != lifecycle.ChainOpenWorkspace) {
			t.Errorf("%s: Adopts says %v", c.name, got.Adopts())
		}
	}
}

// TestStartForRefusesBeforeReadingAnything is why the specs are behind a
// function. A run that combined two flags used to report a missing file: the
// specs were read first, and the mistake was in what the caller typed.
func TestStartForRefusesBeforeReadingAnything(t *testing.T) {
	read := false
	specs := func() ([][]byte, []string, error) {
		read = true
		return nil, nil, nil
	}
	for _, c := range []struct {
		name string
		in   Named
		says string
	}{
		{"attach with endpoints", Named{Attach: true, RPCURLs: []string{"http://x"}},
			"does not combine with --rpc"},
		{"attach with no workspace", Named{Attach: true},
			"--attach needs --workspace-dir"},
		{"endpoints and a workspace", Named{RPCURLs: []string{"http://x"}, WorkspaceDir: "/w"},
			"does not combine with --rpc"},
	} {
		c.in.Spell, c.in.Specs = CLISpelling, specs
		_, err := StartFor(c.in)
		if err == nil {
			t.Errorf("%s was accepted", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if read {
		t.Error("the specs were read to refuse something the caller typed")
	}
}

// TestARefusalNamesWhatTheCallerTyped is why Spelling exists: the same rule,
// two surfaces, and a message that names the flag or the key the operator
// actually used.
func TestARefusalNamesWhatTheCallerTyped(t *testing.T) {
	for _, c := range []struct {
		spell Spelling
		says  string
	}{
		{CLISpelling, "--attach takes the endpoints from --workspace-dir; it does not combine with --rpc"},
		{ToolSpelling, "attach takes the endpoints from dataDir; it does not combine with rpc"},
	} {
		_, err := StartFor(Named{Spell: c.spell, Attach: true, RPCURLs: []string{"http://x"}})
		if err == nil || err.Error() != c.says {
			t.Errorf("got %v, want %q", err, c.says)
		}
	}
}

// TestNoNetworkNamedSaysAllFourWays: the one refusal an operator hits when they
// have said nothing at all has to list what they could say.
func TestNoNetworkNamedSaysAllFourWays(t *testing.T) {
	_, err := StartFor(Named{Spell: CLISpelling, Specs: func() ([][]byte, []string, error) {
		return nil, nil, nil
	}})
	if err == nil {
		t.Fatal("a run that named no network was accepted")
	}
	for _, want := range []string{"--workspace-dir", "--attach", "--rpc", "env.attach"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not mention %s: %v", want, err)
		}
	}
}
