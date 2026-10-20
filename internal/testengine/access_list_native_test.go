package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestNativeDeployedSlotAccessList(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native access-list slot requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			p := filepath.Join(repoRoot(t), "tests/tc/common/tx/CT-TX-004-access-list-tx.json")
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
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Managed local original TX007 deployed-slot access-list fixture, not complete87/mainnet/Linux/final Web acceptance."}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/access-list-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("public access-list failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) < 4 {
				t.Fatal("missing actual nodes", err)
			}
			dir := filepath.Join(out.SessionRoot, "tests/001_access-list-tx")
			b, err = os.ReadFile(filepath.Join(dir, "steps.json"))
			if err != nil {
				t.Fatal(err)
			}
			var steps []session.StepResult
			if err = json.Unmarshal(b, &steps); err != nil {
				t.Fatal(err)
			}
			hashes := []string{}
			for _, s := range steps {
				if s.Type == "sendTx" || s.Type == "deployContract" {
					hashes = append(hashes, s.Hash)
				}
			}
			if len(hashes) != 5 {
				t.Fatal("missing old and slot transaction witnesses", len(hashes))
			}
			hash, deployHash := hashes[4], hashes[3]
			c := rpc.Dial(status.Nodes.Nodes[0].RPCURL)
			var tx, receipt, deployment, block map[string]any
			if err = c.Call(ctx, "eth_getTransactionByHash", &tx, hash); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getTransactionReceipt", &deployment, deployHash); err != nil {
				t.Fatal(err)
			}
			addr, ok := deployment["contractAddress"].(string)
			if !ok || len(addr) != 42 || deployment["status"] != "0x1" || receipt["status"] != "0x1" || receipt["type"] != "0x1" || tx["type"] != "0x1" || tx["to"] != addr || tx["hash"] != hash || receipt["transactionHash"] != hash || tx["blockHash"] != receipt["blockHash"] || tx["blockNumber"] != receipt["blockNumber"] {
				t.Fatal("invalid actual access-list transaction/receipt", tx, receipt, deployment)
			}
			slot := "0x" + strings.Repeat("0", 64)
			value := "0x" + strings.Repeat("0", 62) + "2a"
			list, ok := tx["accessList"].([]any)
			if !ok || len(list) != 1 {
				t.Fatal("missing required access-list tuple")
			}
			tuple, ok := list[0].(map[string]any)
			if !ok || tuple["address"] != addr {
				t.Fatal("access-list address differs from deployment")
			}
			keys, ok := tuple["storageKeys"].([]any)
			if !ok || len(keys) != 1 || keys[0] != slot || tx["input"] != value {
				t.Fatal("wrong signed storage slot/input", tuple, tx)
			}
			tag := receipt["blockNumber"]
			var stored, code, raw string
			if err = c.Call(ctx, "eth_getStorageAt", &stored, addr, slot, tag); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getCode", &code, addr, tag); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getRawTransactionByHash", &raw, hash); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getBlockByNumber", &block, tag, false); err != nil || block["hash"] != receipt["blockHash"] {
				t.Fatal("wrong actual receipt block", err)
			}
			if stored != value || code != "0x60003560005500" || !strings.HasPrefix(raw, "0x01") {
				t.Fatal("missing deployed slot write or signed type1 wire", stored, code, raw)
			}
			proof["transaction"] = tx
			proof["receipt"] = receipt
			proof["deployment"] = deployment
			proof["minedBlock"] = block
			proof["raw"] = raw
			proof["storage"] = stored
			proof["code"] = code
			proof["slot"] = slot

			proof["assertions"] = filepath.Join(dir, "assert.json")
			proof["steps"] = filepath.Join(dir, "steps.json")
		})
	}
}
