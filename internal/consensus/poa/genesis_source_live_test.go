package poa_test

import (
	"context"
	"encoding/json"
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/wemix" // register the wemix plugin
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// TestWemixGenesisSource_Live_GeneratesAndInits is the claim that matters: the
// config this source assembles is one the real gwemix accepts, and the genesis
// it writes is one a real datadir initializes from.
//
// The path it replaces produced a genesis that also initialized — and left the
// node on ethash with no wemix namespace — so "init succeeded" is not the
// assertion. What is asserted is that the generated file carries the alloc and
// the wemix governance section the substituted template never had.
//
// Gated on GWEMIX_BIN; CI has no chain binary and skips.
func TestWemixGenesisSource_Live_GeneratesAndInits(t *testing.T) {
	bin := os.Getenv("GWEMIX_BIN")
	if bin == "" {
		t.Skip("set GWEMIX_BIN to the go-wemix gwemix binary to run this")
	}
	plugin, err := registry.Get("wemix")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	dir := t.TempDir()

	src := poa.GenesisSource{
		KeysDir: filepath.Join(repoRoot(t), "keys", "preset"),
		Binary:  bin,
		WorkDir: dir,
	}
	art, err := src.Genesis(context.Background(), plugin, genesis.Request{Validators: 4, Nodes: wemixPlacement(t)})
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}

	var g struct {
		Config map[string]any             `json:"config"`
		Alloc  map[string]json.RawMessage `json:"alloc"`
		Extra  string                     `json:"extraData"`
	}
	if err := json.Unmarshal(art.Genesis, &g); err != nil {
		t.Fatalf("generated genesis is not valid JSON: %v\n%s", err, art.Genesis)
	}
	// What the substituted template got wrong, measured against the template
	// it starts from: an empty alloc and an empty extraData. The binary fills
	// both from the governance config, and a node started on the template's
	// versions has no funded accounts and no encoded bootnode.
	//
	// minerNodeId stays "0x0" and that is correct — the template ships it that
	// way and it is a block-header field, so the genesis block has none.
	if len(g.Alloc) == 0 {
		t.Fatalf("generated genesis has an empty alloc — this is the dead genesis the old path produced:\n%s", art.Genesis)
	}
	if len(g.Extra) <= len("0x") {
		t.Fatalf("generated genesis has no extraData: %q", g.Extra)
	}
	// The chain id has to be the manifest's, not the template's default: the
	// generator passes config through, so a template prepared without it comes
	// up on the wrong network.
	if g.Config["chainId"] != float64(plugin.Manifest().ChainID) {
		t.Fatalf("chainId = %v, want the manifest's %d", g.Config["chainId"], plugin.Manifest().ChainID)
	}
	// Every declared account is funded, so a test does not have to arrange gas
	// before it can do anything.
	if len(g.Alloc) < 4 {
		t.Fatalf("alloc has %d accounts, want the key set's producers", len(g.Alloc))
	}

	// And a real datadir initializes from it.
	genesisPath := filepath.Join(dir, "for-init.json")
	if err := os.WriteFile(genesisPath, art.Genesis, 0o644); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(dir, "node1")
	if err := driver.InitDatadir(context.Background(), bin, dataDir, genesisPath); err != nil {
		t.Fatalf("gwemix init rejected the generated genesis: %v", err)
	}
}
