package verb

import (
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlaceRequest_PlacesEveryReferenceOnce (W4c) covers the `chain up` entry,
// which reaches none of the test engine's path.
func TestPlaceRequest_PlacesEveryReferenceOnce(t *testing.T) {
	cfg := writeWorkspaceConfig(t, "/data", nil)
	declared := map[string]string{"upgrade": "gstable-next"}

	in := chainsetup.NetUpIn{
		DataDir: t.TempDir(), Chain: "stablenet", Stage: chainsetup.UpStart,
		Binary: "gstable", Binaries: declared, WorkspaceConfigPath: cfg,
	}
	if err := placeUpRequest(&in); err != nil {
		t.Fatal(err)
	}
	if want := "/data/bin/gstable"; in.Binary != want {
		t.Errorf("binary = %q, want %q", in.Binary, want)
	}
	if want := "/data/bin/gstable-next"; in.Binaries["upgrade"] != want {
		t.Errorf("binaries[upgrade] = %q, want %q", in.Binaries["upgrade"], want)
	}
	// The caller's map is not the request's to rewrite: a declaration may be
	// shared with whoever is still reading it.
	if declared["upgrade"] != "gstable-next" {
		t.Errorf("the declared map was rewritten: %v", declared)
	}
	// Idempotent, so a second placement on the way through is harmless and no
	// single site has to be the one that remembers.
	if err := placeUpRequest(&in); err != nil {
		t.Fatal(err)
	}
	if want := "/data/bin/gstable"; in.Binary != want {
		t.Errorf("placing twice moved it to %q", in.Binary)
	}
}

// TestPlaceRequest_WithoutAnEnvironmentFileANameStaysAName: the target resolves
// it on its PATH, and inventing a path here would name a file nobody put there.
func TestPlaceRequest_WithoutAnEnvironmentFileANameStaysAName(t *testing.T) {
	in := chainsetup.NetUpIn{Binary: "gstable", Binaries: map[string]string{"upgrade": "gstable-next"}}
	if err := chainsetup.PlaceRequest(&in, nil); err != nil {
		t.Fatal(err)
	}
	if in.Binary != "gstable" || in.Binaries["upgrade"] != "gstable-next" {
		t.Errorf("a name was placed without an environment file: %q %v", in.Binary, in.Binaries)
	}
}

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
