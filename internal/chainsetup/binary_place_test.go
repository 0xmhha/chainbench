package chainsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeWorkspaceConfig puts an environment file on disk and returns its path.
func writeWorkspaceConfig(t *testing.T, dataRoot string, aliases map[string]string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("version: 1\ndataRoot: " + dataRoot + "\n")
	b.WriteString("paths:\n  binaries: bin\n  configs: configs\n  genesis: genesis\n")
	b.WriteString("  keystore: keystore\n  keyrings: keys\n  nodes: node\n  runtime: runtime\n  logs: logs\n")
	b.WriteString("control:\n  artifactRoot: " + dataRoot + "\ninputs:\n  mode: generated\nexecution:\n  chain: fresh\n")
	if len(aliases) > 0 {
		b.WriteString("binaryAliases:\n")
		for name, file := range aliases {
			b.WriteString("  " + name + ": " + file + "\n")
		}
	}
	p := filepath.Join(t.TempDir(), "workspace-config.yaml")
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestBinary_TheChainNamesItWhenTheDeclarationDoesNot is why 164 specs repeated
// one word and thirteen more repeated a different spelling of it.
//
// A definition that names the binary is stating a fact the chain already
// states. Nothing checked the two agreed, so gstable and go-stablenet both
// "worked" — each went to exec unread, and only one of them was ever right for
// a given machine. The manifest answers now, and a definition that stays quiet
// gets the chain's own name.
func TestBinary_TheChainNamesItWhenTheDeclarationDoesNot(t *testing.T) {
	w := &Workspace{state: State{Chain: "stablenet"}}
	got, err := w.binary("")
	if err != nil {
		t.Fatalf("binary: %v", err)
	}
	if got != "gstable" {
		t.Errorf("binary = %q, want the manifest's %q", got, "gstable")
	}
}

// TestBinary_MostSpecificWins pins the order an operator expects: what this
// command was given beats what the composition recorded, which beats what the
// chain calls it.
func TestBinary_MostSpecificWins(t *testing.T) {
	w := &Workspace{state: State{Chain: "stablenet", Binary: "/opt/recorded"}}
	if got, _ := w.binary("/opt/argument"); got != "/opt/argument" {
		t.Errorf("an argument must win: got %q", got)
	}
	if got, _ := w.binary(""); got != "/opt/recorded" {
		t.Errorf("the record must beat the manifest: got %q", got)
	}
	w.state.Binary = ""
	if got, _ := w.binary(""); got != "gstable" {
		t.Errorf("the manifest must answer last: got %q", got)
	}
}

// TestBinary_AnEnvironmentFilePlacesTheName covers the half that existed and
// was never reached: binaryAliases and paths.binaries say which file this
// environment calls a name and where the files live, and before this the
// mapping ran only on the upgrade path.
func TestBinary_AnEnvironmentFilePlacesTheName(t *testing.T) {
	root := t.TempDir()
	wcPath := writeWorkspaceConfig(t, root, map[string]string{"gstable": "gstable-2.1.0"})
	w := &Workspace{state: State{Chain: "stablenet", WorkspaceConfig: wcPath}}

	got, err := w.binary("")
	if err != nil {
		t.Fatalf("binary: %v", err)
	}
	if want := filepath.Join(root, "bin", "gstable-2.1.0"); got != want {
		t.Errorf("binary = %q, want %q", got, want)
	}

	// A name with no alias is still placed under the binaries directory: the
	// environment says where binaries live, and an alias only renames the file.
	w2 := &Workspace{state: State{Chain: "wbft", WorkspaceConfig: wcPath}}
	got, err = w2.binary("")
	if err != nil {
		t.Fatalf("binary: %v", err)
	}
	if want := filepath.Join(root, "bin", "gwbft"); got != want {
		t.Errorf("unaliased binary = %q, want %q", got, want)
	}
}

// TestBinary_AnAbsolutePathIsUsedAsGiven: an operator who passes a path has
// answered the question, and placing it again would be second-guessing an
// explicit choice.
func TestBinary_AnAbsolutePathIsUsedAsGiven(t *testing.T) {
	root := t.TempDir()
	w := &Workspace{state: State{Chain: "stablenet", WorkspaceConfig: writeWorkspaceConfig(t, root, nil)}}
	if got, err := w.binary("/somewhere/else/gstable"); err != nil || got != "/somewhere/else/gstable" {
		t.Errorf("binary = %q, %v; want the path as given", got, err)
	}
}

// TestBinary_ARelativePathIsRefused is the gap between the two forms. A name is
// the target's to resolve and a path is placed; "./build/gstable" is neither,
// and resolving it against whatever directory happens to be current when the
// launch runs is how a composition comes up on a binary nobody chose.
func TestBinary_ARelativePathIsRefused(t *testing.T) {
	w := &Workspace{state: State{Chain: "stablenet"}}
	_, err := w.binary("build/gstable")
	if err == nil {
		t.Fatal("a relative path must be refused")
	}
	for _, want := range []string{"PATH", "absolute", "workspace-config"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not offer %q", err, want)
		}
	}
}

// TestBinary_ANameStaysANameWithoutAnEnvironmentFile: with nothing saying where
// binaries live, the name is what the target resolves on its PATH — the same
// thing exec does, said where the pre-launch check can agree with it.
func TestBinary_ANameStaysANameWithoutAnEnvironmentFile(t *testing.T) {
	w := &Workspace{state: State{Chain: "wemix"}}
	if got, err := w.binary(""); err != nil || got != "gwemix" {
		t.Errorf("binary = %q, %v; want the bare name %q", got, err, "gwemix")
	}
}
