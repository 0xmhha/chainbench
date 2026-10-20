//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

type observedWS struct {
	sub     *rpc.Subscription
	mu      sync.Mutex
	rows    []wsObservedPayload
	failure error
	done    chan struct{}
}

func observeWS(sub *rpc.Subscription) *observedWS {
	o := &observedWS{sub: sub, done: make(chan struct{})}
	go func() {
		defer close(o.done)
		for raw := range sub.Notifications() {
			at := time.Now().UnixNano()
			var payload map[string]any
			err := json.Unmarshal(raw, &payload)
			o.mu.Lock()
			if err != nil {
				o.failure = err
			} else {
				o.rows = append(o.rows, wsObservedPayload{ReceivedUnixNano: at, Payload: payload})
			}
			o.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	return o
}
func (o *observedWS) snapshot(t *testing.T) []wsObservedPayload {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.failure != nil {
		t.Fatal(o.failure)
	}
	return append([]wsObservedPayload(nil), o.rows...)
}
func (o *observedWS) finish(t *testing.T) []wsObservedPayload {
	t.Helper()
	_ = o.sub.Close()
	select {
	case <-o.done:
	case <-time.After(5 * time.Second):
		t.Fatal("WS collector did not close")
	}
	return o.snapshot(t)
}

func pauseWSProduction(t *testing.T, nodes []node.Node) (uint64, []lifecycleControl) {
	t.Helper()
	controls := []lifecycleControl{}
	for _, n := range nodes {
		controls = append(controls, lifecycleMinerControl(t, n, "miner_stop", false))
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		heads := lifecycleHeads(t, nodes)
		same := true
		for _, h := range heads {
			same = same && h == heads[0]
		}
		if same {
			time.Sleep(2100 * time.Millisecond)
			after := lifecycleHeads(t, nodes)
			stable := true
			for _, h := range after {
				stable = stable && h == heads[0]
			}
			if stable {
				return heads[0], controls
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("WS observation boundary not stable")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestWemixWebsocketWitnessE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	key, sender, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	workspace, url := runGovHandoffWorkspace(t, from, to, writeHandoffCase(t, map[string]any{"alloc": map[string]any{sender: map[string]any{"balance": handoffBalance}}}))
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
		if n.Index != i+1 || n.Role != node.RoleBP || n.Host != "127.0.0.1" || n.PID <= 0 || n.WSURL == "" {
			t.Fatal("missing owned local WS endpoint", n)
		}
	}
	observed := &observedRawRPC{client: c}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := provider.OpenWallet(ctx, key, url)
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := "../../tests/tc/common/rpc/CT-RPC-014-ws-subscribe-logs.json"
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "public-ws-log-case.json"), source, 0600); err != nil {
		t.Fatal(err)
	}
	var sourceCase map[string]any
	if err := json.Unmarshal(source, &sourceCase); err != nil {
		t.Fatal(err)
	}
	initHex := sourceCase["steps"].([]any)[0].(map[string]any)["data"].(string)
	initCode, err := hex.DecodeString(strings.TrimPrefix(initHex, "0x"))
	if err != nil || len(initCode) != 51 {
		t.Fatal("unexpected public LOG1 fixture", err)
	}
	p := websocketWitness{Topic: "0x" + strings.Repeat("11", 32)}
	proof := map[string]any{"workspace": workspace, "genesis": filepath.Join(workspace, "genesis.json"), "sender": sender, "wsEndpoint": nodes[1], "initCode": initHex, "scope": "RPC020/021 local WEMIX-to-WBFT Croissant20, actual production-bounded WS window. Exact newHeads count versus canonical generated range with all number/hash/parent links. Public LOG1 runtime reused; address+topic filter returns exact mined target log and excludes same-topic other-address log. Submit-to-receive upper bound includes queueing/signing/consensus and must be below actual block period, stronger than notification-only delay; no coarse header timestamp subtraction. Not entire public DSL cases/Linux/mainnet/all87/legacy/final Web."}
	defer func() {
		proof["witness"] = p
		proof["submittedRaws"] = observed.submitted()
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/websocket-witness-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	deployments := []map[string]any{}
	for i := 0; i < 2; i++ {
		hash, addr, err := wallet.Deploy(ctx, initCode, nil)
		if err != nil {
			t.Fatal(err)
		}
		receipt := policyReceipt(t, c, hash)
		row := map[string]any{"hash": hash, "address": addr, "receipt": receipt, "tag": receipt["blockNumber"]}
		deployments = append(deployments, row)
		proof["deployments"] = deployments
		number, ok := lifecycleQuantity(receipt["blockNumber"])
		if !ok || !number.IsUint64() {
			t.Fatal("invalid deployment receipt height")
		}
		waitForHead(t, c, number.Uint64()+1, 90*time.Second)
		var block map[string]any
		if err := c.Call(ctx, "eth_getBlockByNumber", &block, receipt["blockNumber"], false); err != nil {
			t.Fatal(err)
		}
		row["block"] = block
		var code string
		if err := c.Call(ctx, "eth_getCode", &code, addr, receipt["blockNumber"]); err != nil {
			t.Fatal(err)
		}
		row["code"] = code
		if err := verifyWSDeployment(row, "0x"+hex.EncodeToString(initCode[12:])); err != nil {
			t.Fatal(err, addr, code)
		}
		if i == 0 {
			p.Contract = addr
		} else {
			p.OtherContract = addr
		}
	}
	proof["deployments"] = deployments
	gd := readPostforkPublicJSON[map[string]any](t, filepath.Join(workspace, "genesis.json"))
	period := gd["config"].(map[string]any)["croissant"].(map[string]any)["wBFT"].(map[string]any)["blockPeriodSeconds"].(float64)
	if period <= 0 {
		t.Fatal("invalid actual block period")
	}
	p.PeriodNanos = int64(period * float64(time.Second))
	start, paused := pauseWSProduction(t, nodes)
	proof["initialPause"] = paused
	if err := c.Call(ctx, "eth_getBlockByNumber", &p.StartBlock, hexUint(start), false); err != nil {
		t.Fatal(err)
	}
	headSub, err := rpc.Subscribe(ctx, nodes[1].WSURL, "newHeads")
	if err != nil {
		t.Fatal(err)
	}
	defer headSub.Close()
	p.HeadID = headSub.ID()
	heads := observeWS(headSub)
	p.Filter = map[string]any{"address": p.Contract, "topics": []any{p.Topic}}
	logSub, err := rpc.Subscribe(ctx, nodes[1].WSURL, "logs", p.Filter)
	if err != nil {
		t.Fatal(err)
	}
	defer logSub.Close()
	p.LogID = logSub.ID()
	logs := observeWS(logSub)
	var priceHex, tipHex string
	if err := c.Call(ctx, "eth_gasPrice", &priceHex); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(ctx, "eth_maxPriorityFeePerGas", &tipHex); err != nil {
		t.Fatal(err)
	}
	price, ok := lifecycleQuantity(priceHex)
	if !ok {
		t.Fatal("invalid price")
	}
	tip, ok := lifecycleQuantity(tipHex)
	if !ok {
		t.Fatal("invalid tip")
	}
	cap := new(big.Int).Add(new(big.Int).Mul(price, big.NewInt(2)), tip)
	nonce, err := c.NonceAt(ctx, sender)
	if err != nil {
		t.Fatal(err)
	}
	negativeHash, err := wallet.SendDynamicFeeTx(ctx, accounts.DynamicTxArgs{ToHex: p.OtherContract, Value: big.NewInt(0), Gas: 100000, GasFeeCap: cap, GasTipCap: tip, Nonce: &nonce})
	if err != nil {
		t.Fatal(err)
	}
	nonce++
	p.SubmittedUnixNano = time.Now().UnixNano()
	p.TargetHash, err = wallet.SendDynamicFeeTx(ctx, accounts.DynamicTxArgs{ToHex: p.Contract, Value: big.NewInt(0), Gas: 100000, GasFeeCap: cap, GasTipCap: tip, Nonce: &nonce})
	if err != nil {
		t.Fatal(err)
	}
	// All endpoints must have the target available before production resumes.
	deadline := time.Now().Add(15 * time.Second)
	for {
		ready := true
		for _, n := range nodes {
			var tx map[string]any
			if err := rpc.Dial(n.RPCURL).Call(ctx, "eth_getTransactionByHash", &tx, p.TargetHash); err != nil {
				t.Fatal(err)
			}
			ready = ready && tx["hash"] == p.TargetHash && tx["blockNumber"] == nil
		}
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("target emission did not propagate before resume")
		}
		time.Sleep(50 * time.Millisecond)
	}
	resumed := []lifecycleControl{}
	for _, n := range nodes {
		resumed = append(resumed, lifecycleMinerControl(t, n, "miner_start", true))
	}
	proof["resumed"] = resumed
	p.Receipt = policyReceipt(t, c, p.TargetHash)
	p.OtherReceipt = policyReceipt(t, c, negativeHash)
	for len(heads.snapshot(t)) < 5 {
		if time.Now().After(deadline.Add(30 * time.Second)) {
			t.Fatal("fewer than five generated WS heads")
		}
		time.Sleep(100 * time.Millisecond)
	}
	end, paused := pauseWSProduction(t, nodes)
	proof["finalPause"] = paused
	if err := c.Call(ctx, "eth_getBlockByNumber", &p.EndBlock, hexUint(end), false); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for uint64(len(heads.snapshot(t))) < end-start {
		if time.Now().After(deadline) {
			t.Fatal("WS head count differs from paused canonical window")
		}
		time.Sleep(50 * time.Millisecond)
	}
	p.Headers = heads.finish(t)
	p.Logs = logs.finish(t)
	for n := start + 1; n <= end; n++ {
		var block map[string]any
		if err := c.Call(ctx, "eth_getBlockByNumber", &block, hexUint(n), false); err != nil {
			t.Fatal(err)
		}
		p.CanonicalBlocks = append(p.CanonicalBlocks, block)
	}
	proof["submittedRaws"] = observed.submitted()
	if err := verifyWebsocketWitness(p); err != nil {
		t.Fatal(err)
	}
}
