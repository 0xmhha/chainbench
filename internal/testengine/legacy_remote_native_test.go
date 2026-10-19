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

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// An owned attach endpoint verifies the original remote predicates without
// claiming to have reached an external testnet or changing its account store.
func TestNativeLegacyRemotePredicates(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("requires GSTABLE_BIN")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	bootstrap := filepath.Join(repoRoot(t), "tests/tc/common/node/CT-NODE-003-genesis-init.json")
	raws, err := dsl.ReadFiles([]string{bootstrap})
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(root, "w")
	input := testengine.RunSuiteIn{SpecContent: raws, DataDir: workspace, ArtifactRoot: filepath.Join(root, "bootstrap"), Binary: binary, KeysSource: "generate", KeysDir: filepath.Join(workspace, "keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2, LaunchOpts: []string{"cache=64"}}
	up, plan, err := testengine.PlanSuiteLayout(ctx, input)
	if err != nil || up.KeysSource != "generate" || plan.Keys.Dir != input.KeysDir {
		t.Fatal("keys must belong to this fixture", err)
	}
	initial, err := testengine.RunSuite(ctx, chainsetup.Deps{}, input)
	if err != nil || initial.Summary.Summary.Pass != 1 || initial.Summary.Failed() {
		t.Fatal("owned bootstrap failed", err, initial.Summary)
	}
	status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil || len(status.Nodes.Nodes) != 4 {
		t.Fatal("missing owned BP4", err)
	}
	endpoint := status.Nodes.Nodes[0].RPCURL
	c := rpc.Dial(endpoint)
	proof := map[string]any{"scope": "Four public legacy remote cases against a freshly composed owned StableNet BP4 endpoint. Attach does not control nodes. Real external testnet, original topology and all legacy/L1/Web remain unaccepted.", "endpoint": endpoint, "bootstrapSession": initial.SessionRoot, "bootstrapSummary": initial.Summary, "generatedKeysDir": plan.Keys.Dir, "all14Accepted": false, "actualTestnetAccepted": false}
	defer func() {
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/legacy-remote-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	privateKey, sender, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	_, recipient, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	var imported string
	if err = c.Call(ctx, "personal_importRawKey", &imported, hex.EncodeToString(privateKey), "chainbench-owned-remote"); err != nil || !strings.EqualFold(imported, sender) {
		t.Fatal("owned key import failed", err)
	}
	clear(privateKey)
	var unlocked bool
	if err = c.Call(ctx, "personal_unlockAccount", &unlocked, sender, "chainbench-owned-remote", uint64(120)); err != nil || !unlocked {
		t.Fatal("owned account unlock failed", err)
	}
	var coinbase, fundingHash string
	if err = c.Call(ctx, "eth_coinbase", &coinbase); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(ctx, "eth_sendTransaction", &fundingHash, map[string]any{"from": coinbase, "to": sender, "value": "0x56bc75e2d63100000", "gas": "0x5208"}); err != nil {
		t.Fatal("owned funding failed", err)
	}
	var funding map[string]any
	for funding == nil {
		if err = c.Call(ctx, "eth_getTransactionReceipt", &funding, fundingHash); err != nil {
			t.Fatal(err)
		}
		if funding != nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
	if funding["status"] != "0x1" {
		t.Fatal("funding not mined successfully", funding)
	}
	proof["sender"], proof["recipient"], proof["imported"], proof["unlocked"], proof["fundingReceipt"] = sender, recipient, imported, unlocked, funding
	var specs [][]byte
	names := []string{"06-remote-balance-query.json", "07-remote-chain-info.json", "08-remote-rpc-health.json", "09-remote-unlocked-transfer.json"}
	for _, name := range names {
		path := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/testnet", name)
		source, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		var doc map[string]any
		if e = json.Unmarshal(source, &doc); e != nil {
			t.Fatal(e)
		}
		// Only endpoint and input values change. Original public statements are
		// preserved, including the node-managed eth_sendTransaction path.
		doc["chainPreset"] = map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "owned-legacy-remote", "chain": "stablenet", "attach": map[string]any{"rpc": []string{endpoint}, "provides": []string{"rpc"}}, "values": map[string]any{"queryAddress": recipient, "remoteSender": sender, "remoteRecipient": recipient}}
		b, e := json.MarshalIndent(doc, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, name), source, 0600); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, "resolved-"+name), b, 0600); e != nil {
			t.Fatal(e)
		}
		resolved, e := dsl.ReadContent(b, filepath.Dir(path))
		if e != nil {
			t.Fatal(e)
		}
		specs = append(specs, resolved)
	}
	sessionRoot, runErr := testengine.AttachWorkspaceRun(ctx, chainsetup.Deps{}, testengine.AttachWorkspaceIn{DataDir: workspace, Chain: "stablenet", ArtifactRoot: filepath.Join(root, "reports"), Specs: specs, ReadOnlyKeys: true, NoNodeControl: true})
	proof["session"] = sessionRoot
	summary, err := testengine.ReadSessionSummary(sessionRoot)
	proof["summary"] = summary
	if runErr != nil || err != nil || summary.Summary.Pass != 4 || summary.Failed() || summary.Summary.Skip != 0 {
		t.Fatal("remote public cases failed", runErr, err, summary)
	}
	assertions := []session.AssertResult{}
	var hash string
	for i, row := range summary.Tests {
		p := filepath.Join(sessionRoot, "tests", fmt.Sprintf("%03d_%s", i+1, row.ID), "assert.json")
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		var actual []session.AssertResult
		if e = json.Unmarshal(b, &actual); e != nil {
			t.Fatal(e)
		}
		for _, a := range actual {
			if !a.Pass {
				t.Fatal("nonpassing public assertion", a)
			}
			if a.Assert == "txStatus" {
				hash, _ = a.Provenance["hash"].(string)
			}
		}
		assertions = append(assertions, actual...)
	}
	if hash == "" {
		t.Fatal("missing exact submitted transaction hash")
	}
	var tx, receipt, block map[string]any
	var raw string
	if err = c.Call(ctx, "eth_getTransactionByHash", &tx, hash); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(ctx, "eth_getBlockByNumber", &block, receipt["blockNumber"], false); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(ctx, "eth_getRawTransactionByHash", &raw, hash); err != nil {
		t.Fatal(err)
	}
	if tx["hash"] != hash || receipt["transactionHash"] != hash || receipt["status"] != "0x1" || tx["value"] != "0x1" || tx["from"] != strings.ToLower(sender) || tx["to"] != strings.ToLower(recipient) || block["hash"] != receipt["blockHash"] {
		t.Fatal("remote transaction witness disagrees", tx, receipt, block)
	}
	proof["transaction"], proof["receipt"], proof["block"], proof["raw"], proof["assertions"] = tx, receipt, block, raw, assertions
}
