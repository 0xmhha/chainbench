package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativePeerFields(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native peer requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			p := filepath.Join(repoRoot(t), "tests/tc/common/node/CT-NODE-008-peers.json")
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, "case.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			raws, err := dsl.ReadFilesWithChainPreset([]string{p}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4-en1.json"))
			if err != nil {
				t.Fatal(err)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Managed local public RPC-019 fixture, not complete87/mainnet/Linux/final Web acceptance."}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/peer-fields-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("public peers failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) != 5 {
				t.Fatal("missing actual nodes", err)
			}
			var peers []map[string]any
			if err = rpc.Dial(status.Nodes.Nodes[0].RPCURL).Call(ctx, "admin_peers", &peers); err != nil {
				t.Fatal(err)
			}
			if len(peers) != 4 {
				t.Fatal("wrong fixed-fixture peer count", len(peers))
			}
			seen := map[string]bool{}
			for _, peer := range peers {
				id, _ := peer["id"].(string)
				network, _ := peer["network"].(map[string]any)
				remote, _ := network["remoteAddress"].(string)
				protocols, _ := peer["protocols"].(map[string]any)
				if id == "" || remote == "" || len(protocols) == 0 || seen[id] {
					t.Fatal("missing/duplicate actual peer fields", peer)
				}
				seen[id] = true
			}
			remoteIDs := []string{}
			for _, n := range status.Nodes.Nodes[1:] {
				var info map[string]any
				if err = rpc.Dial(n.RPCURL).Call(ctx, "admin_nodeInfo", &info); err != nil {
					t.Fatal(err)
				}
				id, _ := info["id"].(string)
				remoteIDs = append(remoteIDs, id)
				if !seen[id] {
					t.Fatal("actual remote node missing from peers", id)
				}
			}
			proof["peers"] = peers
			proof["remoteNodeIDs"] = remoteIDs
			proof["assertions"] = filepath.Join(out.SessionRoot, "tests/001_basic-peers/assert.json")
			proof["steps"] = filepath.Join(out.SessionRoot, "tests/001_basic-peers/steps.json")

		})
	}
}
