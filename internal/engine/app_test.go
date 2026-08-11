package engine_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// TestLocalEngine_RunRegistersNodeIdentities proves the wiring end to end
// without a chain binary: Run creates the session, materializes the key set, and
// registers each node identity before it ever tries to launch. The launch then
// fails (the binary does not exist) and the spec is recorded as blocked, which
// is exactly the sequencing under test — identity registration must not depend
// on a network coming up.
func TestLocalEngine_RunRegistersNodeIdentities(t *testing.T) {
	root := t.TempDir()
	e, err := engine.NewLocalEngine(engine.LocalConfig{
		Chain:        "stablenet",
		Binary:       filepath.Join(t.TempDir(), "no-such-binary"),
		Keys:         engine.PresetKeySource{Path: presetDir},
		ArtifactRoot: root,
		Validators:   4,
	})
	if err != nil {
		t.Fatalf("NewLocalEngine: %v", err)
	}

	sessionRoot, err := e.Run(context.Background(), [][]byte{[]byte(minimalSpec)})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for i := 1; i <= 4; i++ {
		addr := filepath.Join(sessionRoot, "keys", fmt.Sprintf("node%d", i), "address")
		b, readErr := os.ReadFile(addr)
		if readErr != nil {
			t.Fatalf("node%d identity was not registered in the session: %v", i, readErr)
		}
		if len(b) == 0 {
			t.Errorf("node%d address file is empty", i)
		}
	}
}

// TestLocalEngine_RunFailsOnUndersizedKeySet checks that a topology larger than
// the key set stops the run at composition time with a message naming the key
// set, rather than launching nodes that have no identity.
func TestLocalEngine_RunFailsOnUndersizedKeySet(t *testing.T) {
	e, err := engine.NewLocalEngine(engine.LocalConfig{
		Chain:        "stablenet",
		Binary:       "unused",
		Keys:         engine.PresetKeySource{Path: presetDir},
		ArtifactRoot: t.TempDir(),
		Validators:   999,
	})
	if err != nil {
		t.Fatalf("NewLocalEngine: %v", err)
	}
	_, err = e.Run(context.Background(), [][]byte{[]byte(minimalSpec)})
	if err == nil {
		t.Fatal("want an error when the topology exceeds the key set")
	}
	if !strings.Contains(err.Error(), "node identities") {
		t.Errorf("error should explain the key set is too small, got: %v", err)
	}
}

// minimalSpec is a valid spec whose assertions never run here: the environment
// build fails first. It exists so Run reaches session creation.
const minimalSpec = `{
  "schemaVersion": "1",
  "id": "identity-wiring",
  "chain": {"name": "stablenet", "binary": "gstable"},
  "assertions": [{"assert": "chainId", "expected": 8283}]
}`
