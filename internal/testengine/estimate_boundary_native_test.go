package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
	"testing"
)

type estimateBoundary struct {
	BalanceBefore, BalanceAfter, Fee string
	Estimate, Limit, Used            uint64
	Status, Before, After            string
	Transaction, Receipt             map[string]any
	Trace                            failedExecutionTrace
}

func validateEstimateBoundary(p estimateBoundary, counterexample bool) error {
	if p.Estimate <= 21016 || p.Limit < 21016 || p.Used == 0 || p.Used > p.Limit || p.Before != "42" {
		return fmt.Errorf("missing or invalid estimate, gas, or initial state")
	}
	tx, receipt := p.Transaction, p.Receipt
	if tx["hash"] == nil || tx["hash"] == "" || tx["hash"] != receipt["transactionHash"] || tx["blockHash"] == nil || tx["blockHash"] == "" || tx["blockHash"] != receipt["blockHash"] || receipt["blockNumber"] == nil || tx["input"] != "0x01" || tx["gas"] != fmt.Sprintf("0x%x", p.Limit) || receipt["gasUsed"] != fmt.Sprintf("0x%x", p.Used) || receipt["status"] != p.Status {
		return fmt.Errorf("actual mined transaction or receipt does not match")
	}
	wrote := false
	for i, step := range p.Trace.StructLogs {
		if step.Op == "SSTORE" && step.Error == "" && i+1 < len(p.Trace.StructLogs) {
			wrote = true
		}
	}
	if !wrote {
		return fmt.Errorf("no completed SSTORE witness")
	}
	if counterexample {
		if p.Limit != p.Estimate-1 || p.Status != "0x1" || p.Used >= p.Limit || p.After != "1" || p.Trace.Failed {
			return fmt.Errorf("not an estimate-minus-one success counterexample")
		}
	} else if p.Status != "0x0" || p.Used != p.Limit || p.After != "42" || !p.Trace.Failed {
		return fmt.Errorf("not a mined full-gas failure with rollback")
	}
	return nil
}

func TestEstimateBoundaryRejectsFalseEvidence(t *testing.T) {
	good := func() estimateBoundary {
		return estimateBoundary{Estimate: 32000, Limit: 31999, Used: 31165, Status: "0x1", Before: "42", After: "1", Transaction: map[string]any{"hash": "tx", "blockHash": "block", "gas": "0x7cff", "input": "0x01"}, Receipt: map[string]any{"transactionHash": "tx", "blockHash": "block", "blockNumber": "0x5", "status": "0x1", "gasUsed": "0x79bd"}, Trace: boundaryTrace(t, false)}
	}
	if err := validateEstimateBoundary(good(), true); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*estimateBoundary)
	}{
		{"estimate limit mismatch", func(p *estimateBoundary) { p.Limit-- }},
		{"no estimate", func(p *estimateBoundary) { p.Estimate = 0 }},
		{"missing transaction", func(p *estimateBoundary) { p.Transaction = nil }},
		{"wrong transaction input", func(p *estimateBoundary) { p.Transaction["input"] = "0x" }},
		{"wrong mined hash", func(p *estimateBoundary) { p.Receipt["transactionHash"] = "other" }},
		{"wrong mined block", func(p *estimateBoundary) { p.Receipt["blockHash"] = "other" }},
		{"wrong actual limit", func(p *estimateBoundary) { p.Transaction["gas"] = "0x10000" }},
		{"wrong actual used", func(p *estimateBoundary) { p.Receipt["gasUsed"] = "0x1" }},
		{"failed receipt claimed as success", func(p *estimateBoundary) { p.Status = "0x0" }},
		{"unchanged state", func(p *estimateBoundary) { p.After = "42" }},
		{"no initial storage witness", func(p *estimateBoundary) { p.Before = "" }},
		{"no completed write", func(p *estimateBoundary) { p.Trace = failedExecutionTrace{} }},
		{"trace failure", func(p *estimateBoundary) { p.Trace.Failed = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := good()
			tc.change(&p)
			if validateEstimateBoundary(p, true) == nil {
				t.Fatal("invalid estimate boundary evidence accepted")
			}
		})
	}
}

func boundaryTrace(t *testing.T, failed bool) failedExecutionTrace {
	t.Helper()
	var trace failedExecutionTrace
	if err := json.Unmarshal([]byte(fmt.Sprintf(`{"failed":%v,"structLogs":[{"op":"SSTORE"},{"op":"STOP"}]}`, failed)), &trace); err != nil {
		t.Fatal(err)
	}
	return trace
}

func TestEstimateBoundaryRejectsFalseRollback(t *testing.T) {
	good := func() estimateBoundary {
		return estimateBoundary{Estimate: 32000, Limit: 31164, Used: 31164, Status: "0x0", Before: "42", After: "42", Transaction: map[string]any{"hash": "tx", "blockHash": "block", "gas": "0x79bc", "input": "0x01"}, Receipt: map[string]any{"transactionHash": "tx", "blockHash": "block", "blockNumber": "0x6", "status": "0x0", "gasUsed": "0x79bc"}, Trace: boundaryTrace(t, true)}
	}
	if err := validateEstimateBoundary(good(), false); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*estimateBoundary)
	}{
		{"partial consumption", func(p *estimateBoundary) { p.Used-- }},
		{"state committed", func(p *estimateBoundary) { p.After = "1" }},
		{"no failed trace", func(p *estimateBoundary) { p.Trace.Failed = false }},
		{"no write witness", func(p *estimateBoundary) { p.Trace = failedExecutionTrace{Failed: true} }},
		{"successful receipt", func(p *estimateBoundary) { p.Status = "0x1" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := good()
			tc.change(&p)
			if validateEstimateBoundary(p, false) == nil {
				t.Fatal("invalid rollback accepted")
			}
		})
	}
}

// Finite London bytecode: initial slot 0 = 42; calldata writes 1, burns a
// bounded 5,120 gas and stops. Empty calldata reads slot 0. No storage refund.
func finiteWriteCode() (initCode, runtime string) {
	burn := strings.Repeat("600050", 1024)
	view := 11 + len(burn)/2 + 1
	runtime = fmt.Sprintf("361561%04x576001600055%s005b60005460005260206000f3", view, burn)
	return fmt.Sprintf("0x602a60005561%04x806100126000396000f3%s", len(runtime)/2, runtime), "0x" + runtime
}

// waitHeadAt waits until the node's head is at least the receipt's block, so
// reads of "latest" state include that block's effects.
func waitHeadAt(ctx context.Context, t *testing.T, c *rpc.Client, blockNumber any, deadline time.Time) {
	t.Helper()
	raw, ok := blockNumber.(string)
	if !ok {
		t.Fatalf("receipt has no block number: %v", blockNumber)
	}
	want, err := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 64)
	if err != nil {
		t.Fatalf("receipt block number %q: %v", raw, err)
	}
	for {
		head, err := c.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if head >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("head %d did not reach receipt block %d", head, want)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// This diagnostic records TX-018's original input without assuming that an
// estimate is exact. A success counterexample is not original acceptance.
func TestNativeEstimateMinusOneBoundary(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("estimate boundary proof requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			key, caller, err := accounts.GenerateKey()
			if err != nil {
				t.Fatal(err)
			}
			initCode, runtime := finiteWriteCode()
			steps := []map[string]any{}
			for _, name := range []string{"a", "b", "c"} {
				steps = append(steps, map[string]any{"do": "deployContract", "from": "node1", "bytecode": initCode, "gas": "1000000", "save": name}, map[string]any{"expect": "codeAt", "address": "$" + name, "compare": "Equal", "is": runtime})
			}
			steps = append(steps, map[string]any{"do": "sendTx", "from": "node1", "to": caller, "value": "10000000000000000000", "gas": "21000", "expect": "receipt"})
			raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "estimate-boundary", "requires": []string{"rpc"}, "chainPreset": map[string]any{"extends": f.chain + "-bp4"}, "steps": steps})
			if err != nil {
				t.Fatal(err)
			}
			input := filepath.Join(root, "estimate-boundary.json")
			if err := os.WriteFile(input, raw, 0600); err != nil {
				t.Fatal(err)
			}
			specs, err := dsl.ReadFilesWithChainPreset([]string{input}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4.json"))
			if err != nil {
				t.Fatal(err)
			}
			canonical, err := dsl.ReadFilesWithChainPreset([]string{filepath.Join(repoRoot(t), "tests/tc/common/contract/CT-CONTRACT-008-finite-gas-boundary.json")}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4.json"))
			if err != nil {
				t.Fatal(err)
			}
			specs = append(canonical, specs...)
			out, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "tx018OriginalComplete": false}
			defer func() {
				data, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err := os.WriteFile(filepath.Join(root, "reports/estimate-boundary-proof.json"), data, 0600); err != nil {
					t.Error(err)
				}
			}()
			if err != nil || out.Summary.Summary.Pass != 2 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatalf("setup failed: %v %+v", err, out.Summary)
			}
			data, err := os.ReadFile(filepath.Join(out.SessionRoot, "tests/002_estimate-boundary/steps.json"))
			if err != nil {
				t.Fatal(err)
			}
			var records []session.StepResult
			if err := json.Unmarshal(data, &records); err != nil {
				t.Fatal(err)
			}
			contracts := []string{}
			for _, r := range records {
				if address, ok := r.Receipt["contractAddress"].(string); ok && address != "" {
					contracts = append(contracts, address)
				}
			}
			if len(contracts) != 3 || contracts[0] == contracts[1] || contracts[1] == contracts[2] || contracts[0] == contracts[2] {
				t.Fatal("not three independent deployed contracts", contracts)
			}
			status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(status.Nodes.Nodes) == 0 {
				t.Fatal(err)
			}
			endpoint := status.Nodes.Nodes[0].RPCURL
			c := rpc.Dial(endpoint)
			provider, err := accounts.ForChain(f.chain)
			if err != nil {
				t.Fatal(err)
			}
			wallet, err := provider.OpenWallet(ctx, key, endpoint)
			if err != nil {
				t.Fatal(err)
			}
			word := func(address string) string {
				v, err := c.EthCall(ctx, address, "0x")
				if err != nil {
					t.Fatal(err)
				}
				var slot string
				if err := c.Call(ctx, "eth_getStorageAt", &slot, address, "0x0", "latest"); err != nil || slot != v {
					t.Fatalf("call/storage differ: %s %s %v", v, slot, err)
				}
				n, ok := new(big.Int).SetString(strings.TrimPrefix(v, "0x"), 16)
				if !ok {
					t.Fatal("invalid storage word", v)
				}
				return n.String()
			}
			request := map[string]any{"from": caller, "to": contracts[0], "data": "0x01"}
			var estimateHex string
			if err := c.Call(ctx, "eth_estimateGas", &estimateHex, request); err != nil {
				t.Fatal(err)
			}
			estimate, err := strconv.ParseUint(strings.TrimPrefix(estimateHex, "0x"), 16, 64)
			if err != nil || estimate <= 21016 {
				t.Fatal("invalid estimate", estimateHex, err)
			}
			proof["estimateRequest"] = request
			proof["estimateResult"] = estimateHex
			observe := func(address string, limit uint64) estimateBoundary {
				p := estimateBoundary{Estimate: estimate, Limit: limit, Before: word(address)}
				balance, err := c.BalanceAt(ctx, caller)
				if err != nil {
					t.Fatal(err)
				}
				var priceHex string
				if err := c.Call(ctx, "eth_gasPrice", &priceHex); err != nil {
					t.Fatal(err)
				}
				price, ok := new(big.Int).SetString(strings.TrimPrefix(priceHex, "0x"), 16)
				if !ok {
					t.Fatal(priceHex)
				}
				hash, err := wallet.SendLegacyTx(ctx, accounts.LegacyTxArgs{ToHex: address, Value: new(big.Int), Data: []byte{1}, Gas: limit, GasPrice: price})
				if err != nil {
					t.Fatal(err)
				}
				deadline := time.Now().Add(90 * time.Second)
				for {
					if err := c.Call(ctx, "eth_getTransactionReceipt", &p.Receipt, hash); err != nil {
						t.Fatal(err)
					}
					if p.Receipt != nil {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("receipt timeout", hash)
					}
					select {
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					case <-time.After(200 * time.Millisecond):
					}
				}
				// A node can serve the receipt a moment before its head moves to
				// that block, and the balance and storage below read "latest".
				// Read them only once the head has reached the receipt's block.
				waitHeadAt(ctx, t, c, p.Receipt["blockNumber"], deadline)
				if err := c.Call(ctx, "eth_getTransactionByHash", &p.Transaction, hash); err != nil {
					t.Fatal(err)
				}
				if err := c.Call(ctx, "debug_traceTransaction", &p.Trace, hash, map[string]any{"disableStack": true, "disableMemory": true, "disableStorage": true}); err != nil {
					t.Fatal(err)
				}
				p.Status, _ = p.Receipt["status"].(string)
				used, _ := p.Receipt["gasUsed"].(string)
				p.Used, err = strconv.ParseUint(strings.TrimPrefix(used, "0x"), 16, 64)
				if err != nil {
					t.Fatal(err)
				}
				p.After = word(address)
				actualPrice, ok := new(big.Int).SetString(strings.TrimPrefix(fmt.Sprint(p.Receipt["effectiveGasPrice"]), "0x"), 16)
				if !ok {
					t.Fatal("missing actual fee price")
				}
				after, err := c.BalanceAt(ctx, caller)
				if err != nil {
					t.Fatal(err)
				}
				fee := new(big.Int).Mul(new(big.Int).SetUint64(p.Used), actualPrice)
				p.BalanceBefore, p.BalanceAfter, p.Fee = balance.String(), after.String(), fee.String()
				want := new(big.Int).Sub(balance, fee)
				if after.Cmp(want) != 0 {
					t.Fatal("fee balance mismatch", after, want)
				}
				return p
			}
			baseline := observe(contracts[0], estimate)
			proof["successfulBaseline"] = baseline
			if baseline.Status != "0x1" || baseline.Before != "42" || baseline.After != "1" || baseline.Trace.Failed || baseline.Used <= 21016 || baseline.Used > estimate || baseline.Transaction["gas"] != estimateHex || baseline.Transaction["hash"] != baseline.Receipt["transactionHash"] || baseline.Transaction["input"] != "0x01" {
				t.Fatal("invalid successful estimate baseline", baseline)
			}
			first := observe(contracts[1], estimate-1)
			proof["estimateMinusOne"] = first
			counterexample := first.Status == "0x1"
			if err := validateEstimateBoundary(first, counterexample); err != nil {
				t.Fatal(err)
			}
			proof["estimateMinusOneCounterexample"] = counterexample
			proof["originalExpectationObserved"] = !counterexample
			control := observe(contracts[2], baseline.Used-1)
			proof["actualUsedMinusOne"] = control
			if err := validateEstimateBoundary(control, false); err != nil {
				t.Fatal(err)
			}
			if control.Limit != baseline.Used-1 {
				t.Fatal("control did not use actual successful consumption minus one")
			}
			height, err := c.BlockNumber(ctx)
			if err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(60 * time.Second)
			for {
				now, err := c.BlockNumber(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if now > height {
					proof["continuedBlockBefore"], proof["continuedBlockAfter"] = height, now
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("blocks stopped after failed transaction")
				}
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(time.Second):
				}
			}
			proof["approvedFiniteGasCase"] = verifyFiniteGasCase(t, ctx, c, out.SessionRoot)
			proof["approvedTx018CriterionComplete"] = true
			t.Logf("estimate=%d baselineUsed=%d estimateMinusOne=%d status=%s used=%d counterexample=%v controlLimit=%d status=%s state=%s", estimate, baseline.Used, first.Limit, first.Status, first.Used, counterexample, control.Limit, control.Status, control.After)
		})
	}
}

// verifyFiniteGasCase checks mined input independently of the DSL assertions.
func verifyFiniteGasCase(t *testing.T, ctx context.Context, c *rpc.Client, sessionRoot string) map[string]any {
	t.Helper()
	dir := filepath.Join(sessionRoot, "tests/001_finite-gas-boundary")
	data, err := os.ReadFile(filepath.Join(dir, "steps.json"))
	if err != nil {
		t.Fatal(err)
	}
	var steps []session.StepResult
	if err := json.Unmarshal(data, &steps); err != nil {
		t.Fatal(err)
	}
	calls := []map[string]any{}
	used := uint64(0)
	for _, step := range steps {
		if step.Hash == "" {
			continue
		}
		var tx map[string]any
		if err := c.Call(ctx, "eth_getTransactionByHash", &tx, step.Hash); err != nil {
			t.Fatal(err)
		}
		if tx["input"] != "0x01" {
			continue
		}
		receipt := step.Receipt
		if tx["hash"] != step.Hash || tx["hash"] != receipt["transactionHash"] || tx["blockHash"] != receipt["blockHash"] {
			t.Fatal("DSL case mined facts differ", tx, receipt)
		}
		gas, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(tx["gas"]), "0x"), 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		consumed, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(receipt["gasUsed"]), "0x"), 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		var trace failedExecutionTrace
		if err := c.Call(ctx, "debug_traceTransaction", &trace, step.Hash, map[string]any{"disableStack": true, "disableMemory": true, "disableStorage": true}); err != nil {
			t.Fatal(err)
		}
		if len(calls) == 0 {
			if receipt["status"] != "0x1" || trace.Failed || consumed <= 21016 || consumed > gas {
				t.Fatal("invalid DSL successful baseline")
			}
			used = consumed
		} else if len(calls) == 1 {
			if receipt["status"] != "0x0" || gas != used-1 || consumed != gas || !traceWroteBeforeFailure(trace) {
				t.Fatal("invalid DSL actual-used-minus-one rollback")
			}
		} else {
			t.Fatal("unexpected additional DSL write transaction")
		}
		calls = append(calls, map[string]any{"transaction": tx, "receipt": receipt, "trace": trace})
	}
	if len(calls) != 2 {
		t.Fatal("DSL case lacks two actual write transactions", len(calls))
	}
	return map[string]any{"steps": filepath.Join(dir, "steps.json"), "assertions": filepath.Join(dir, "assert.json"), "calls": calls}
}
