package testengine_test

import (
	"context"
	"encoding/hex"
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
	"golang.org/x/crypto/sha3"
)

func TestNativeDelayedForkCodeCheckpoint(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native delayed-code proof requires GSTABLE_BIN")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()
	source := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "public-source.json"), body, 0600); err != nil {
		t.Fatal(err)
	}
	raws, err := dsl.ReadFiles([]string{source})
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(root, "w")
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: workspace, ArtifactRoot: filepath.Join(root, "reports"), Binary: binary, KeysSource: "generate", KeysDir: filepath.Join(workspace, "keys"), ReadOnlyKeys: true, KeepUp: true, LaunchOpts: []string{"cache=64"}})
	proof := map[string]any{"scope": "Public delayed-injection case with Boho3/wait6/history1 fixture; exact local/rpc Keccak v1/v2 runtime identities and P256 activation. Original Boho100/post101/history0 differs. Not full legacy/L1/Web acceptance.", "source": source, "binary": binary, "session": out.SessionRoot, "summary": out.Summary, "fullL1Accepted": false, "all14Accepted": false}
	defer func() {
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/delayed-code-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	if runErr != nil {
		proof["runError"] = runErr.Error()
		t.Fatal(runErr)
	}
	if out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 1 || out.Summary.Tests[0].ID != "stablenet-delayed-fork" {
		t.Fatal("missing exact public pass", out.Summary)
	}
	var assertions []session.AssertResult
	b, err := os.ReadFile(filepath.Join(out.SessionRoot, "tests/001_stablenet-delayed-fork/assert.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &assertions); err != nil || len(assertions) != 6 {
		t.Fatal("missing public assertions", err, len(assertions))
	}
	for _, a := range assertions {
		if !a.Pass {
			t.Fatal(a)
		}
	}
	proof["publicAssertions"] = assertions
	var genesis struct{ Config struct{ BohoBlock uint64 } }
	b, err = os.ReadFile(filepath.Join(workspace, "genesis.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &genesis); err != nil || genesis.Config.BohoBlock != 3 {
		t.Fatal("fixture is not Boho3", err, genesis)
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil || len(state.Nodes.Nodes) != 4 {
		t.Fatal("missing actual BP4", err)
	}
	var keys struct {
		Nodes []struct {
			Index   int
			Address string
		}
	}
	b, err = os.ReadFile(filepath.Join(workspace, "keys/metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &keys); err != nil {
		t.Fatal(err)
	}
	account := ""
	for _, n := range keys.Nodes {
		if n.Index == 1 {
			account = n.Address
		}
	}
	if account == "" {
		t.Fatal("generated node1 identity missing")
	}
	var sourceDoc struct {
		Steps []struct {
			Method string
			Params []json.RawMessage
		}
	}
	if err = json.Unmarshal(body, &sourceDoc); err != nil {
		t.Fatal(err)
	}
	var p256Call map[string]any
	for _, s := range sourceDoc.Steps {
		if s.Method == "eth_call" {
			if err = json.Unmarshal(s.Params[0], &p256Call); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if p256Call == nil {
		t.Fatal("public P256 activation vector missing")
	}
	observed := []map[string]any{}
	for _, n := range state.Nodes.Nodes {
		client := rpc.Dial(n.RPCURL)
		samples := []map[string]any{}
		for _, sample := range []struct{ tag, wantHash, wantCall string }{{"0x1", "0x3ea99dd77ff9bbc4a88b126e9aa06ddbf144f535a5b432040dbedadd66f3c759", "0x"}, {"0x6", "0xf369fcc4edab330169f427fd2aedbe97432cf1cfa31200da06464034f3edaa05", "0x" + strings.Repeat("0", 63) + "1"}} {
			var code, remote, output, balance string
			if err = client.Call(ctx, "eth_getCode", &code, "0x0000000000000000000000000000000000001003", sample.tag); err != nil {
				t.Fatal(err)
			}
			raw, e := hex.DecodeString(strings.TrimPrefix(code, "0x"))
			if e != nil || !strings.HasPrefix(code, "0x") || len(raw) < 50 {
				t.Fatal("missing runtime code", e)
			}
			h := sha3.NewLegacyKeccak256()
			_, _ = h.Write(raw)
			local := "0x" + hex.EncodeToString(h.Sum(nil))
			if err = client.Call(ctx, "web3_sha3", &remote, code); err != nil || local != sample.wantHash || remote != local {
				t.Fatal("wrong exact runtime identity", err, local, remote)
			}
			if err = client.Call(ctx, "eth_call", &output, p256Call, sample.tag); err != nil || output != sample.wantCall {
				t.Fatal("P256 activation differs", err, output)
			}
			if err = client.Call(ctx, "eth_getBalance", &balance, account, sample.tag); err != nil {
				t.Fatal(err)
			}
			samples = append(samples, map[string]any{"block": sample.tag, "code": code, "codeBytes": len(raw), "localKeccak256": local, "rpcKeccak256": remote, "p256Output": output, "preallocBalance": balance})
		}
		if samples[0]["code"] == samples[1]["code"] || samples[0]["preallocBalance"] != samples[1]["preallocBalance"] {
			t.Fatal("transition or prealloc preservation failed", n.Index)
		}
		observed = append(observed, map[string]any{"node": n.Index, "rpcURL": n.RPCURL, "samples": samples})
	}
	proof["generatedAccount"] = account
	proof["nodes"] = observed
	proof["independentCodeIdentityVerified"] = true
}
