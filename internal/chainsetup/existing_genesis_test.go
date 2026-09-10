package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
)

// TestGenesis_ExistingIsUsedVerbatim is W4's finished-genesis contract: a genesis
// named by an "existing" reference is written to the target verbatim, not built
// from a template. This is what lets a prepared bundle (existing genesis +
// preset keyring + per-node configs) drive a regression run.
func TestGenesis_ExistingIsUsedVerbatim(t *testing.T) {
	dir := t.TempDir()
	presetDir := filepath.Join("..", "..", "keys", "preset")
	// A distinctive finished genesis (valid JSON, not what the builder would
	// make) that still names the validators the two-producer key set provides —
	// a wbft genesis without them is refused, and rightly, since it cannot sign.
	preset, err := store.LoadPreset(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	net := preset.NetworkFor(2)
	extra, err := wbft.ExtraData(net.Validators, net.BLSKeys)
	if err != nil {
		t.Fatal(err)
	}
	finished := `{"config":{"chainId":424242},"note":"prepared-regression","alloc":{},"extraData":"` + extra + `"}`
	genPath := filepath.Join(dir, "prepared-genesis.json")
	if err := os.WriteFile(genPath, []byte(finished), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: presetDir}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp"}, {Index: 2, Role: "bp"},
	}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{Existing: genPath}); err != nil {
		t.Fatalf("genesis (existing): %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "genesis.json"))
	if err != nil {
		t.Fatalf("read genesis: %v", err)
	}
	if string(got) != finished {
		t.Fatalf("genesis is not the finished file verbatim:\n%s", got)
	}
}

// TestGenesis_ExistingRejectsInvalidJSON: a finished genesis that is not valid
// JSON fails the step rather than being written and failing at node init.
func TestGenesis_ExistingRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	genPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(genPath, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: filepath.Join("..", "..", "keys", "preset")}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{{Index: 1, Role: "bp"}, {Index: 2, Role: "bp"}}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{Existing: genPath}); err == nil {
		t.Fatal("an invalid-JSON existing genesis was accepted")
	}
}

// TestGenesis_ExistingRejectsChangeRequests is MON-007. A finished genesis is
// written byte for byte, so a change arriving with it was dropped in silence —
// and the step still reported it as applied, while the advertised capabilities
// were derived from the dropped fork heights. The request is refused now, and
// the refusal names what could not be applied.
func TestGenesis_ExistingRejectsChangeRequests(t *testing.T) {
	cases := []struct {
		name string
		in   chainsetup.NetGenesisIn
		want string
	}{
		{"chain id", chainsetup.NetGenesisIn{GenesisExisting: "/g.json", ChainID: 424243}, "chain id"},
		{"hardfork height", chainsetup.NetGenesisIn{GenesisExisting: "/g.json", Set: []string{"bohoBlock=10"}}, "override"},
		{"both", chainsetup.NetGenesisIn{GenesisExisting: "/g.json", ChainID: 7, Set: []string{"bohoBlock=10"}}, "chain id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := chainsetup.NetGenesis(context.Background(), chainsetup.Deps{}, tc.in)
			if err == nil {
				t.Fatal("a change alongside an existing genesis must be refused")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the refusal should name what was asked for (%s): %v", tc.want, err)
			}
		})
	}
}

// TestGenesis_ExistingAloneIsStillAccepted: the refusal is about the pairing,
// not about naming a finished genesis.
func TestGenesis_ExistingAloneIsStillAccepted(t *testing.T) {
	// A missing workspace fails later than the conflict check, which is enough
	// to show the conflict check did not fire.
	_, err := chainsetup.NetGenesis(context.Background(), chainsetup.Deps{},
		chainsetup.NetGenesisIn{DataDir: t.TempDir(), GenesisExisting: "/g.json"})
	if err != nil && strings.Contains(err.Error(), "used verbatim") {
		t.Fatalf("an existing genesis on its own must not be refused: %v", err)
	}
}

// TestWorkspaceGenesis_RefusesChangeRequestsOnTheMethodItself puts MON-007's
// rule where the operation is rather than where one caller happens to be.
//
// The check lived in genesisOpts, the helper that turns a NetGenesisIn into
// options. Every caller went through it, so the behaviour was right — but
// Workspace.Genesis is exported and takes the options directly, so "a finished
// genesis is never quietly changed" was a property of the callers, not of the
// step. The next caller to assemble a GenesisOpts by hand would have inherited
// the silent-drop bug the rule exists to prevent, and no test would have moved.
func TestWorkspaceGenesis_RefusesChangeRequestsOnTheMethodItself(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: filepath.Join(dir, "keys")}); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		opts chainsetup.GenesisOpts
		want string
	}{
		{"chain id", chainsetup.GenesisOpts{Existing: "/g.json", ChainID: 4242}, "chain id"},
		{"override", chainsetup.GenesisOpts{Existing: "/g.json", Overrides: map[string]string{"bohoBlock": "10"}}, "override"},
		{"overlay", chainsetup.GenesisOpts{Existing: "/g.json", Overlay: []byte(`{"config":{}}`)}, "overlay"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ws.Genesis(context.Background(), tc.opts)
			if err == nil {
				t.Fatal("a change alongside an existing genesis must be refused by the method itself")
			}
			if !strings.Contains(err.Error(), "used verbatim") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the refusal should name what could not be applied (%s): %v", tc.want, err)
			}
		})
	}

	// The refusal is about the pairing. A finished genesis on its own still
	// reaches the build, and fails later for its own reasons.
	_, err = ws.Genesis(context.Background(), chainsetup.GenesisOpts{Existing: "/g.json"})
	if err != nil && strings.Contains(err.Error(), "used verbatim") {
		t.Fatalf("an existing genesis on its own must not be refused: %v", err)
	}
}
