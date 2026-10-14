package testengine_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

type failedExecutionTrace struct {
	Failed     bool `json:"failed"`
	StructLogs []struct {
		Op    string `json:"op"`
		Error string `json:"error"`
	} `json:"structLogs"`
}

// A storage assertion is vacuous unless execution actually wrote before failing.
func traceWroteBeforeFailure(trace failedExecutionTrace) bool {
	if !trace.Failed {
		return false
	}
	for i, step := range trace.StructLogs {
		if step.Op == "SSTORE" && step.Error == "" && i+1 < len(trace.StructLogs) {
			return true
		}
	}
	return false
}

func TestFailureTraceRejectsVacuousRollback(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		want       bool
	}{
		{"write then revert", `{"failed":true,"structLogs":[{"op":"SSTORE"},{"op":"REVERT"}]}`, true},
		{"revert without write", `{"failed":true,"structLogs":[{"op":"REVERT"}]}`, false},
		{"failed write", `{"failed":true,"structLogs":[{"op":"SSTORE","error":"out of gas"},{"op":"STOP"}]}`, false},
		{"last instruction write", `{"failed":true,"structLogs":[{"op":"SSTORE"}]}`, false},
		{"successful execution", `{"failed":false,"structLogs":[{"op":"SSTORE"},{"op":"STOP"}]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var trace failedExecutionTrace
			if err := json.Unmarshal([]byte(tc.data), &trace); err != nil {
				t.Fatal(err)
			}
			if got := traceWroteBeforeFailure(trace); got != tc.want {
				t.Fatalf("write before failure=%v, want %v", got, tc.want)
			}
		})
	}
}

func failureRuntime(t *testing.T) string {
	t.Helper()
	const script = `import json,sys
sys.path.insert(0,"../../tests/webui")
from runtime_storage import create_runtime,reclaim_after_shutdown
request=json.load(sys.stdin)
if request["operation"]=="create":
 print(json.dumps(str(create_runtime(prefix="txfail-"))))
else:
 print(json.dumps(reclaim_after_shutdown(request["runtime"],request["runtime"]+"/reports")))
`
	call := func(ctx context.Context, operation, root string) ([]byte, error) {
		body, err := json.Marshal(map[string]string{"operation": operation, "runtime": root})
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(ctx, "python3", "-B", "-c", script)
		cmd.Stdin = bytes.NewReader(body)
		return cmd.CombinedOutput()
	}
	body, err := call(t.Context(), "create", "")
	if err != nil {
		t.Fatalf("create owned runtime: %v %s", err, body)
	}
	var root string
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "reports"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if _, err := verb.ChainStop(ctx, chainsetup.Deps{}, verb.ChainStopIn{DataDir: filepath.Join(root, "w")}); err != nil {
			t.Error(err)
			return
		}
		if body, err := call(ctx, "reclaim", root); err != nil {
			t.Errorf("preserve runtime after reclamation refusal: %v %s", err, body)
		}
	})
	t.Log("owned transaction failure runtime:", root)
	return root
}

func TestNativeTransactionFailureRollback(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native rollback proof requires " + f.variable)
		}
	}
	cases := []string{"CT-CONTRACT-006-revert-tx-status-zero.json", "CT-CONTRACT-007-out-of-gas-consumes-all.json"}
	for _, family := range families {
		t.Run(family.chain, func(t *testing.T) {
			root := failureRuntime(t)
			workspace := filepath.Join(root, "w")
			raws := [][]byte{}
			for _, name := range cases {
				p := filepath.Join(repoRoot(t), "tests/tc/common/contract", name)
				raw, err := dsl.ReadFilesWithChainPreset([]string{p}, family.chain+"-bp4")
				if err != nil {
					t.Fatal(err)
				}
				raws = append(raws, raw...)
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
			defer cancel()
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: workspace, ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(family.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": family.chain, "binary": os.Getenv(family.variable), "session": out.SessionRoot, "summary": out.Summary}
			if runErr != nil {
				proof["runError"] = runErr.Error()
			}
			traces := map[string]failedExecutionTrace{}
			transactions := map[string]map[string]any{}
			defer func() {
				proof["traces"] = traces
				proof["transactions"] = transactions
				data, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				p := filepath.Join(root, "reports/transaction-failure-proof.json")
				if err := os.WriteFile(p, data, 0600); err != nil {
					t.Error(err)
				}
				t.Log("transaction failure proof:", p)
			}()
			if runErr != nil || out.Summary.Summary.Pass != 2 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("native cases failed: %v %+v", runErr, out.Summary)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
			if err != nil {
				t.Fatal(err)
			}
			if len(status.Nodes.Nodes) == 0 {
				t.Fatal("no actual node record")
			}
			client := rpc.Dial(status.Nodes.Nodes[0].RPCURL)
			for i, name := range []string{"revert-tx-status-zero", "out-of-gas-consumes-all"} {
				dir := filepath.Join(out.SessionRoot, "tests", fmt.Sprintf("%03d_%s", i+1, name))
				data, err := os.ReadFile(filepath.Join(dir, "steps.json"))
				if err != nil {
					t.Fatal(err)
				}
				var steps []session.StepResult
				if err := json.Unmarshal(data, &steps); err != nil {
					t.Fatal(err)
				}
				var failed []session.StepResult
				for _, step := range steps {
					if step.Receipt["status"] == "0x0" {
						failed = append(failed, step)
					}
				}
				if len(failed) != 1 || failed[0].Hash == "" || failed[0].Receipt["blockNumber"] == nil {
					t.Fatal("not exactly one mined failed transaction", failed)
				}
				var transaction map[string]any
				if err := client.Call(ctx, "eth_getTransactionByHash", &transaction, failed[0].Hash); err != nil {
					t.Fatal(err)
				}
				limit := "0x186a0"
				if name == "out-of-gas-consumes-all" {
					limit = "0xc350"
				}
				if transaction["hash"] != failed[0].Hash || transaction["gas"] != limit || transaction["input"] != "0x01" || transaction["blockHash"] != failed[0].Receipt["blockHash"] {
					t.Fatal("actual mined transaction does not match the declared failure input", transaction)
				}
				transactions[name] = transaction
				var trace failedExecutionTrace
				if err := client.Call(ctx, "debug_traceTransaction", &trace, failed[0].Hash, map[string]any{"disableStack": true, "disableMemory": true, "disableStorage": true}); err != nil {
					t.Fatal(err)
				}
				traces[name] = trace
				if !traceWroteBeforeFailure(trace) {
					t.Errorf("%s never completed SSTORE before failure; rollback proof is vacuous", name)
				}
			}
		})
	}
}
