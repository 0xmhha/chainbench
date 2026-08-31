package genesis_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/registry"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin
)

// TestPresetGenesisSource_Live_GstableInit proves the genesis built by
// PresetGenesisSource from the committed stablenet preset is accepted by a real
// gstable binary: `gstable init` must succeed and populate the datadir.
//
// It is gated on GSTABLE_BIN (path to a real gstable). Without it the test
// skips, so CI (which has no chain binary) stays green while a developer with
// the binary can verify the deferred assumption for real.
func TestPresetGenesisSource_Live_GstableInit(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the live genesis-init check")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GSTABLE_BIN=%q: %v", bin, err)
	}

	presetDir := filepath.Join(repoRoot(t), "keys", "preset")
	if _, err := os.Stat(filepath.Join(presetDir, "metadata.json")); err != nil {
		t.Skipf("preset not found at %s: %v", presetDir, err)
	}

	plugin, err := registry.Get("stablenet")
	if err != nil {
		t.Fatalf("registry.Get(stablenet): %v", err)
	}

	gen, err := genesis.PresetSource{KeysDir: presetDir}.Genesis(context.Background(), plugin, genesis.Request{Validators: 4})
	if err != nil {
		t.Fatalf("build genesis: %v", err)
	}

	dataDir := t.TempDir()
	genesisPath := filepath.Join(dataDir, "genesis.json")
	if err := os.WriteFile(genesisPath, gen.Genesis, 0o644); err != nil {
		t.Fatalf("write genesis: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := driver.InitDatadir(ctx, bin, dataDir, genesisPath); err != nil {
		t.Fatalf("gstable init rejected the genesis: %v", err)
	}

	// A successful init writes a chaindata store under a client-named subdir
	// (e.g. <datadir>/gstable/chaindata), so look for it anywhere under the dir.
	if !hasChaindata(t, dataDir) {
		t.Fatal("init did not populate a chaindata store under the datadir")
	}
}

// hasChaindata reports whether a "chaindata" directory exists anywhere under
// root, the marker of a successful genesis init.
func hasChaindata(t *testing.T, root string) bool {
	t.Helper()
	found := false
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "chaindata" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk datadir: %v", err)
	}
	return found
}

// repoRoot walks up from the test's working directory to the module root (the
// directory holding go.mod).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from test dir")
		}
		dir = parent
	}
}
