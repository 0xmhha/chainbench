//go:build e2e

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixCrossForkColdSyncE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	t.Setenv("GWEMIX_BIN", from)
	t.Setenv("GWBFT_BIN", to)
	senderKey, sender, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	payerKey, payer, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	_, recipient, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	p := crossSyncProof{Sender: sender, Payer: payer, Recipient: recipient, Funding: "1000000000000000000000000000"}
	spec := writeHandoffCase(t, map[string]any{"alloc": map[string]any{sender: map[string]any{"balance": p.Funding}, payer: map[string]any{"balance": p.Funding}}})
	doc := readPostforkPublicJSON[map[string]any](t, spec)
	preset := doc["chainPreset"].(map[string]any)
	preset["upgrade"].(map[string]any)["at"] = 100
	preset["genesis"].(map[string]any)["overlay"].(map[string]any)["config"].(map[string]any)["croissantBlock"] = 100
	doc["steps"] = []map[string]any{{"do": "waitBlock", "target": 102, "timeout": "300s"}, {"expect": "blockNumber", "compare": "Greater", "is": "100"}}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(spec, b, 0600); err != nil {
		t.Fatal(err)
	}
	root := createHandoffRuntime(t, spec)
	workspace := filepath.Join(root, "w")
	defer func() {
		b, e := json.MarshalIndent(p, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/crossfork-sync-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
	defer cancel()
	cmd := newRootCmd()
	cmd.SetArgs([]string{"run", filepath.Join(root, "case.json"), "--workspace-dir", workspace, "--artifact-root", filepath.Join(root, "reports"), "--keys", presetKeysDir(t), "--read-only-keys", "--no-skips", "--keep-up", "--node-monitor-timeout", "5m"})
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	done := make(chan struct{})
	var runErr error
	go func() { defer close(done); runErr = cmd.ExecuteContext(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Error("CLI did not stop; preserve owned runtime")
			return
		}
		preserveHandoffRun(t, root, output.String(), runErr)
	})
	// The run remains in flight while the original WPOA producer receives the
	// first five signed transactions. Reading its output before done would race.
	var preURL string
	deadline := time.Now().Add(2 * time.Minute)
	for preURL == "" {
		select {
		case <-done:
			t.Fatal("CLI ended before pre-fork dataset", runErr, output.String())
		default:
		}
		var rec handoffNodes
		if b, e := os.ReadFile(filepath.Join(workspace, "chain-record.json")); e == nil && json.Unmarshal(b, &rec) == nil {
			for _, n := range rec.Nodes {
				if n.Binary == "next" || n.HTTP == 0 {
					continue
				}
				u := fmt.Sprintf("http://%s:%d", n.Host, n.HTTP)
				if h, e := rpc.Dial(u).BlockNumber(ctx); e == nil && h >= 18 && h < 100 {
					preURL = u
					break
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("pre-fork producer not ready")
		}
		if preURL == "" {
			time.Sleep(100 * time.Millisecond)
		}
	}
	crossSyncPublish(t, ctx, &p, preURL, senderKey, payerKey, "pre")
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("handoff timeout", ctx.Err())
	}
	preserveHandoffRun(t, root, output.String(), runErr)
	if runErr != nil || !strings.Contains(output.String(), "fail=0") {
		t.Fatal("handoff failed", runErr, output.String())
	}
	source := rpc.Dial(successorRPC(t, workspace))
	crossSyncPublish(t, ctx, &p, successorRPC(t, workspace), senderKey, payerKey, "post")
	waitForHead(t, source, 210, 5*time.Minute)
	var info struct{ Enode string }
	if err := source.Call(ctx, "admin_nodeInfo", &info); err != nil {
		t.Fatal(err)
	}
	peer, err := url.Parse(info.Enode)
	if err != nil || peer.User == nil || peer.Port() == "" {
		t.Fatal("missing actual source peer", info, err)
	}
	peer.Host = net.JoinHostPort("127.0.0.1", peer.Port())
	for i, mode := range []string{"full", "snap"} {
		r := crossSyncReader{Mode: mode, Bootnode: peer.String(), Fork: 100, Progress: []uint64{0}}
		p.Readers = append(p.Readers, r)
		row := &p.Readers[len(p.Readers)-1]
		row.SourceStart, err = source.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		dataDir := filepath.Join(workspace, fmt.Sprintf("node%d", i+6))
		if err := process.InitDatadir(ctx, to, dataDir, filepath.Join(workspace, "genesis.json")); err != nil {
			t.Fatal(err)
		}
		logPath := filepath.Join(root, "reports", "cold-"+mode+".log")
		readerURL, stop := launchCrossSyncReader(t, to, dataDir, logPath, mode, peer.String())
		reader := rpc.Dial(readerURL)
		crossSyncWaitRPC(t, ctx, reader)
		row.StartHeight, err = reader.BlockNumber(ctx)
		if err != nil || row.StartHeight != 0 {
			t.Fatal("reader is not cold", row.StartHeight, err)
		}
		var genesis, sourceGenesis map[string]any
		crossSyncCall(t, ctx, reader, "eth_getBlockByNumber", &genesis, "0x0", false)
		crossSyncCall(t, ctx, source, "eth_getBlockByNumber", &sourceGenesis, "0x0", false)
		row.GenesisHash, row.SourceGenesisHash = fmt.Sprint(genesis["hash"]), fmt.Sprint(sourceGenesis["hash"])
		var added bool
		crossSyncCall(t, ctx, reader, "admin_addPeer", &added, peer.String())
		if !added {
			t.Fatal("source peer not added")
		}
		deadline := time.Now().Add(4 * time.Minute)
		for {
			row.Head, err = reader.BlockNumber(ctx)
			if err != nil {
				t.Fatal(err)
			}
			row.SourceHead, err = source.BlockNumber(ctx)
			if err != nil {
				t.Fatal(err)
			}
			row.Progress = append(row.Progress, row.Head)
			crossSyncCall(t, ctx, reader, "net_peerCount", &row.PeerCount)
			var syncing json.RawMessage
			crossSyncCall(t, ctx, reader, "eth_syncing", &syncing)
			row.Syncing = string(syncing)
			if row.Head >= 200 && row.SourceHead <= row.Head+2 && row.Syncing == "false" {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("cold sync did not complete", mode, row.Head, row.SourceHead, row.Syncing)
			}
			time.Sleep(time.Second)
		}
		row.Reference = min(row.Head, row.SourceHead) - 1
		if mode == "snap" {
			log, e := os.ReadFile(logPath)
			if e != nil {
				t.Fatal(e)
			}
			matches := regexp.MustCompile(`Committing snap sync pivot as new head\s+number=(\d+)`).FindAllSubmatch(log, -1)
			if len(matches) == 0 {
				t.Fatal("no actual snap pivot commit in reader log")
			}
			row.Pivot, err = strconv.ParseUint(string(matches[len(matches)-1][1]), 10, 64)
			if err != nil {
				t.Fatal(err)
			}
		}
		crossSyncCall(t, ctx, source, "eth_getBlockByNumber", &row.SourceFork, "0x64", false)
		crossSyncCall(t, ctx, reader, "eth_getBlockByNumber", &row.AtFork, "0x64", false)
		tag := hexUint(row.Reference)
		crossSyncCall(t, ctx, source, "eth_getBlockByNumber", &row.SourceRef, tag, false)
		crossSyncCall(t, ctx, reader, "eth_getBlockByNumber", &row.AtRef, tag, false)
		row.SourceState = crossSyncReadState(t, ctx, source, p, tag)
		row.State = crossSyncReadState(t, ctx, reader, p, tag)
		for _, tx := range p.Dataset {
			v := crossSyncReplay{Hash: tx.Hash}
			crossSyncCall(t, ctx, reader, "eth_getTransactionByHash", &v.Transaction, tx.Hash)
			crossSyncCall(t, ctx, reader, "eth_getTransactionReceipt", &v.Receipt, tx.Hash)
			row.Transactions = append(row.Transactions, v)
		}
		stop() // Full must be stopped before snap starts, including its database.
	}
	if err := verifyCrossSync(p); err != nil {
		t.Fatal(err)
	}
}

func crossSyncCall(t *testing.T, ctx context.Context, c *rpc.Client, method string, result any, args ...any) {
	t.Helper()
	if err := c.Call(ctx, method, result, args...); err != nil {
		t.Fatal(method, err)
	}
}

func crossSyncPublish(t *testing.T, ctx context.Context, p *crossSyncProof, endpoint string, senderKey, payerKey []byte, phase string) {
	t.Helper()
	c := rpc.Dial(endpoint)
	observed := &observedRawRPC{client: c}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := provider.OpenWallet(ctx, senderKey, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	for k, kind := range []string{"legacy", "dynamic", "feeDelegation", "deploy", "write"} {
		var priceHex, tipHex string
		crossSyncCall(t, ctx, c, "eth_gasPrice", &priceHex)
		crossSyncCall(t, ctx, c, "eth_maxPriorityFeePerGas", &tipHex)
		price, ok := lifecycleQuantity(priceHex)
		if !ok || price.Sign() <= 0 {
			t.Fatal("invalid suggested price")
		}
		tip, ok := lifecycleQuantity(tipHex)
		if !ok || tip.Sign() <= 0 {
			t.Fatal("invalid suggested tip")
		}
		nonce := uint64(len(p.Dataset))
		a := accounts.DynamicTxArgs{ToHex: p.Recipient, Value: big.NewInt(0), Gas: 21000, GasFeeCap: new(big.Int).Add(new(big.Int).Mul(price, big.NewInt(2)), tip), GasTipCap: tip, Nonce: &nonce}
		if k < 3 {
			n := int64(k + 1)
			if phase == "post" {
				n += 3
			}
			a.Value.Mul(big.NewInt(n), big.NewInt(1000000000000000000))
		} else if kind == "deploy" {
			a.ToHex, a.Gas = "", 500000
			a.Data, err = hex.DecodeString(crossSyncInitCode[2:])
		} else {
			a.ToHex, a.Gas = p.ContractA, 100000
			n := 42
			if phase == "post" {
				a.ToHex, n = p.ContractB, 84
			}
			a.Data, err = hex.DecodeString(fmt.Sprintf("%064x", n))
		}
		if err != nil {
			t.Fatal(err)
		}
		var hash string
		switch kind {
		case "legacy":
			hash, err = wallet.SendLegacyTx(ctx, accounts.LegacyTxArgs{ToHex: a.ToHex, Value: a.Value, Gas: a.Gas, GasPrice: price, Nonce: a.Nonce})
		case "feeDelegation":
			hash, err = wallet.SendFeeDelegatedTx(ctx, payerKey, a)
		default:
			hash, err = wallet.SendDynamicFeeTx(ctx, a)
		}
		if err != nil {
			t.Fatal("publish", phase, kind, err)
		}
		raw := observed.submitted()
		if len(raw) != k+1 {
			t.Fatal("missing exact raw submission")
		}
		v := crossSyncTx{Phase: phase, Kind: kind, Hash: hash, Raw: raw[k]}
		p.Dataset = append(p.Dataset, v)
		row := &p.Dataset[len(p.Dataset)-1]
		row.Receipt = policyReceipt(t, c, hash)
		crossSyncCall(t, ctx, c, "eth_getTransactionByHash", &row.Transaction, hash)
		crossSyncCall(t, ctx, c, "eth_getBlockByNumber", &row.Block, row.Receipt["blockNumber"], false)
		if kind == "deploy" {
			contract := fmt.Sprint(row.Receipt["contractAddress"])
			if phase == "pre" {
				p.ContractA = contract
			} else {
				p.ContractB = contract
			}
			h, ok := lifecycleQuantity(row.Receipt["blockNumber"])
			if !ok {
				t.Fatal("missing deployment height")
			}
			waitForHead(t, c, h.Uint64()+1, time.Minute)
			var code string
			crossSyncCall(t, ctx, c, "eth_getCode", &code, contract, row.Receipt["blockNumber"])
			if code != crossSyncRuntimeCode {
				t.Fatal("deployment code absent at receipt height", code)
			}
		}
	}
}

func crossSyncReadState(t *testing.T, ctx context.Context, c *rpc.Client, p crossSyncProof, tag string) crossSyncState {
	t.Helper()
	s := crossSyncState{Tag: tag, Balances: map[string]string{}, Nonces: map[string]string{}, Codes: map[string]string{}, Storage: map[string]string{}}
	for _, a := range []string{p.Sender, p.Payer, p.Recipient} {
		var v string
		crossSyncCall(t, ctx, c, "eth_getBalance", &v, a, tag)
		s.Balances[a] = v
	}
	for _, a := range []string{p.Sender, p.Payer} {
		var v string
		crossSyncCall(t, ctx, c, "eth_getTransactionCount", &v, a, tag)
		s.Nonces[a] = v
	}
	for _, a := range []string{p.ContractA, p.ContractB} {
		var code, slot string
		crossSyncCall(t, ctx, c, "eth_getCode", &code, a, tag)
		crossSyncCall(t, ctx, c, "eth_getStorageAt", &slot, a, "0x0", tag)
		s.Codes[a], s.Storage[a] = code, slot
	}
	return s
}

func crossSyncWaitRPC(t *testing.T, ctx context.Context, c *rpc.Client) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := c.BlockNumber(ctx); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("cold reader RPC not ready")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func launchCrossSyncReader(t *testing.T, binary, dataDir, logPath, mode, peer string) (string, func()) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err = l.Close(); err != nil {
		t.Fatal(err)
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--datadir", dataDir, "--port", "0", "--nodiscover", "--maxpeers", "4", "--syncmode", mode, "--cache", "64", "--verbosity", "4", "--http", "--http.addr", "127.0.0.1", "--http.port", fmt.Sprint(port), "--http.api", "eth,net,web3,admin,debug", "--networkid", "8285", "--bootnodes", peer}
	child := exec.Command(binary, args...)
	child.Stdout, child.Stderr = log, log
	if err = child.Start(); err != nil {
		_ = log.Close()
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = child.Wait(); close(done) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			select {
			case <-done:
			default:
				_ = child.Process.Signal(syscall.SIGTERM)
				select {
				case <-done:
				case <-time.After(10 * time.Second):
					_ = child.Process.Kill()
					<-done
				}
			}
			if err := log.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	t.Cleanup(stop)
	return fmt.Sprintf("http://127.0.0.1:%d", port), stop
}
