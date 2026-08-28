package chainsetup_test

import (
	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"os"
	"path/filepath"
	"testing"
)

// repoRoot walks up from the test's working directory to the module root (the
// directory holding go.mod), so the shipped preset can be read wherever the
// test runs.
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
			t.Fatal("go.mod not found above the test directory")
		}
		dir = parent
	}
}

// wbftTestPlugin is a StaticPlugin with the real wbft family and a minimal
// template exercising the consensus-critical placeholders.
func wbftTestPlugin() registry.ChainPlugin {
	tmpl := `{"config":{"chainId":__CHAIN_ID__},` +
		`"validators":"__VALIDATORS_JSON__",` +
		`"blsKeys":"__BLS_PUBLIC_KEYS_JSON__",` +
		`"extraData":"__EXTRA_DATA__",` +
		`"alloc":__ALLOC_JSON__}`
	return registry.StaticPlugin{
		M:    registry.Manifest{ID: "wbfttest", Binary: "go-wbft", ChainID: 1337, ConsensusFamily: "wbft"},
		Fam:  wbftfam.New(),
		Tmpl: []byte(tmpl),
	}
}
