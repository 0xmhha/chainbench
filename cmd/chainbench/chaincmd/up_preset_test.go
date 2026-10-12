package chaincmd_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// TestChainUp_ComposesFromAChainPreset: --chain-preset composes the network a
// chain-preset document declares, including what no other flag can say, such
// as a launch option scoped to one role.
func TestChainUp_ComposesFromAChainPreset(t *testing.T) {
	dir := t.TempDir()
	keys, err := filepath.Abs(filepath.Join("..", "..", "..", "presets", "keys"))
	if err != nil {
		t.Fatal(err)
	}
	preset := filepath.Join(t.TempDir(), "preset.json")
	if err = os.WriteFile(preset, []byte(`{"schemaVersion":"2","kind":"chain-preset","id":"scoped","chain":"stablenet",
"topology":{"bp":2,"en":1},"launch":{"en":{"nodiscover":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "chain", "up", "--workspace-dir", dir, "--stage", "deploy", "--keys", keys, "--chain-preset", preset)
	if err != nil {
		t.Fatalf("chain up: %v\n%s", err, out)
	}
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]int{}
	for _, n := range ws.State().Nodes {
		roles[n.Role]++
		scoped := slices.ContainsFunc(n.Args, func(a string) bool { return a == "--nodiscover" })
		if scoped != (n.Role == "en") {
			t.Errorf("node%d (%s) args %v: the en-scoped launch option landed on the wrong node", n.Index, n.Role, n.Args)
		}
	}
	if roles["bp"] != 2 || roles["en"] != 1 {
		t.Fatalf("roles = %v, want the preset's 2 bp and 1 en", roles)
	}
}

// TestChainUp_PresetAndLayoutFlagsDoNotMix: a preset declares the layout, so
// a layout flag beside it would be a second answer to the same question.
func TestChainUp_PresetAndLayoutFlagsDoNotMix(t *testing.T) {
	preset := filepath.Join(t.TempDir(), "preset.json")
	if err := os.WriteFile(preset, []byte(`{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":2}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, "chain", "up", "--workspace-dir", t.TempDir(), "--stage", "deploy", "--chain-preset", preset, "--bp", "3")
	if err == nil || !strings.Contains(err.Error(), "--bp") {
		t.Fatalf("a preset with --bp gave %v, want a refusal naming --bp", err)
	}
}
