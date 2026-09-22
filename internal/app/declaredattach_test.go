package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

// writeCase writes a v2 case with the given env object and returns its path.
func writeCase(t *testing.T, dir, id string, env map[string]any) string {
	t.Helper()
	doc := map[string]any{
		"schemaVersion": "2", "kind": "case", "id": id, "chainPreset": env,
		"steps": []map[string]any{{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "0"}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id+".json")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func attachEnv(rpc ...string) map[string]any {
	return map[string]any{
		"schemaVersion": "2", "kind": "chain-preset", "id": "e", "chain": "stablenet",
		"attach": map[string]any{"rpc": rpc},
	}
}

func composeEnv() map[string]any {
	return map[string]any{
		"schemaVersion": "2", "kind": "chain-preset", "id": "e", "chain": "stablenet",
		"topology": map[string]any{"bp": 4},
	}
}

// TestDeclaredAttach_OneRunIsOneNetwork is the agreement a run rests on. The
// specs of one run are answered by one network, so specs that name different
// ones — or a mix of attaching and composing — cannot be run together: whichever
// was taken, the others would report on a network they never named.
func TestDeclaredAttach_OneRunIsOneNetwork(t *testing.T) {
	dir := t.TempDir()
	a := writeCase(t, dir, "a", attachEnv("http://127.0.0.1:8600"))
	b := writeCase(t, dir, "b", attachEnv("http://127.0.0.1:8600"))
	c := writeCase(t, dir, "c", attachEnv("http://127.0.0.1:9999"))
	d := writeCase(t, dir, "d", composeEnv())

	got, err := app.DeclaredAttach([]string{a, b})
	if err != nil {
		t.Fatalf("two specs naming one network: %v", err)
	}
	if got == nil || len(got.RPCURLs) != 1 || got.RPCURLs[0] != "http://127.0.0.1:8600" {
		t.Fatalf("attach = %+v, want the declared endpoint", got)
	}
	if got.Chain != "stablenet" {
		t.Errorf("chain = %q, want stablenet", got.Chain)
	}

	if _, err := app.DeclaredAttach([]string{a, c}); err == nil {
		t.Error("two specs attaching to different networks were accepted")
	} else if !strings.Contains(err.Error(), "different networks") {
		t.Errorf("the refusal does not say what disagrees: %v", err)
	}
	if _, err := app.DeclaredAttach([]string{a, d}); err == nil {
		t.Error("an attaching spec and a composing one were accepted together")
	}
	if _, err := app.DeclaredAttach([]string{d, a}); err == nil {
		t.Error("a composing spec and an attaching one were accepted together")
	}
}

// TestDeclaredAttach_ComposingSpecsDeclareNothing keeps the ordinary path
// unchanged: a run of composing specs must answer nil, or every composed run
// would be diverted to attach.
func TestDeclaredAttach_ComposingSpecsDeclareNothing(t *testing.T) {
	dir := t.TempDir()
	a := writeCase(t, dir, "a", composeEnv())
	b := writeCase(t, dir, "b", composeEnv())
	got, err := app.DeclaredAttach([]string{a, b})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("attach = %+v, want nil", got)
	}
}
