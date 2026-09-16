package dsl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// envFile writes one chain declaration and returns the directory it is in, laid
// out the way tests/tc is: an env/ directory the cases sit beside.
func envFile(t *testing.T, id, chain string, bp int) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{
		"schemaVersion": "2", "kind": "env", "id": id, "chain": chain,
		"topology": map[string]any{"bp": bp},
	}
	b, _ := json.Marshal(doc)
	if err := os.WriteFile(filepath.Join(dir, "env", id+".env.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// caseFile writes a case with the given env value (raw JSON) and returns its path.
func caseFile(t *testing.T, dir, id, env string) string {
	t.Helper()
	raw := `{"schemaVersion":"2","kind":"case","id":"` + id + `","env":` + env + `,
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	p := filepath.Join(dir, id+".json")
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// specOf reads one spec the way every surface does and parses it.
func specOf(t *testing.T, path, envRef string) dsl.Spec {
	t.Helper()
	raws, err := dsl.ReadFilesWithEnv([]string{path}, envRef)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s, err := dsl.Parse(raws[0])
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return s
}

// TestReadFilesWithEnv_MovesACaseOntoAnotherChain is the point of the whole
// thing: the steps say nothing about which mainnet they run on, so choosing a
// different declaration is all it takes to meet a different chain.
func TestReadFilesWithEnv_MovesACaseOntoAnotherChain(t *testing.T) {
	dir := envFile(t, "a-bp4", "stablenet", 4)
	writeEnv(t, dir, "b-bp7", "wbft", 7)
	p := caseFile(t, dir, "c", `"a-bp4"`)

	if got := specOf(t, p, "").Chain.Name; got != "stablenet" {
		t.Fatalf("as written the case runs on %q", got)
	}
	moved := specOf(t, p, "b-bp7")
	if moved.Chain.Name != "wbft" {
		t.Fatalf("moved case runs on %q, want wbft", moved.Chain.Name)
	}
	if bpOf(moved) != 7 {
		t.Errorf("the new declaration's layout must come with it: %v", moved.Topology)
	}
}

// TestReadFilesWithEnv_KeepsWhatTheCaseOverrode: the override belongs to the
// test, not to the chain. A case that needs a fork on gets it on whichever
// chain it lands.
func TestReadFilesWithEnv_KeepsWhatTheCaseOverrode(t *testing.T) {
	dir := envFile(t, "a-bp4", "stablenet", 4)
	writeEnv(t, dir, "b-bp7", "wbft", 7)
	p := caseFile(t, dir, "c", `{"extends":"a-bp4","capabilities":["ws"],
	  "genesis":{"overlay":{"config":{"applepieBlock":0}}}}`)

	moved := specOf(t, p, "b-bp7")
	if moved.Chain.Name != "wbft" || bpOf(moved) != 7 {
		t.Fatalf("base not swapped: chain %q topology %v", moved.Chain.Name, moved.Topology)
	}
	if len(moved.Chain.GenesisOverlay) == 0 {
		t.Error("the case's genesis overlay must survive the swap")
	}
	var hasWS bool
	for _, r := range moved.Requires {
		hasWS = hasWS || r == "ws"
	}
	if !hasWS {
		t.Errorf("the case's capabilities must survive the swap: requires = %v", moved.Requires)
	}
}

// TestReadFilesWithEnv_RefusesACaseThatDeclaresItsEnvInline: an inline env IS
// the case's declaration. Swapping it would throw away what the case asked for
// with no way to tell which parts mattered, so it is refused by name.
func TestReadFilesWithEnv_RefusesACaseThatDeclaresItsEnvInline(t *testing.T) {
	dir := envFile(t, "b-bp7", "wbft", 7)
	p := caseFile(t, dir, "inline-case", `{"schemaVersion":"2","kind":"env","id":"x",
	  "chain":"stablenet","topology":{"bp":4}}`)

	_, err := dsl.ReadFilesWithEnv([]string{p}, "b-bp7")
	if err == nil {
		t.Fatal("a case with an inline env must be refused, not silently rewritten")
	}
	for _, want := range []string{"inline-case", "extends"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal must say %q: %v", want, err)
		}
	}
}

// TestReadFilesWithEnv_TakesAPath is how a declaration for a chain with no home
// in this tree is run against: the value is read as a file when it looks like
// one, decided on the spelling alone so the same input always means the same
// thing.
func TestReadFilesWithEnv_TakesAPath(t *testing.T) {
	dir := envFile(t, "a-bp4", "stablenet", 4)
	p := caseFile(t, dir, "c", `"a-bp4"`)

	outside := filepath.Join(t.TempDir(), "newchain.json")
	b, _ := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "env", "id": "newchain", "chain": "wemix",
		"topology": map[string]any{"bp": 5},
	})
	if err := os.WriteFile(outside, b, 0o644); err != nil {
		t.Fatal(err)
	}

	moved := specOf(t, p, outside)
	if moved.Chain.Name != "wemix" || bpOf(moved) != 5 {
		t.Fatalf("path form did not take: chain %q topology %v", moved.Chain.Name, moved.Topology)
	}
}

// TestReadFilesWithEnv_SaysWhereItLooked: an id that resolves to nothing is a
// typo, and the message has to say what was searched for or the author guesses.
func TestReadFilesWithEnv_SaysWhereItLooked(t *testing.T) {
	dir := envFile(t, "a-bp4", "stablenet", 4)
	p := caseFile(t, dir, "c", `"a-bp4"`)

	_, err := dsl.ReadFilesWithEnv([]string{p}, "nonesuch")
	if err == nil {
		t.Fatal("an unknown env id must be refused")
	}
	if !strings.Contains(err.Error(), "nonesuch.env.json") {
		t.Errorf("message must name the file it looked for: %v", err)
	}
}

// TestReadFiles_IsReadFilesWithNoEnv guards the delegation: every surface still
// reads a spec as written when no env is named.
func TestReadFiles_IsReadFilesWithNoEnv(t *testing.T) {
	dir := envFile(t, "a-bp4", "stablenet", 4)
	p := caseFile(t, dir, "c", `"a-bp4"`)

	a, err := dsl.ReadFiles([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	b, err := dsl.ReadFilesWithEnv([]string{p}, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(a[0]) != string(b[0]) {
		t.Errorf("ReadFiles and an empty env must read the same bytes")
	}
}

// bpOf reads the bp count off a lowered spec. The topology is a decoded JSON
// object, so its numbers are float64 and comparing them to an int silently
// fails — which is how the first version of these tests passed nothing.
func bpOf(s dsl.Spec) int {
	n, _ := s.Topology["bp"].(float64)
	return int(n)
}

// writeEnv adds another declaration to a directory prepared by envFile.
func writeEnv(t *testing.T, dir, id, chain string, bp int) {
	t.Helper()
	b, _ := json.Marshal(map[string]any{
		"schemaVersion": "2", "kind": "env", "id": id, "chain": chain,
		"topology": map[string]any{"bp": bp},
	})
	if err := os.WriteFile(filepath.Join(dir, "env", id+".env.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
