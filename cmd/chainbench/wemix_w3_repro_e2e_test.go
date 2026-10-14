//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// This is a diagnostic witness, not a W3 fix or an acceptance award. It uses
// the real importer/seal verifier with a fixed partial-governance state, then
// observes network sync from that same state. All databases are disposable;
// exported blocks, RPC observations, configuration and logs are retained.
func TestWemixW3PartialGovernanceReproductionE2E(t *testing.T) {
	if os.Getenv("CHAINBENCH_W3_PROOF") != "1" {
		t.Skip("set CHAINBENCH_W3_PROOF=1 for the explicit W3 diagnostic")
	}
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Fatal("W3 diagnostic requires both explicit binaries")
	}
	ctx := context.Background()
	workspace, producerURL, _ := runHandoffKeepDatadir(t, from, to)
	root := filepath.Dir(workspace)
	c := rpc.Dial(producerURL)
	head, err := c.BlockNumber(ctx)
	if err != nil || head < 3 {
		t.Fatalf("producer chain unavailable: %d %v", head, err)
	}
	registry, partial, creation := findW3Registry(t, c, head)
	data := accounts.EncodeCallArgs("getContractAddress(bytes32)", accounts.Word([]byte("GovernanceContract")))
	var domain string
	err = c.Call(ctx, "eth_call", &domain, map[string]string{"to": registry, "data": data}, hexUint(partial))
	var refusal *rpc.ServerError
	if !errors.As(err, &refusal) || !strings.Contains(refusal.Message, "address should be non-zero") {
		t.Fatalf("registry creation state is not the W3 incomplete window: %v", err)
	}
	var nodeInfo struct {
		Enode string `json:"enode"`
	}
	if err := c.Call(ctx, "admin_nodeInfo", &nodeInfo); err != nil {
		t.Fatal(err)
	}
	peer, err := url.Parse(nodeInfo.Enode)
	if err != nil || peer.Port() == "" {
		t.Fatalf("producer enode: %q %v", nodeInfo.Enode, err)
	}
	peer.Host = net.JoinHostPort("127.0.0.1", peer.Port())
	prefix, remaining := filepath.Join(root, "reports/w3-prefix.rlp"), filepath.Join(root, "reports/w3-remaining.rlp")
	for _, export := range []struct {
		path        string
		first, last uint64
	}{{prefix, 1, partial}, {remaining, partial + 1, head}} {
		var ok bool
		if err := c.Call(ctx, "admin_exportChain", &ok, export.path, export.first, export.last); err != nil || !ok {
			t.Fatalf("export blocks: %v %v", ok, err)
		}
		if err := os.Chmod(export.path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	readerDir := filepath.Join(workspace, "node99")
	if err := os.MkdirAll(readerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := process.InitDatadir(ctx, from, readerDir, filepath.Join(workspace, "genesis.json")); err != nil {
		t.Fatal(err)
	}
	readerLog := filepath.Join(root, "reports/w3-reader.log")
	readerURL := launchHandoffReader(t, from, readerDir, readerLog)
	reader := rpc.Dial(readerURL)
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := reader.BlockNumber(ctx); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("W3 reader RPC never became available")
		}
		time.Sleep(time.Second)
	}
	var imported bool
	if err := reader.Call(ctx, "admin_importChain", &imported, prefix); err != nil || !imported {
		t.Fatalf("prefix import did not establish a controlled state: %v %v", imported, err)
	}
	got, err := reader.BlockNumber(ctx)
	if err != nil || got != partial {
		t.Fatalf("reader prefix head=%d, want %d: %v", got, partial, err)
	}
	var producerBlock, readerBlock struct {
		Hash string `json:"hash"`
	}
	if err := c.Call(ctx, "eth_getBlockByNumber", &producerBlock, hexUint(partial), false); err != nil {
		t.Fatal(err)
	}
	if err := reader.Call(ctx, "eth_getBlockByNumber", &readerBlock, hexUint(partial), false); err != nil {
		t.Fatal(err)
	}
	if producerBlock.Hash == "" || readerBlock.Hash != producerBlock.Hash {
		t.Fatal("reader prefix is not the producer's exact chain")
	}
	err = reader.Call(ctx, "eth_call", &domain, map[string]string{"to": registry, "data": data}, "latest")
	if !errors.As(err, &refusal) || !strings.Contains(refusal.Message, "address should be non-zero") {
		t.Fatalf("reader does not hold the incomplete registry state: %v", err)
	}
	failures := []string{}
	for attempt := 0; attempt < 2; attempt++ {
		imported = false
		err = reader.Call(ctx, "admin_importChain", &imported, remaining)
		if !errors.As(err, &refusal) || !strings.Contains(refusal.Message, "unauthorized block") {
			t.Fatalf("W3 verifier rejection not reproduced (not a fix verdict): imported=%v err=%v", imported, err)
		}
		failures = append(failures, refusal.Error())
		got, err = reader.BlockNumber(ctx)
		if err != nil || got != partial {
			t.Fatalf("rejected batch changed reader head: %d %v", got, err)
		}
	}
	var added bool
	if err := reader.Call(ctx, "admin_addPeer", &added, peer.String()); err != nil || !added {
		t.Fatalf("cannot observe actual full sync: %v %v", added, err)
	}
	observations := []map[string]any{}
	connected := false
	for i := 0; i < 10; i++ {
		got, err := reader.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var peers string
		if err := reader.Call(ctx, "net_peerCount", &peers); err != nil {
			t.Fatal(err)
		}
		observations = append(observations, map[string]any{"head": got, "peers": peers})
		if err := validateW3Observation(partial, got, peers); err != nil {
			t.Fatal(err)
		}
		connected = connected || peers != "0x0"
		time.Sleep(2 * time.Second)
	}
	if !connected {
		t.Fatal("no connected peer; the full-sync witness is not established")
	}
	readerOutput, err := os.ReadFile(readerLog)
	if err != nil {
		t.Fatal(err)
	}
	if !w3NetworkRejection(string(readerOutput)) {
		t.Fatal("no unauthorized-block downloader failure; the full-sync witness is not established")
	}
	result := map[string]any{"outcome": "reproduced", "issueResolved": false, "scope": "fixed partial-governance state through real importer/seal verifier plus full-sync observation; not the historical 15-node race", "producerHead": head, "partialHead": partial, "registry": registry, "creationReceipt": creation, "prefixHash": readerBlock.Hash, "rejections": failures, "networkObservations": observations, "peer": peer.String(), "sourceBinary": from, "workspace": workspace}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "reports/w3-proof.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("W3 remains open; real rejection and persistent reader head reproduced: %s", path)
}

func findW3Registry(t *testing.T, c *rpc.Client, head uint64) (string, uint64, json.RawMessage) {
	t.Helper()
	for n := uint64(1); n < head; n++ {
		var block struct {
			Transactions []struct {
				Hash string  `json:"hash"`
				To   *string `json:"to"`
			} `json:"transactions"`
		}
		if err := c.Call(context.Background(), "eth_getBlockByNumber", &block, hexUint(n), true); err != nil {
			t.Fatal(err)
		}
		for _, tx := range block.Transactions {
			if tx.To != nil {
				continue
			}
			receipt, err := c.TxReceipt(context.Background(), tx.Hash)
			if err != nil {
				t.Fatal(err)
			}
			var r struct {
				Address string `json:"contractAddress"`
				Status  string `json:"status"`
			}
			if err := json.Unmarshal(receipt, &r); err != nil {
				t.Fatal(err)
			}
			if r.Status != "0x1" || r.Address == "" {
				continue
			}
			magic, err := c.EthCall(context.Background(), r.Address, accounts.EncodeCallArgs("magic()"))
			if err == nil && strings.Contains(strings.ToLower(magic), hex.EncodeToString([]byte("Wemix Registry"))) {
				return r.Address, n, receipt
			}
		}
	}
	t.Fatal("no registry creation transaction in the bounded producer history")
	return "", 0, nil
}

func launchHandoffReader(t *testing.T, binary, dataDir, logPath string) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	// A fresh node key avoids connecting the producer to its own identity.
	args := []string{"--datadir", dataDir, "--port", "0", "--nodiscover", "--maxpeers", "4", "--syncmode", "full", "--cache", "64", "--http", "--http.addr", "127.0.0.1", "--http.port", fmt.Sprint(port), "--http.api", "eth,net,web3,admin,debug", "--networkid", "8285"}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(binary, args...)
	child.Stdout = log
	child.Stderr = log
	if err := child.Start(); err != nil {
		_ = log.Close()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	t.Cleanup(func() {
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
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func validateW3Observation(partial, head uint64, peers string) error {
	if head != partial {
		return fmt.Errorf("W3 reader advanced to %d; persistent-stall witness is not established", head)
	}
	if !strings.HasPrefix(peers, "0x") {
		return fmt.Errorf("invalid peer count %q", peers)
	}
	count, err := strconv.ParseUint(strings.TrimPrefix(peers, "0x"), 16, 64)
	if err != nil || hexUint(count) != peers {
		return fmt.Errorf("invalid peer count %q", peers)
	}
	return nil
}

func TestW3ObservationRejectsInvalidEvidence(t *testing.T) {
	for _, tc := range []struct {
		name      string
		head      uint64
		peers     string
		wantError bool
	}{
		{"connected stalled reader", 4, "0x1", false},
		{"awaiting peer", 4, "0x0", false},
		{"advanced reader", 5, "0x1", true},
		{"empty peer count", 4, "", true},
		{"malformed peer count", 4, "0xwat", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateW3Observation(4, tc.head, tc.peers); (err != nil) != tc.wantError {
				t.Fatalf("observation error=%v; want error=%v", err, tc.wantError)
			}
		})
	}
}

func w3NetworkRejection(output string) bool {
	download, dropped := false, false
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "unauthorized block") {
			continue
		}
		download = download || strings.Contains(line, "Downloaded item processing failed")
		dropped = dropped || strings.Contains(line, "Synchronisation failed, dropping peer")
	}
	return download && dropped
}

func TestW3NetworkWitnessRequiresDownloaderRejection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		output string
		want   bool
	}{
		{"actual download rejection", "Downloaded item processing failed err=\"unauthorized block\"\nSynchronisation failed, dropping peer err=\"retrieved hash chain is invalid: unauthorized block\"", true},
		{"manual import only", "Error: unauthorized block", false},
		{"unrelated network error", "Downloaded item processing failed err=\"timeout\"\nSynchronisation failed, dropping peer err=\"timeout\"", false},
		{"separate unrelated refusal", "Error: unauthorized block\nDownloaded item processing failed err=\"timeout\"\nSynchronisation failed, dropping peer err=\"timeout\"", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := w3NetworkRejection(tc.output); got != tc.want {
				t.Fatalf("network witness=%v, want %v", got, tc.want)
			}
		})
	}
}
