//go:build e2e

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func lifecycleMinerControl(t *testing.T, n node.Node, method string, want bool) lifecycleControl {
	t.Helper()
	c := rpc.Dial(n.RPCURL)
	// Current WBFT MinerAPI.Start has no argument despite its older comment.
	if err := c.Call(t.Context(), method, nil); err != nil {
		t.Fatal(method, n.Index, err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		var mining bool
		if err := c.Call(t.Context(), "eth_mining", &mining); err != nil {
			t.Fatal(err)
		}
		if mining == want {
			return lifecycleControl{Node: n.Index, Method: method, Mining: mining}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("miner control not observed", method, n.Index)
	return lifecycleControl{}
}

func lifecycleHeads(t *testing.T, nodes []node.Node) []uint64 {
	t.Helper()
	heads := []uint64{}
	for _, n := range nodes {
		h, err := rpc.Dial(n.RPCURL).BlockNumber(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		heads = append(heads, h)
	}
	return heads
}

func TestWemixTransactionLifecycleE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	key, sender, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	_, recipient, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	overlay := map[string]any{"alloc": map[string]any{sender: map[string]any{"balance": handoffBalance}}}
	workspace, url := runGovHandoffWorkspace(t, from, to, writeHandoffCase(t, overlay))
	root := filepath.Dir(workspace)
	c := rpc.Dial(url)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil {
		t.Fatal(err)
	}
	nodes := []node.Node{}
	for _, n := range state.Nodes.Nodes {
		if n.Chain == "wbft" {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) != 4 {
		t.Fatal("not four recorded WBFT successors")
	}
	for i, n := range nodes {
		if n.Index != i+1 || n.Role != node.RoleBP || n.Host != "127.0.0.1" || n.PID <= 0 {
			t.Fatal("not owned local successor", n)
		}
	}
	p := transactionLifecycleProof{Sender: sender, Recipient: recipient}
	proof := map[string]any{"workspace": workspace, "genesis": filepath.Join(workspace, "genesis.json"), "successors": nodes, "scope": "TX001 same actual locally signed type2 1WMX/21000 transfer after WEMIX-to-WBFT Croissant20. Controlled mining pause on four owned successors preserves P2P RPC; exact pending transaction and executable pool at all four endpoints before mining, then production resumes. Receipt-indexed inclusion in first resumed block, three further blocks, canonical agreement and exact value/base-plus-capped-tip fee ledger. New test wallet keys are never retained. Not proxy-topology/public DSL re-execution/full87/Linux/mainnet/legacy/final Web."}
	defer func() {
		proof["lifecycle"] = p
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/transaction-lifecycle-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	for _, n := range nodes {
		p.Paused = append(p.Paused, lifecycleMinerControl(t, n, "miner_stop", false))
	}
	// The owning fixture's cleanup stops all processes even if a later check fails.
	deadline := time.Now().Add(20 * time.Second)
	for {
		heads := lifecycleHeads(t, nodes)
		same := true
		for _, h := range heads {
			same = same && h == heads[0]
		}
		if same {
			p.HeldHeight = heads[0]
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("paused heads did not converge", heads)
		}
		time.Sleep(200 * time.Millisecond)
	}
	start := time.Now()
	p.StableHeads[0] = lifecycleHeads(t, nodes)
	time.Sleep(2100 * time.Millisecond)
	p.StableHeads[1] = lifecycleHeads(t, nodes)
	p.StableMillis = time.Since(start).Milliseconds()
	for _, row := range p.StableHeads {
		for _, h := range row {
			if h != p.HeldHeight {
				t.Fatal("mining pause did not hold production", p.StableHeads)
			}
		}
	}
	if err := c.Call(ctx, "eth_getBlockByNumber", &p.HeldBlock, hexUint(p.HeldHeight), false); err != nil {
		t.Fatal(err)
	}
	p.SenderBefore = balanceOf(t, c, sender).String()
	p.RecipientBefore = balanceOf(t, c, recipient).String()
	p.NonceBefore, err = c.NonceAt(ctx, sender)
	if err != nil {
		t.Fatal(err)
	}
	var priceHex, tipHex string
	if err := c.Call(ctx, "eth_gasPrice", &priceHex); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(ctx, "eth_maxPriorityFeePerGas", &tipHex); err != nil {
		t.Fatal(err)
	}
	price, ok := lifecycleQuantity(priceHex)
	if !ok {
		t.Fatal("invalid gas price")
	}
	tip, ok := lifecycleQuantity(tipHex)
	if !ok {
		t.Fatal("invalid tip")
	}
	cap := new(big.Int).Add(new(big.Int).Mul(price, big.NewInt(2)), tip)
	observed := &observedRawRPC{client: c}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := provider.OpenWallet(ctx, key, url)
	if err != nil {
		t.Fatal(err)
	}
	p.Hash, err = wallet.SendDynamicFeeTx(ctx, accounts.DynamicTxArgs{ToHex: recipient, Value: big.NewInt(1000000000000000000), Gas: 21000, GasFeeCap: cap, GasTipCap: tip, Nonce: &p.NonceBefore})
	if err != nil {
		t.Fatal(err)
	}
	raws := observed.submitted()
	if len(raws) != 1 {
		t.Fatal("normal transaction not submitted exactly once")
	}
	p.Raw = raws[0]
	deadline = time.Now().Add(30 * time.Second)
	for {
		rows := []lifecyclePending{}
		ready := true
		for _, n := range nodes {
			rpcClient := rpc.Dial(n.RPCURL)
			row := lifecyclePending{Node: n.Index}
			if err := rpcClient.Call(ctx, "eth_getTransactionByHash", &row.Transaction, p.Hash); err != nil {
				t.Fatal(err)
			}
			if err := rpcClient.Call(ctx, "txpool_contentFrom", &row.Pool, sender); err != nil {
				t.Fatal(err)
			}
			pending, ok := row.Pool["pending"].(map[string]any)
			var item map[string]any
			if ok {
				item, _ = pending["0"].(map[string]any)
			}
			if row.Transaction["hash"] != p.Hash || item["hash"] != p.Hash {
				ready = false
			}
			rows = append(rows, row)
		}
		p.Pending = rows
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("same unmined transfer did not propagate", p.Pending)
		}
		time.Sleep(200 * time.Millisecond)
	}
	for _, row := range p.Pending {
		if row.Transaction["blockNumber"] != nil || row.Transaction["blockHash"] != nil {
			t.Fatal("transfer already mined before pending observation")
		}
	}
	afterSubmitHeads := lifecycleHeads(t, nodes)
	for _, h := range afterSubmitHeads {
		if h != p.HeldHeight {
			t.Fatal("production advanced before resume", afterSubmitHeads)
		}
	}
	proof["headsAfterPending"] = afterSubmitHeads
	for _, n := range nodes {
		p.Resumed = append(p.Resumed, lifecycleMinerControl(t, n, "miner_start", true))
	}
	p.Receipt = policyReceipt(t, c, p.Hash)
	if err := c.Call(ctx, "eth_getTransactionByHash", &p.Transaction, p.Hash); err != nil {
		t.Fatal(err)
	}
	number, ok := lifecycleQuantity(p.Receipt["blockNumber"])
	if !ok || !number.IsUint64() {
		t.Fatal("invalid receipt height")
	}
	if err := c.Call(ctx, "eth_getBlockByNumber", &p.MinedBlock, p.Receipt["blockNumber"], false); err != nil {
		t.Fatal(err)
	}
	waitForHead(t, c, number.Uint64()+3, 90*time.Second)
	p.ConfirmedHeight, err = c.BlockNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p.SenderAfter = balanceOf(t, c, sender).String()
	p.RecipientAfter = balanceOf(t, c, recipient).String()
	p.NonceAfter, err = c.NonceAt(ctx, sender)
	if err != nil {
		t.Fatal(err)
	}
	canonical := []map[string]any{}
	for _, n := range nodes {
		rpcClient := rpc.Dial(n.RPCURL)
		var block, receipt map[string]any
		if err := rpcClient.Call(ctx, "eth_getBlockByNumber", &block, p.Receipt["blockNumber"], false); err != nil {
			t.Fatal(err)
		}
		if err := rpcClient.Call(ctx, "eth_getTransactionReceipt", &receipt, p.Hash); err != nil {
			t.Fatal(err)
		}
		if block["hash"] != p.MinedBlock["hash"] || receipt["transactionHash"] != p.Hash || receipt["blockHash"] != p.MinedBlock["hash"] || receipt["status"] != "0x1" {
			t.Fatal("confirmed endpoint disagrees", n.Index, block, receipt)
		}
		canonical = append(canonical, map[string]any{"node": n.Index, "block": block, "receipt": receipt})
	}
	proof["canonicalEndpoints"] = canonical
	if err := verifyTransactionLifecycle(p); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(fmt.Sprint(p.Transaction["from"]), sender) {
		t.Fatal("actual signature recovered a different sender")
	}
}
