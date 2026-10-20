package testengine_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"golang.org/x/crypto/sha3"
)

func TestNativeLegacyPersonalDelegation(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native personal API proof requires GSTABLE_BIN")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	var specs [][]byte
	names := []string{"01b-personal-normal.json", "02b-personal-sender.json", "03b-personal-feepayer.json", "04b-personal-insufficient.json"}
	for _, name := range names {
		p := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/regression/fee-delegation", name)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, name), b, 0600); err != nil {
			t.Fatal(err)
		}
		raw, err := dsl.ReadFiles([]string{p})
		if err != nil {
			t.Fatal(err)
		}
		specs = append(specs, raw...)
	}
	input := testengine.RunSuiteIn{SpecContent: specs, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: binary, ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2, LaunchOpts: []string{"cache=64"}}
	up, plan, err := testengine.PlanSuiteLayout(ctx, input)
	if err != nil || up.KeysSource != "generate" || plan.Keys.Source != "generate" || plan.Keys.Dir != filepath.Join(root, "w", "keys") || up.KeysDir != plan.Keys.Dir {
		t.Fatal("refuse personal import outside the owned workspace", plan.Keys, err)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, input)
	proof := map[string]any{"binary": binary, "plannedKeysDir": plan.Keys.Dir, "session": out.SessionRoot, "summary": out.Summary, "scope": "Four public personal-import/unlock/sign/sign-fee-delegate cases on owned StableNet BP4/EOAs, not original EN topology. Additional repeat raw submissions observe exact signature refusal and non-inclusion/state invariance; not all legacy/87/L1/Web or external issue resolution."}
	defer func() {
		b, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err = os.WriteFile(filepath.Join(root, "reports/legacy-personal-proof.json"), b, 0600); err != nil {
			t.Error(err)
		}
	}()
	if runErr != nil {
		proof["runError"] = runErr.Error()
		t.Fatal(runErr)
	}
	if out.Summary.Summary.Pass != 4 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 4 {
		t.Fatal("expected four exact public passes", out.Summary)
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
	if err != nil || len(state.Nodes.Nodes) != 4 {
		t.Fatal("missing actual BP4", err)
	}
	c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
	observations := []map[string]any{}
	for i, mode := range []string{"normal", "sender", "feepayer", "insufficient"} {
		id := "legacy-personal-fd-" + mode
		if out.Summary.Tests[i].ID != id || out.Summary.Tests[i].Status != "pass" {
			t.Fatal("wrong public case identity", out.Summary.Tests[i])
		}
		p := filepath.Join(out.SessionRoot, "tests", fmt.Sprintf("%03d_%s", i+1, id), "assert.json")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var assertions []session.AssertResult
		if err = json.Unmarshal(b, &assertions); err != nil {
			t.Fatal(err)
		}
		var senderRaw, payerRaw, acceptedHash string
		var unlocks []string
		for _, a := range assertions {
			if !a.Pass {
				t.Fatal("non-passing actual assertion", a)
			}
			params, _ := a.Provenance["params"].([]any)
			switch a.Provenance["method"] {
			case "personal_unlockAccount":
				unlocks = append(unlocks, fmt.Sprint(params[0]))
			case "web3_sha3":
				raw, _ := params[0].(string)
				if strings.HasPrefix(raw, "0x02") {
					senderRaw = raw
				} else if strings.HasPrefix(raw, "0x16") {
					payerRaw = raw
					if a.Expected == a.Actual {
						acceptedHash, _ = a.Actual.(string)
					}
				}
			}
		}
		if senderRaw == "" || payerRaw == "" || len(unlocks) != 2 || unlocks[0] == unlocks[1] {
			t.Fatal("missing actual personal signing/unlock output", id)
		}
		observed := map[string]any{"id": id, "assertions": p, "senderRaw": senderRaw, "payerRaw": payerRaw, "sender": unlocks[0], "payer": unlocks[1]}
		observations = append(observations, observed)
		proof["observations"] = observations
		if mode != "sender" && mode != "feepayer" {
			var tx, receipt map[string]any
			if err = c.Call(ctx, "eth_getTransactionByHash", &tx, acceptedHash); err != nil {
				t.Fatal(err)
			}
			if err = c.Call(ctx, "eth_getTransactionReceipt", &receipt, acceptedHash); err != nil {
				t.Fatal(err)
			}
			if acceptedHash == "" || tx["hash"] != acceptedHash || receipt["transactionHash"] != acceptedHash || receipt["status"] != "0x1" || tx["type"] != "0x16" || tx["blockHash"] != receipt["blockHash"] {
				t.Fatal("missing exact personal-signed mined transaction", id)
			}
			observed["transaction"], observed["receipt"] = tx, receipt
			continue
		}
		raw, err := hex.DecodeString(payerRaw[2:])
		if err != nil {
			t.Fatal(err)
		}
		changed, err := accounts.TamperFeeDelegatedRaw(raw, mode)
		if err != nil {
			t.Fatal(err)
		}
		h := sha3.NewLegacyKeccak256()
		_, _ = h.Write(changed)
		hash := fmt.Sprintf("0x%x", h.Sum(nil))
		height, err := c.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		before := map[string]string{}
		for _, addr := range unlocks {
			balance, err := c.BalanceAt(ctx, addr)
			if err != nil {
				t.Fatal(err)
			}
			before[addr] = balance.String()
		}
		var returned string
		submitErr := c.Call(ctx, "eth_sendRawTransaction", &returned, "0x"+hex.EncodeToString(changed))
		var nodeError *rpc.ServerError
		if returned != "" || !errors.As(submitErr, &nodeError) || nodeError == nil || nodeError.Method != "eth_sendRawTransaction" || nodeError.Code != -32000 || nodeError.Message != "fee delegation: invalid feePayer" {
			t.Fatal("not exact signature refusal", returned, submitErr)
		}
		lookups := []map[string]any{}
		for _, phase := range []string{"initial", "later"} {
			if phase == "later" {
				for {
					head, err := c.BlockNumber(ctx)
					if err != nil {
						t.Fatal(err)
					}
					if head >= height+3 {
						observed["afterHeight"] = head
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					case <-time.After(300 * time.Millisecond):
					}
				}
			}
			for _, method := range []string{"eth_getTransactionByHash", "eth_getTransactionReceipt"} {
				var result json.RawMessage
				if err = c.Call(ctx, method, &result, hash); err != nil || strings.TrimSpace(string(result)) != "null" {
					t.Fatal("rejected bytes entered node state", method, err)
				}
				lookups = append(lookups, map[string]any{"phase": phase, "method": method, "hash": hash, "result": string(result)})
			}
		}
		after := map[string]string{}
		for _, addr := range unlocks {
			balance, err := c.BalanceAt(ctx, addr)
			if err != nil || balance.String() != before[addr] {
				t.Fatal("rejected signature changed funded balance", addr, err)
			}
			after[addr] = balance.String()
		}
		nonce, err := c.NonceAt(ctx, unlocks[0])
		if err != nil || nonce != 0 {
			t.Fatal("rejected signature consumed sender nonce", err)
		}
		observed["raw"], observed["hash"], observed["returnedHash"] = "0x"+hex.EncodeToString(changed), hash, returned
		observed["error"], observed["beforeHeight"], observed["lookups"] = nodeError, height, lookups
		observed["balancesBefore"], observed["balancesAfter"], observed["nonceAfter"] = before, after, nonce
	}
}
