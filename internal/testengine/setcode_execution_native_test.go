package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
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

// TestNativeSetCodeExecution verifies the public EIP-7702 case and actual
// authority/delegate storage against a compatible WBFT binary.
func TestNativeSetCodeExecution(t *testing.T) {
	binary := os.Getenv("GWBFT_BIN")
	if binary == "" {
		t.Skip("native set-code execution requires GWBFT_BIN")
	}
	root := failureRuntime(t)
	workspace := filepath.Join(root, "w")
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()
	input := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json")
	content, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "set-code-delegation.json"), content, 0600); err != nil {
		t.Fatal(err)
	}
	specs, err := dsl.ReadFilesWithChainPreset([]string{input}, filepath.Join(repoRoot(t), "presets/chain/wbft-bp4-en1.json"))
	if err != nil {
		t.Fatal(err)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: workspace, ArtifactRoot: filepath.Join(root, "reports"), Binary: binary, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
	proof := map[string]any{"binary": binary, "session": out.SessionRoot, "summary": out.Summary, "scope": "Public EIP-7702 delegated storage execution on a managed local WBFT network. Not all87, mainnet or final Web acceptance."}
	if runErr != nil {
		proof["runError"] = runErr.Error()
	}
	defer func() {
		b, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(root, "reports/setcode-execution-proof.json"), b, 0600); err != nil {
			t.Error(err)
		}
	}()
	if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatalf("public set-code case failed: %v %+v", runErr, out.Summary)
	}
	status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil || len(status.Nodes.Nodes) < 5 {
		t.Fatal("missing actual 4BP + EN environment", err)
	}
	c := rpc.Dial(status.Nodes.Nodes[0].RPCURL)
	dir := filepath.Join(out.SessionRoot, "tests/001_set-code-delegation")
	data, err := os.ReadFile(filepath.Join(dir, "steps.json"))
	if err != nil {
		t.Fatal(err)
	}
	var steps []session.StepResult
	if err := json.Unmarshal(data, &steps); err != nil {
		t.Fatal(err)
	}
	var delegate, authority string
	var setCode, write map[string]any
	var setReceipt, writeReceipt map[string]any
	for _, step := range steps {
		if step.Type == "deployContract" {
			delegate, _ = step.Receipt["contractAddress"].(string)
		}
		if step.Type == "sendSetCode" {
			if err := c.Call(ctx, "eth_getTransactionByHash", &setCode, step.Hash); err != nil {
				t.Fatal(err)
			}
			if err := c.Call(ctx, "eth_getTransactionReceipt", &setReceipt, step.Hash); err != nil {
				t.Fatal(err)
			}
		}
		if step.Type == "sendTx" && step.Receipt["status"] == "0x1" {
			var tx map[string]any
			if err := c.Call(ctx, "eth_getTransactionByHash", &tx, step.Hash); err != nil {
				t.Fatal(err)
			}
			if tx["input"] == "0x"+strings.Repeat("0", 62)+"2a" {
				write = tx
				writeReceipt = step.Receipt
				authority, _ = tx["to"].(string)
			}
		}
	}
	if delegate == "" || authority == "" || delegate == authority || setCode["type"] != "0x4" || setReceipt["status"] != "0x1" || write["hash"] != writeReceipt["transactionHash"] || write["blockHash"] != writeReceipt["blockHash"] {
		t.Fatal("missing actual deployment, set-code or authority call", delegate, authority, setCode, write)
	}
	marker, err := c.CodeAt(ctx, authority)
	if err != nil || marker != "0xef0100"+strings.TrimPrefix(delegate, "0x") {
		t.Fatal("actual delegation marker mismatch", marker, err)
	}
	var values []map[string]any
	for _, address := range []string{authority, delegate} {
		word, err := c.EthCall(ctx, address, "0x00")
		if err != nil {
			t.Fatal(err)
		}
		var slot string
		if err := c.Call(ctx, "eth_getStorageAt", &slot, address, "0x0", "latest"); err != nil {
			t.Fatal(err)
		}
		expected := "0x" + strings.Repeat("0", 64)
		if address == authority {
			expected = "0x" + strings.Repeat("0", 62) + "2a"
		}
		if word != expected || slot != expected {
			t.Fatal("wrong actual execution storage context", address, word, slot)
		}
		values = append(values, map[string]any{"address": address, "view": word, "storage": slot})
	}
	var authorization []any
	if raw, ok := setCode["authorizationList"].([]any); ok {
		authorization = raw
	}
	if len(authorization) != 1 {
		t.Fatal("actual set-code transaction missing authorization", setCode)
	}
	a, ok := authorization[0].(map[string]any)
	if !ok || !strings.EqualFold(fmt.Sprint(a["address"]), delegate) {
		t.Fatal("actual authorization target mismatch", authorization)
	}
	proof["delegate"] = delegate
	proof["authority"] = authority
	proof["marker"] = marker
	proof["setCodeTransaction"] = setCode
	proof["setCodeReceipt"] = setReceipt
	proof["writeTransaction"] = write
	proof["writeReceipt"] = writeReceipt
	proof["storageContexts"] = values
	proof["steps"] = filepath.Join(dir, "steps.json")
	proof["assertions"] = filepath.Join(dir, "assert.json")
}
