package testengine_test

import (
	"context"
	"encoding/hex"
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
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
	"golang.org/x/crypto/sha3"
)

func TestNativeLegacyGovMinterCodeAndState(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native legacy code/state proof requires GSTABLE_BIN")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()
	base := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/post-v1.0.0-change/common-all")
	names := []string{"02-govminter-v2-code.json", "10-prealloc-preserved-across-boho.json"}
	raws := [][]byte{}
	for i, name := range names {
		b, e := os.ReadFile(filepath.Join(base, name))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, fmt.Sprintf("source-case%d.json", i+1)), b, 0600); e != nil {
			t.Fatal(e)
		}
		loaded, e := dsl.ReadFiles([]string{filepath.Join(base, name)})
		if e != nil {
			t.Fatal(e)
		}
		raws = append(raws, loaded...)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: binary, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2, LaunchOpts: []string{"cache=64"}})
	proof := map[string]any{"scope": "Recovered legacy TC-5-2-05 and TC-1-1-11/12 predicates on actual current StableNet BP4, Boho10 with explicit cache64 command override. Public case identities, exact v1/v2 Keccak runtime code hashes at9/10, GovMinter balance and four literal source storage slots preserved, v2 refundableBalance zero-address call. Existing prealloc balance/nonce conditions retained. Not full254/87/Linux/mainnet/final Web acceptance.", "binary": binary, "session": out.SessionRoot, "summary": out.Summary}
	defer func() {
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/legacy-code-state-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	if runErr != nil {
		proof["runError"] = runErr.Error()
		t.Fatal(runErr)
	}
	if out.Summary.Summary.Pass != 2 || out.Summary.Summary.Fail != 0 || out.Summary.Summary.Blocked != 0 || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 2 {
		t.Fatal("not exact two public passes", out.Summary)
	}
	for i, id := range []string{"govminter-v2-code", "prealloc-preserved-across-boho"} {
		if out.Summary.Tests[i].ID != id || out.Summary.Tests[i].Status != "pass" {
			t.Fatal("wrong public case identity", out.Summary.Tests)
		}
	}
	state, e := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
	if e != nil || len(state.Nodes.Nodes) != 4 {
		t.Fatal("no actual BP4", e)
	}
	client := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
	address := "0x0000000000000000000000000000000000001003"
	observed := map[string]any{}
	for _, sample := range []struct{ version, block, want string }{{"v1", "0x9", "0x3ea99dd77ff9bbc4a88b126e9aa06ddbf144f535a5b432040dbedadd66f3c759"}, {"v2", "0xa", "0xf369fcc4edab330169f427fd2aedbe97432cf1cfa31200da06464034f3edaa05"}} {
		var code, remote string
		if e = client.Call(ctx, "eth_getCode", &code, address, sample.block); e != nil {
			t.Fatal(e)
		}
		b, e := hex.DecodeString(strings.TrimPrefix(code, "0x"))
		if e != nil || len(b) <= 100 {
			t.Fatal("missing runtime code", e)
		}
		h := sha3.NewLegacyKeccak256()
		_, _ = h.Write(b)
		actual := "0x" + hex.EncodeToString(h.Sum(nil))
		if e = client.Call(ctx, "web3_sha3", &remote, code); e != nil {
			t.Fatal(e)
		}
		if actual != sample.want || remote != actual {
			t.Fatal("runtime code hash differs", sample.version, actual, remote)
		}
		var balance string
		if e = client.Call(ctx, "eth_getBalance", &balance, address, sample.block); e != nil {
			t.Fatal(e)
		}
		var block map[string]any
		if e = client.Call(ctx, "eth_getBlockByNumber", &block, sample.block, false); e != nil {
			t.Fatal(e)
		}
		if block["number"] != sample.block {
			t.Fatal("wrong boundary block", block)
		}
		observed[sample.version] = map[string]any{"blockTag": sample.block, "code": code, "codeBytes": len(b), "localKeccak256": actual, "rpcKeccak256": remote, "balance": balance, "header": block}
	}
	v1 := observed["v1"].(map[string]any)
	v2 := observed["v2"].(map[string]any)
	if v1["balance"] != v2["balance"] || v1["header"].(map[string]any)["hash"] != v2["header"].(map[string]any)["parentHash"] {
		t.Fatal("boundary balance or block links differ")
	}
	slots := []string{"0x0000000000000000000000000000000000000000000000000000000000000004", "0x000000000000000000000000000000000000000000000000000000000000000c", "0x0000000000000000000000000000000000000000000000000000000000000032", "0x80497882cf9008f7f796a89e5514a7b55bd96eab88ecb66aee4fb0a6fd34811c"}
	values := map[string]any{}
	for _, slot := range slots {
		var before, after string
		if e = client.Call(ctx, "eth_getStorageAt", &before, address, slot, "0x9"); e != nil {
			t.Fatal(e)
		}
		if e = client.Call(ctx, "eth_getStorageAt", &after, address, slot, "0xa"); e != nil {
			t.Fatal(e)
		}
		if len(before) != 66 || before != after {
			t.Fatal("literal legacy slot changed", slot, before, after)
		}
		values[slot] = map[string]string{"before": before, "after": after}
	}
	var getter string
	data := "0xb03d36cd" + strings.Repeat("0", 64)
	if e = client.Call(ctx, "eth_call", &getter, map[string]string{"to": address, "data": data}, "0xa"); e != nil {
		t.Fatal(e)
	}
	if getter != "0x"+strings.Repeat("0", 64) {
		t.Fatal("unexpected v2 zero-address refundable balance", getter)
	}
	proof["runtimeCode"] = observed
	proof["storageSlots"] = values
	proof["refundableBalance"] = map[string]string{"input": data, "block": "0xa", "output": getter}
}
