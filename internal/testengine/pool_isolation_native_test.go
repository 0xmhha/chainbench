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
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativePoolIsolation(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native pool requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			p := filepath.Join(repoRoot(t), "tests/tc/common/rpc/CT-RPC-016-isolated-en-pending-queued.json")
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, "case.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			// Include the modified consensus-halt content case as a separate related
			// regression; it is never substituted for the EN isolation case.
			paths := []string{filepath.Join(repoRoot(t), "tests/tc/common/rpc/CT-RPC-012-txpool-content-well-formed.json"), p}
			raws, err := dsl.ReadFilesWithChainPreset(paths, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4-en1.json"))
			if err != nil {
				t.Fatal(err)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Managed local RPC-018 EN peer isolation fixture, not complete87/mainnet/Linux/final Web acceptance."}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/pool-isolation-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 2 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
				if err == nil {
					observations := []map[string]any{}
					for _, n := range status.Nodes.Nodes {
						if n.Index != 1 && n.Index != 5 {
							continue
						}
						o := map[string]any{"index": n.Index}
						c := rpc.Dial(n.RPCURL)
						for _, method := range []string{"eth_blockNumber", "net_peerCount", "txpool_status", "txpool_content"} {
							var value any
							if err = c.Call(ctx, method, &value); err != nil {
								o[method+"Error"] = err.Error()
							} else {
								o[method] = value
							}
						}
						observations = append(observations, o)
					}
					proof["failureObservations"] = observations
				}
				t.Fatalf("public isolated pool failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) != 5 {
				t.Fatal("missing actual nodes", err)
			}
			dir := filepath.Join(out.SessionRoot, "tests/002_isolated-en-pending-queued")
			raw, err := os.ReadFile(filepath.Join(dir, "steps.json"))
			if err != nil {
				t.Fatal(err)
			}
			var steps []session.StepResult
			if err = json.Unmarshal(raw, &steps); err != nil {
				t.Fatal(err)
			}
			hashes := []string{}
			for _, st := range steps {
				if st.Type == "sendTx" {
					hashes = append(hashes, st.Hash)
				}
			}
			if len(hashes) != 4 {
				t.Fatal("expected funding and three isolated-gap transactions", hashes)
			}
			actual := []map[string]any{}
			c := rpc.Dial(status.Nodes.Nodes[4].RPCURL)
			for i, hash := range hashes[1:] {
				var tx, receipt map[string]any
				if err = c.Call(ctx, "eth_getTransactionByHash", &tx, hash); err != nil {
					t.Fatal(err)
				}
				if err = c.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
					t.Fatal(err)
				}
				want := []string{"0x0", "0x2", "0x1"}[i]
				if tx["hash"] != hash || receipt["transactionHash"] != hash || tx["nonce"] != want || receipt["status"] != "0x1" || tx["blockHash"] != receipt["blockHash"] {
					t.Fatal("reconnected actual pool tx mismatch", tx, receipt)
				}
				actual = append(actual, map[string]any{"transaction": tx, "receipt": receipt})
			}
			proof["transactions"] = actual
			proof["assertions"] = filepath.Join(dir, "assert.json")
			proof["steps"] = filepath.Join(dir, "steps.json")

		})
	}
}
