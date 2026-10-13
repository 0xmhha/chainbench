package chainsetup_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/preset"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestBlueprintPinsServersInDeclaredOrder(t *testing.T) {
	dir := t.TempDir()
	d := chainsetup.Deps{Clock: fixedClock()}
	ctx := context.Background()
	if _, err := verb.ChainNew(ctx, d, verb.ChainNewIn{DataDir: dir, Chain: "wbft"}); err != nil {
		t.Fatal(err)
	}
	declaration := filepath.Join(t.TempDir(), "network.yaml")
	if err := os.WriteFile(declaration, []byte("chain: wbft\nnodes:\n  - {role: bp, server: b}\n  - {role: bp, server: a}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory := writeInventory(t, `version: 2
pool:
  hosts: [{name: a, addr: 192.0.2.1}, {name: b, addr: 192.0.2.2}]
  slots: 2
  ports: {p2p: {base: 31000, step: 10}, rpc: {base: 8600, step: 10}}
`)
	if _, err := verb.ChainAllocate(ctx, d, chainsetup.ChainAllocateIn{DataDir: dir, BlueprintPath: declaration, Server: resource.ServerRef{SetPath: inventory, All: true}}); err != nil {
		t.Fatal(err)
	}
	state := stateOf(t, dir, d)
	for i, want := range []struct{ server, host string }{{"b", "192.0.2.2"}, {"a", "192.0.2.1"}} {
		if state.Nodes[i].Server != want.server || state.Nodes[i].Host != want.host {
			t.Fatalf("node%d=%+v, want %+v", i+1, state.Nodes[i], want)
		}
	}
}

func TestFreshKeysAskFamilyForBLS(t *testing.T) {
	for _, chain := range []string{"wemix", "wbft", "stablenet"} {
		for _, source := range []string{"generate", "declared", "mixed-table"} {
			t.Run(chain+"/"+source, func(t *testing.T) {
				dir := t.TempDir()
				keys := filepath.Join(dir, "keys")
				ws, err := chainsetup.Open(dir, fixedClock())
				if err != nil {
					t.Fatal(err)
				}
				if _, err = ws.New(chainsetup.NewOpts{Chain: chain, KeysDir: keys}); err != nil {
					t.Fatal(err)
				}
				opts := chainsetup.AllocateOpts{BPCount: 2}
				keyOpts := chainsetup.KeysOpts{Source: source}
				if source == "declared" {
					doc := fmt.Sprintf("chain: %s\nnodes:\n - {role: bp, nodekey: {hex: '0x%s'}}\n - {role: bp, nodekey: {hex: '0x%s'}}\n", chain, strings.Repeat("11", 32), strings.Repeat("22", 32))
					bp, e := blueprint.Parse([]byte(doc))
					if e != nil {
						t.Fatal(e)
					}
					opts.Blueprint = &bp
					keyOpts.Blueprint = &bp
				}
				if source == "mixed-table" {
					pinned := filepath.Join(dir, "pinned.key")
					if err := os.WriteFile(pinned, []byte(strings.Repeat("11", 32)), 0o600); err != nil {
						t.Fatal(err)
					}
					opts.Topology = &node.Topology{Chain: chain, Nodes: []node.Entry{{Index: 1, Role: "bp", Key: pinned}, {Index: 2, Role: "bp"}}}
					keyOpts.Source = ""
				}
				if _, err := ws.Allocate(opts); err != nil {
					t.Fatal(err)
				}
				if _, err := ws.Keys(context.Background(), keyOpts); err != nil {
					t.Fatal(err)
				}
				set, err := preset.LoadKeyPreset(keys)
				if err != nil {
					t.Fatal(err)
				}
				for _, identity := range set.Nodes {
					if (identity.BLS != nil) != (chain != "wemix") {
						t.Errorf("%s %s node%d BLS=%v", chain, source, identity.Index, identity.BLS != nil)
					}
				}
				if len(set.Network.Validators) != 2 {
					t.Fatalf("wrong validator set: %+v", set.Network)
				}
				if source != "generate" && !strings.EqualFold(set.Nodes[0].Address, "0x19e7e376e7c213b7e7e7e46cc70a5dd086daff2a") {
					t.Fatal("BLS selection changed the declared account address")
				}
			})
		}
	}
}

func TestPinnedLocalBlueprintDoesNotRequestSSHCredentials(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := chainsetup.Deps{Clock: fixedClock()}
	if _, err := verb.ChainNew(ctx, d, verb.ChainNewIn{DataDir: dir, Chain: "wbft", KeysDir: filepath.Join(dir, "keys")}); err != nil {
		t.Fatal(err)
	}
	declaration := filepath.Join(dir, "network.yaml")
	if err := os.WriteFile(declaration, []byte("chain: wbft\nnodes:\n - {role: bp, server: local}\n - {role: bp, server: local}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	set := writeInventory(t, localInventory)
	if _, err := verb.ChainAllocate(ctx, d, chainsetup.ChainAllocateIn{DataDir: dir, BlueprintPath: declaration, Server: resource.ServerRef{SetPath: set, Name: "local"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := verb.ChainKeys(ctx, d, verb.ChainKeysIn{DataDir: dir, BlueprintPath: declaration, Source: "generate"}); err != nil {
		t.Fatal(err)
	}
	if _, err := verb.ChainGenesis(ctx, d, chainsetup.ChainGenesisIn{DataDir: dir}); err != nil {
		t.Fatalf("local pin requested SSH: %v", err)
	}
	for _, node := range stateOf(t, dir, d).Nodes {
		if node.Server != "" || node.Host != "127.0.0.1" {
			t.Fatalf("locality changed: %+v", node)
		}
	}
}
