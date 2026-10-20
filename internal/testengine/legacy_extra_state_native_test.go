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
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
	"golang.org/x/crypto/sha3"
)

func TestNativeLegacyAccountExtraState(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native legacy extra proof requires GSTABLE_BIN")
	}
	for _, sample := range []struct{ name, file string }{{"params", "02-stablenet-account-extra.json"}, {"delayed", "07b-extra-state-across-delayed-boho.json"}} {
		if !t.Run(sample.name, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			path := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/post-v1.0.0-change/extra-state", sample.file)
			raw, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(root, "source-case.json"), raw, 0600); e != nil {
				t.Fatal(e)
			}
			raws, e := dsl.ReadFiles([]string{path})
			if e != nil {
				t.Fatal(e)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: binary, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2, LaunchOpts: []string{"cache=64"}})
			proof := map[string]any{"scope": "Recovered legacy TC-4-5-03/04 and10/11/12 on current StableNet BP4 cache64. Public02 preserves alloc-origin checks and adds params-only accounts; public07b pins blocks5/6 around explicit Boho6/v2, all six flags, exact extra and balances. Different local addresses/topology and Boho6 vs historical100; not full254/87/Linux/mainnet/final Web or external issue resolution.", "profile": sample.name, "binary": binary, "session": out.SessionRoot, "summary": out.Summary}
			defer func() {
				b, e := json.MarshalIndent(proof, "", "  ")
				if e != nil {
					t.Error(e)
					return
				}
				if e = os.WriteFile(filepath.Join(root, "reports/legacy-extra-proof.json"), b, 0600); e != nil {
					t.Error(e)
				}
			}()
			if runErr != nil {
				proof["runError"] = runErr.Error()
				t.Fatal(runErr)
			}
			if out.Summary.Summary.Pass != 1 || out.Summary.Summary.Fail != 0 || out.Summary.Summary.Blocked != 0 || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 1 {
				t.Fatal("not exact one public pass", out.Summary)
			}
			state, e := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if e != nil || len(state.Nodes.Nodes) != 4 {
				t.Fatal("missing actual BP4", e)
			}
			plugin, e := registry.Get("stablenet")
			if e != nil {
				t.Fatal(e)
			}
			manager := plugin.Manifest().SystemContracts["accountManager"]
			client := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
			accounts := []struct {
				address string
				bits    uint64
			}{{legacyAuthAccount, uint64(1) << 62}, {legacyBlackAccount, uint64(1) << 63}, {legacyDualAccount, uint64(3) << 62}}
			blocks := []string{"0x1"}
			if sample.name == "params" {
				accounts = append(accounts, struct {
					address string
					bits    uint64
				}{legacyParamAuth, uint64(1) << 62}, struct {
					address string
					bits    uint64
				}{legacyParamBlack, uint64(1) << 63})
			} else {
				blocks = []string{"0x5", "0x6"}
			}
			observations := []map[string]any{}
			for _, account := range accounts {
				for _, block := range blocks {
					var stateProof map[string]any
					if e = client.Call(ctx, "eth_getProof", &stateProof, account.address, []string{}, block); e != nil {
						t.Fatal(e)
					}
					if stateProof["extra"] != fmt.Sprintf("0x%x", account.bits) {
						t.Fatal("wrong exact extra", account.address, block, stateProof["extra"])
					}
					if account.address != legacyParamAuth && account.address != legacyParamBlack {
						if stateProof["balance"] != "0xde0b6b3a7640000" || stateProof["nonce"] != "0x0" {
							t.Fatal("alloc balance/nonce differs", account.address, block)
						}
					}
					calls := map[string]any{}
					for _, flag := range []struct {
						name, selector string
						mask           uint64
					}{{"authorized", "0xfe9fbb80", uint64(1) << 62}, {"blacklisted", "0xfe575a87", uint64(1) << 63}} {
						data := flag.selector + strings.Repeat("0", 24) + account.address[2:]
						var result string
						if e = client.Call(ctx, "eth_call", &result, map[string]string{"to": manager, "data": data}, block); e != nil {
							t.Fatal(e)
						}
						want := 0
						if account.bits&flag.mask != 0 {
							want = 1
						}
						if result != fmt.Sprintf("0x%064x", want) {
							t.Fatal("wrong account manager flag", account.address, block, flag.name, result)
						}
						calls[flag.name] = map[string]any{"input": data, "output": result, "expected": want}
					}
					observations = append(observations, map[string]any{"address": account.address, "block": block, "expectedExtra": fmt.Sprintf("0x%x", account.bits), "stateProof": stateProof, "calls": calls})
				}
			}
			proof["manager"] = manager
			proof["observations"] = observations
			if sample.name == "delayed" {
				codes := map[string]any{}
				for _, sample := range []struct{ version, block, hash string }{{"v1", "0x5", "0x3ea99dd77ff9bbc4a88b126e9aa06ddbf144f535a5b432040dbedadd66f3c759"}, {"v2", "0x6", "0xf369fcc4edab330169f427fd2aedbe97432cf1cfa31200da06464034f3edaa05"}} {
					var code string
					if e = client.Call(ctx, "eth_getCode", &code, plugin.Manifest().SystemContracts["govMinter"], sample.block); e != nil {
						t.Fatal(e)
					}
					b, e := hex.DecodeString(strings.TrimPrefix(code, "0x"))
					if e != nil || len(b) <= 100 {
						t.Fatal("missing fork runtime code", e)
					}
					h := sha3.NewLegacyKeccak256()
					_, _ = h.Write(b)
					actual := "0x" + hex.EncodeToString(h.Sum(nil))
					if actual != sample.hash {
						t.Fatal("fork not exact expected code", sample.block, actual)
					}
					var header map[string]any
					if e = client.Call(ctx, "eth_getBlockByNumber", &header, sample.block, false); e != nil {
						t.Fatal(e)
					}
					if header["number"] != sample.block {
						t.Fatal("wrong fork header")
					}
					codes[sample.version] = map[string]any{"block": sample.block, "code": code, "hash": actual, "header": header}
				}
				if codes["v1"].(map[string]any)["header"].(map[string]any)["hash"] != codes["v2"].(map[string]any)["header"].(map[string]any)["parentHash"] {
					t.Fatal("fork headers disconnected")
				}
				proof["forkCodes"] = codes
			}
		}) {
			t.Fatal("stop after the preceding proof or cleanup failed")
		}
	}
}
