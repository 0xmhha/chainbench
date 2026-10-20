//go:build e2e

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixRegisteredEpochE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	t.Setenv("GWEMIX_BIN", from)
	t.Setenv("GWBFT_BIN", to)
	root := createHandoffRuntime(t, writeGovernanceOriginalForkCase(t))
	keys := filepath.Join(root, "keyring")
	n := 5
	// WEMIX deploy-governance prompts when its password file is empty.
	if _, err := app.KeyringNew(t.Context(), app.Deps{}, app.RingCreateIn{Ring: app.RingRef{Dir: keys}, Count: 6, Validators: &n, WithBLS: true, Password: "1"}); err != nil {
		t.Fatal(err)
	}
	type ringMetadata struct {
		Nodes []struct {
			Index                         int
			Address, BLSPublicKey, BLSPoP string
		}
	}
	metadata := readPostforkPublicJSON[ringMetadata](t, filepath.Join(keys, "metadata.json"))
	if len(metadata.Nodes) != 6 {
		t.Fatal("not six generated identities")
	}
	doc := readPostforkPublicJSON[map[string]any](t, filepath.Join(root, "case.json"))
	preset := doc["chainPreset"].(map[string]any)
	preset["keys"].(map[string]any)["nodekeys"].(map[string]any)["ref"] = keys
	topology := []any{}
	for i := 1; i <= 6; i++ {
		row := map[string]any{"index": i, "role": "en", "binary": "next"}
		if i == 6 {
			row = map[string]any{"index": i, "role": "bp"}
		}
		topology = append(topology, row)
	}
	preset["topology"] = map[string]any{"nodes": topology}
	genesis := preset["genesis"].(map[string]any)["overlay"].(map[string]any)
	croissant := genesis["config"].(map[string]any)["croissant"].(map[string]any)
	// Two seconds preserves the original coordinates while allowing the caller
	// to hold block102 after CLI completion, before signing the block103 tx.
	croissant["wBFT"] = map[string]any{"epochLength": 10, "blockPeriodSeconds": 2, "targetValidators": 4, "stabilizingStakersThreshold": 4, "useNCP": false}
	alloc := genesis["alloc"].(map[string]any)
	for _, m := range metadata.Nodes {
		alloc[m.Address] = map[string]any{"balance": handoffBalance}
	}
	operatorKeys := make([][]byte, 4)
	operators := make([]string, 4)
	for i := range operatorKeys {
		var err error
		operatorKeys[i], operators[i], err = accounts.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		alloc[operators[i]] = map[string]any{"balance": handoffBalance}
	}
	spec := filepath.Join(root, "registered-case.json")
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(spec, b, 0600); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(root, "w")
	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", spec, "--workspace-dir", workspace, "--artifact-root", filepath.Join(root, "reports"), "--keys", keys, "--read-only-keys", "--no-skips", "--keep-up", "--node-monitor-timeout", "5m"})
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	err = cmd.Execute()
	preserveHandoffRun(t, root, output.String(), err)
	if err != nil || !strings.Contains(output.String(), "fail=0") {
		t.Fatal("original registered epoch handoff failed", err, output.String())
	}
	url := successorRPC(t, workspace)
	c := rpc.Dial(url)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil {
		t.Fatal(err)
	}
	nodes := []node.Node{}
	for _, v := range state.Nodes.Nodes {
		if v.Chain == "wbft" {
			nodes = append(nodes, v)
		}
	}
	if len(nodes) != 5 {
		t.Fatal("not five recorded successor identities")
	}
	for i, v := range nodes {
		if v.Index != i+1 || v.Role != node.RoleBP || v.Host != "127.0.0.1" || v.PID <= 0 {
			t.Fatal("not owned successor", v)
		}
		var address string
		if err := rpc.Dial(v.RPCURL).Call(ctx, "istanbul_nodeAddress", &address); err != nil || !strings.EqualFold(address, metadata.Nodes[i].Address) {
			t.Fatal("live identity differs", err, address)
		}
	}
	p := registeredEpochProof{Fork: 100, EpochLength: 10}
	proof := map[string]any{"workspace": workspace, "successors": nodes, "scope": "WBFT005/GOV005 original Croissant100/epoch10/first registration103/epoch110/next111. Generated five initial seed validators, four newly registered GovStaking stakers with exact original minimum1e25, threshold4/target4/useNCPfalse; next epoch excludes fifth unregistered seed and has four registered identities/keys. Registered stakers already belonged to initial seed set; no promotion from outside seed claimed. Fixture uses2s block period. Not public JSON entire run/Linux/mainnet/full87/legacy/final Web."}
	observed := &observedRawRPC{client: c}
	defer func() {
		proof["witness"], proof["submittedRaws"] = p, observed.submitted()
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/registered-epoch-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	var controls []lifecycleControl
	p.HeldHeight, controls = pauseWSProduction(t, nodes)
	proof["paused"] = controls
	if p.HeldHeight != 102 {
		t.Fatal("missed original registration parent102", p.HeldHeight)
	}
	for i := 0; i < 5; i++ {
		p.Initial = append(p.Initial, epochBoundaryMember{Address: metadata.Nodes[i].Address, BLS: metadata.Nodes[i].BLSPublicKey})
	}
	if err := c.Call(ctx, "istanbul_getValidators", &p.Before, "0x66"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		var before string
		if err := c.Call(ctx, "eth_call", &before, map[string]any{"to": e2eGovStaking, "data": accounts.EncodeCallArgs("stakerInfo(address)", accounts.Address(metadata.Nodes[i].Address))}, "0x66"); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(before, "0x") || len(before) < 2+6*64 {
			t.Fatal("short prior stakerInfo", before)
		}
		proof[fmt.Sprintf("stakerInfoAt102-%d", i)] = before
		p.BeforeStaked = append(p.BeforeStaked, "0x"+before[2+5*64:2+6*64])
	}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	amount, _ := new(big.Int).SetString(governanceForkFixture().Configured[0], 10)
	var tipHex, priceHex string
	if err := c.Call(ctx, "eth_maxPriorityFeePerGas", &tipHex); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(ctx, "eth_gasPrice", &priceHex); err != nil {
		t.Fatal(err)
	}
	tip, validTip := lifecycleQuantity(tipHex)
	price, validPrice := lifecycleQuantity(priceHex)
	if !validTip || !validPrice || tip.Sign() <= 0 || price.Sign() <= 0 {
		t.Fatal("invalid suggested transaction fees")
	}
	cap := new(big.Int).Add(new(big.Int).Mul(price, big.NewInt(2)), tip)
	proof["suggestedTip"], proof["suggestedPrice"] = tipHex, priceHex
	for i := 0; i < 4; i++ {
		m := metadata.Nodes[i]
		pk, e := hex.DecodeString(strings.TrimPrefix(m.BLSPublicKey, "0x"))
		if e != nil {
			t.Fatal(e)
		}
		pop, e := hex.DecodeString(strings.TrimPrefix(m.BLSPoP, "0x"))
		if e != nil {
			t.Fatal(e)
		}
		wallet, e := provider.OpenWallet(ctx, operatorKeys[i], url)
		if e != nil {
			t.Fatal(e)
		}
		input := accounts.EncodeCallArgs("registerStaker(uint256,address,address,uint256,bytes,bytes)", accounts.Uint(amount), accounts.Address(m.Address), accounts.Address(m.Address), accounts.Uint(big.NewInt(0)), accounts.Bytes(pk), accounts.Bytes(pop))
		raw, e := hex.DecodeString(input[2:])
		if e != nil {
			t.Fatal(e)
		}
		zero := uint64(0)
		hash, e := wallet.SendDynamicFeeTx(ctx, accounts.DynamicTxArgs{ToHex: e2eGovStaking, Value: amount, Data: raw, Gas: 2000000, GasFeeCap: cap, GasTipCap: tip, Nonce: &zero})
		if e != nil {
			t.Fatal(e)
		}
		if i == 0 {
			// The only queued transaction is the first registration. Production
			// stays stopped until its exact hash is visible on every successor.
			deadline := time.Now().Add(10 * time.Second)
			pending := make([]map[string]any, len(nodes))
			proof["pendingBeforeResume"] = pending
			for j, v := range nodes {
				for {
					var tx map[string]any
					if e := rpc.Dial(v.RPCURL).Call(ctx, "eth_getTransactionByHash", &tx, hash); e != nil {
						t.Fatal(e)
					}
					pending[j] = map[string]any{"node": v.Index, "hash": hash, "transaction": tx}
					if tx["hash"] == hash && tx["blockNumber"] == nil {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("first registration not pending on every successor")
					}
					time.Sleep(50 * time.Millisecond)
				}
			}
			resumed := []lifecycleControl{}
			for _, v := range nodes {
				resumed = append(resumed, lifecycleMinerControl(t, v, "miner_start", true))
			}
			proof["resumed"] = resumed
		}
		_ = policyReceipt(t, c, hash)
		w := registrationWitness{Hash: hash, PoP: m.BLSPoP, Reward: rewardLedgerProof{Staker: m.Address, Operator: wallet.Address(), Minimum: amount.String(), BLSKey: m.BLSPublicKey}}
		// Keep even a partial row if a pinned state read fails below.
		p.Registrations = append(p.Registrations, w)
		row := &p.Registrations[len(p.Registrations)-1]
		if err := c.Call(ctx, "eth_getTransactionByHash", &row.Transaction, hash); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getTransactionReceipt", &row.Receipt, hash); err != nil {
			t.Fatal(err)
		}
		tag := fmt.Sprint(row.Receipt["blockNumber"])
		height, validHeight := lifecycleQuantity(tag)
		if !validHeight || height.Uint64() >= 110 || (i == 0 && height.Uint64() != 103) {
			t.Fatal("registration outside original interval", tag)
		}
		callAtReceipt := func(sig, address string) string {
			var out string
			if err := c.Call(ctx, "eth_call", &out, map[string]any{"to": e2eGovStaking, "data": accounts.EncodeCallArgs(sig, accounts.Address(address))}, tag); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(out, "0x") || len(out) < 66 {
				t.Fatal("short pinned state", sig, out)
			}
			return out
		}
		row.Active = callAtReceipt("isStaker(address)", m.Address) == "0x"+fmt.Sprintf("%064x", 1)
		mapped := callAtReceipt("stakerByOperator(address)", wallet.Address())
		row.Mapped = "0x" + mapped[len(mapped)-40:]
		info := callAtReceipt("stakerInfo(address)", m.Address)
		if len(info) < 2+6*64 {
			t.Fatal("short pinned stakerInfo")
		}
		proof[fmt.Sprintf("stakerInfoAfter-%d", i)] = info
		p.AfterStaked = append(p.AfterStaked, "0x"+info[2+5*64:2+6*64])
		row.Reward.Rewardee = "0x" + info[2+64+24:2+2*64]
		row.Reward.Logs = feeReceipt(t, c, hash)
		if err := c.Call(ctx, "eth_getCode", &row.BeforeCode, row.Reward.Rewardee, hexUint(height.Uint64()-1)); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getCode", &row.Code, row.Reward.Rewardee, tag); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getStorageAt", &row.Implementation, row.Reward.Rewardee, "0x4ef8d65ed4f969898f05d331f7b880c9611386779b412e35e117f26e0983c85d", tag); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getCode", &row.ImplementationCode, "0x0000000000000000000000000000000000001002", tag); err != nil {
			t.Fatal(err)
		}
		if err := verifyRegistrationWitness(*row); err != nil {
			t.Fatal(err)
		}
	}
	waitForHead(t, c, 112, 90*time.Second)
	for i := 0; i < 3; i++ {
		tag := hexUint(uint64(109 + i))
		if err := c.Call(ctx, "eth_getBlockByNumber", &p.Blocks[i], tag, false); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &p.Extras[i], tag); err != nil {
			t.Fatal(err)
		}
	}
	// The first epoch still uses all five seed validators at109. The next
	// epoch111 must use precisely the four newly registered stakers instead.
	if err := c.Call(ctx, "istanbul_getValidators", &p.Before, "0x6d"); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(ctx, "istanbul_getValidators", &p.After, "0x6f"); err != nil {
		t.Fatal(err)
	}
	if err := verifyRegisteredEpoch(p); err != nil {
		t.Fatal(err)
	}
}
