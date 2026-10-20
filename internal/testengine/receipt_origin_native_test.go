package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeReceiptBlockOrigin(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native receipt origin requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			p := filepath.Join(repoRoot(t), "tests/tc/common/rpc/CT-RPC-005-transaction-receipt-fields.json")
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, "case.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			raws, err := dsl.ReadFilesWithChainPreset([]string{p}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4.json"))
			if err != nil {
				t.Fatal(err)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Managed local public receipt origin and genesis activation fixture, not complete87/mainnet/Linux/final Web acceptance."}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/receipt-origin-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("public receipt origin failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) < 4 {
				t.Fatal("missing actual nodes", err)
			}
			dir := filepath.Join(out.SessionRoot, "tests/001_transaction-receipt-fields")
			b, err = os.ReadFile(filepath.Join(dir, "steps.json"))
			if err != nil {
				t.Fatal(err)
			}
			var steps []session.StepResult
			if err = json.Unmarshal(b, &steps); err != nil {
				t.Fatal(err)
			}
			hash := ""
			for _, s := range steps {
				if s.Type == "sendTx" {
					hash = s.Hash
				}
			}
			c := rpc.Dial(status.Nodes.Nodes[0].RPCURL)
			var receipt, byNumber, byHash map[string]any
			if hash == "" {
				t.Fatal("missing submitted witness hash")
			}
			if err = c.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
				t.Fatal(err)
			}
			if receipt["status"] != "0x1" {
				t.Fatal("witness not mined", receipt)
			}
			tag, ok := receipt["blockNumber"].(string)
			if !ok || tag == "" {
				t.Fatal("missing exact block number")
			}
			if err = c.Call(ctx, "eth_getBlockByNumber", &byNumber, tag, true); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getBlockByHash", &byHash, receipt["blockHash"], true); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(byNumber, byHash) || byNumber["number"] != tag || byNumber["hash"] != receipt["blockHash"] {
				t.Fatal("full block responses differ", byNumber, byHash)
			}
			extra, ok := byNumber["extraData"].(string)
			if !ok || len(extra) <= 2 {
				t.Fatal("empty consensus extra")
			}
			miner, ok := byNumber["miner"].(string)
			if !ok || len(miner) != 42 {
				t.Fatal("missing miner")
			}
			txs, ok := byNumber["transactions"].([]any)
			if !ok || len(txs) == 0 {
				t.Fatal("missing full transaction array")
			}
			seen := false
			for _, raw := range txs {
				tx, ok := raw.(map[string]any)
				if !ok || tx["blockHash"] != receipt["blockHash"] || tx["blockNumber"] != tag {
					t.Fatal("transaction object does not belong to pinned block", raw)
				}
				if tx["hash"] == hash {
					seen = true
				}
			}
			if !seen {
				t.Fatal("submitted transaction absent from block")
			}
			var tx map[string]any
			if err = c.Call(ctx, "eth_getTransactionByHash", &tx, hash); err != nil {
				t.Fatal(err)
			}
			if tx["blockNumber"] != tag || tx["blockHash"] != receipt["blockHash"] || tx["hash"] != hash || tx["type"] != receipt["type"] {
				t.Fatal("transaction/receipt origins differ")
			}
			gas, ok := new(big.Int).SetString(fmt.Sprint(receipt["gasUsed"]), 0)
			if !ok || gas.Sign() <= 0 {
				t.Fatal("missing positive receipt gas")
			}
			limit, ok := new(big.Int).SetString(fmt.Sprint(tx["gas"]), 0)
			if !ok || gas.Cmp(limit) > 0 {
				t.Fatal("receipt gas exceeds transaction limit")
			}
			height, e := strconv.ParseUint(strings.TrimPrefix(tag, "0x"), 16, 64)
			if e != nil {
				t.Fatal(e)
			}
			forkName := map[string]string{"wbft": "croissantBlock", "stablenet": "londonBlock", "wemix": "londonBlock"}[f.chain]
			var genesisPath string
			var activation uint64
			err = filepath.WalkDir(filepath.Join(root, "w"), func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.IsDir() && (d.Name() == "chaindata" || d.Name() == "lightchaindata" || d.Name() == "etcd") {
					return filepath.SkipDir
				}
				if d.IsDir() || d.Name() != "genesis.json" {
					return nil
				}
				raw, e := os.ReadFile(path)
				if e != nil {
					return e
				}
				var g struct{ Config map[string]json.RawMessage }
				if e = json.Unmarshal(raw, &g); e != nil {
					return e
				}
				field, ok := g.Config[forkName]
				if !ok || string(field) == "null" {
					return fmt.Errorf("missing declared activation %s", forkName)
				}
				if e = json.Unmarshal(field, &activation); e != nil {
					return e
				}
				genesisPath = path
				return fs.SkipAll
			})
			if err != nil || genesisPath == "" || height <= activation {
				t.Fatal("receipt not after actual genesis activation", err, activation, height)
			}
			proof["transaction"] = tx
			proof["genesisPath"] = genesisPath
			proof["activationField"] = forkName
			proof["activation"] = activation
			proof["numberParams"] = []any{tag, true}
			proof["hashParams"] = []any{receipt["blockHash"], true}
			proof["byNumber"] = byNumber
			proof["byHash"] = byHash
			proof["receipt"] = receipt
			proof["witnessHash"] = hash
			proof["assertions"] = filepath.Join(dir, "assert.json")
			proof["steps"] = filepath.Join(dir, "steps.json")
		})
	}
}
