package engine_test

import (
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/engine"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin
)

func TestNewLocalEngine_Validation(t *testing.T) {
	if _, err := engine.NewLocalEngine(engine.LocalConfig{}); err == nil {
		t.Fatal("expected error for empty config")
	}
	_, err := engine.NewLocalEngine(engine.LocalConfig{
		Chain: "does-not-exist", Binary: "b", KeysDir: "k", ArtifactRoot: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for unknown chain")
	}
}

func TestNewLocalEngine_Valid(t *testing.T) {
	e, err := engine.NewLocalEngine(engine.LocalConfig{
		Chain:        "stablenet",
		Binary:       "/nonexistent/gstable",
		KeysDir:      filepath.Join(repoRoot(t), "keys", "preset"),
		ArtifactRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewLocalEngine: %v", err)
	}
	if e == nil {
		t.Fatal("engine is nil")
	}
}
