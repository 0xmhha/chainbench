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

func TestNativeReceiptBoundEventLogs(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native event logs requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			p := filepath.Join(repoRoot(t), "tests/tc/common/rpc/CT-RPC-009-event-logs-query.json")
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
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Managed local receipt-bound event log fixture, not complete87/mainnet/Linux/final Web acceptance."}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/event-log-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("public event logs failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) < 4 {
				t.Fatal("missing actual nodes", err)
			}
			dir := filepath.Join(out.SessionRoot, "tests/001_contract-event-emitted")
			b, err = os.ReadFile(filepath.Join(dir, "steps.json"))
			if err != nil {
				t.Fatal(err)
			}
			var steps []session.StepResult
			if err = json.Unmarshal(b, &steps); err != nil {
				t.Fatal(err)
			}
			hash, deployHash := "", ""
			for _, s := range steps {
				if s.Type == "sendTx" {
					if deployHash == "" {
						deployHash = s.Hash
					}
					hash = s.Hash
				}
			}
			c := rpc.Dial(status.Nodes.Nodes[0].RPCURL)
			var receipt, deployment map[string]any
			if hash == "" || deployHash == "" || hash == deployHash {
				t.Fatal("missing actual deploy/event hashes")
			}
			if err = c.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getTransactionReceipt", &deployment, deployHash); err != nil {
				t.Fatal(err)
			}
			addr, ok := deployment["contractAddress"].(string)
			if !ok || len(addr) != 42 || deployment["status"] != "0x1" || receipt["status"] != "0x1" {
				t.Fatal("invalid actual deployment/event receipts")
			}
			tag, ok := receipt["blockNumber"].(string)
			if !ok || tag == "" {
				t.Fatal("missing mined event block")
			}
			filter := map[string]any{"address": addr, "fromBlock": tag, "toBlock": tag}
			var logs []map[string]any
			if err = c.Call(ctx, "eth_getLogs", &logs, filter); err != nil {
				t.Fatal(err)
			}
			if len(logs) != 1 {
				t.Fatal("wrong event count", logs)
			}
			log := logs[0]
			if log["address"] != addr || log["blockNumber"] != tag || log["blockHash"] != receipt["blockHash"] || log["transactionHash"] != hash || log["removed"] != false || log["data"] != "0x" {
				t.Fatal("returned log differs from deployed event receipt", log, receipt)
			}
			topics, ok := log["topics"].([]any)
			if !ok || len(topics) != 1 || topics[0] != "0x1111111111111111111111111111111111111111111111111111111111111111" {
				t.Fatal("wrong actual LOG1 topic", log)
			}
			receiptLogs, ok := receipt["logs"].([]any)
			if !ok || len(receiptLogs) != 1 {
				t.Fatal("missing actual receipt log")
			}
			if r, ok := receiptLogs[0].(map[string]any); !ok || r["logIndex"] != log["logIndex"] || r["transactionHash"] != hash || r["address"] != addr {
				t.Fatal("receipt/query event identity mismatch")
			}
			filter["topics"] = []string{"0x2222222222222222222222222222222222222222222222222222222222222222"}
			var negative []map[string]any
			if err = c.Call(ctx, "eth_getLogs", &negative, filter); err != nil || len(negative) != 0 {
				t.Fatal("wrong-topic filter failed", err, negative)
			}
			delete(filter, "topics")
			proof["filter"] = filter
			proof["logs"] = logs
			proof["receipt"] = receipt
			proof["deployment"] = deployment
			proof["negative"] = negative
			proof["eventHash"] = hash

			proof["assertions"] = filepath.Join(dir, "assert.json")
			proof["steps"] = filepath.Join(dir, "steps.json")
		})
	}
}
