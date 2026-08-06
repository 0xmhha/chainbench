//go:build e2e

// This E2E is gated behind the `e2e` build tag and skips unless the chain
// binaries are provided, so `go test ./...` never runs it. It drives the real
// `chainbench upgrade run` framework path against the built go-wemix and go-wbft
// binaries and asserts the live croissant handoff completes AND the wbft
// successor keeps state and processes new transactions/contracts post-fork
// (scenario 2: state preserved + contracts work across the hardfork).
//
// No external etcd is needed — gwemix embeds an etcd server (admin.etcdInit ->
// go.etcd.io/etcd/server/v3/embed.StartEtcd). Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestUpgradeRunE2E -timeout 8m ./cmd/chainbench
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestUpgradeRunE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	dataDir := t.TempDir()
	// The command leaves the node processes running; stop anything bound to this
	// run's datadir when the test ends so ports are freed for the next run.
	t.Cleanup(func() { _ = exec.Command("pkill", "-9", "-f", dataDir).Run() })

	cmd := newUpgradeRunCmd()
	cmd.SetArgs([]string{
		"--profile", "../../profiles/wemix-upgrade.yaml",
		"--preset", "../../keys/preset",
		"--from-binary", fromBin,
		"--to-binary", toBin,
		"--template", template,
		"--data-dir", dataDir,
		"--wait", "150",
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	t.Logf("upgrade run output:\n%s", out.String())
	if err != nil {
		t.Fatalf("upgrade run failed: %v", err)
	}
	if !strings.Contains(out.String(), "handoff confirmed") {
		t.Fatalf("handoff not confirmed in output:\n%s", out.String())
	}

	// A go-wbft successor validator's RPC (plan nodes 2-5; node 1 is the go-wemix
	// producer, which cannot import post-croissant blocks).
	successor := successorRPC(t, out.String())
	key := presetNode1Key(t)
	ctx := context.Background()

	// State preserved: the funded account's balance survived the engine handoff.
	c := rpc.Dial(successor)
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	w, err := ap.OpenWallet(ctx, key, successor)
	if err != nil {
		t.Fatalf("open wallet on successor: %v", err)
	}
	if bal, err := c.BalanceAt(ctx, w.Address()); err != nil || bal.Sign() == 0 {
		t.Fatalf("funded account state not preserved on successor (bal=%v err=%v)", bal, err)
	}

	// Post-fork tx processes on the wbft successor.
	oneEth := new(big.Int).SetUint64(1_000_000_000_000_000_000)
	txHash, err := w.SendCoin(ctx, "0x000000000000000000000000000000000000dEaD", oneEth)
	if err != nil {
		t.Fatalf("post-fork tx send: %v", err)
	}
	waitReceiptOK(t, c, txHash)

	// Post-fork contract deploy/call (returns 42).
	code, _ := hex.DecodeString("600a600c600039600a6000f3602a60005260206000f3")
	_, addr, err := w.Deploy(ctx, code, nil)
	if err != nil {
		t.Fatalf("post-fork contract deploy: %v", err)
	}
	deadline := time.Now().Add(90 * time.Second)
	for {
		if res, err := c.EthCall(ctx, addr, "0x"); err == nil && strings.HasSuffix(res, "2a") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("post-fork contract %s did not return 42", addr)
		}
		time.Sleep(3 * time.Second)
	}
}

// successorRPC parses a go-wbft successor validator's RPC URL from the upgrade
// run output (lines like "  node2  http://127.0.0.1:PORT  pid=NNN"). Plan node 1
// is the go-wemix producer, so it picks node 2.
func successorRPC(t *testing.T, out string) string {
	t.Helper()
	re := regexp.MustCompile(`node2\s+(http://\S+)\s+pid=`)
	m := re.FindStringSubmatch(out)
	if len(m) != 2 {
		t.Fatalf("could not find successor (node2) RPC in output:\n%s", out)
	}
	return m[1]
}

// presetNode1Key loads node 1's private key from keys/preset — a committed TEST
// fixture (public, local-only) whose address is genesis-funded.
func presetNode1Key(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "keys", "preset", "metadata.json"))
	if err != nil {
		t.Fatalf("read preset metadata: %v", err)
	}
	var m struct {
		Nodes []struct {
			Index   int    `json:"index"`
			NodeKey string `json:"nodekey"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse preset metadata: %v", err)
	}
	for _, n := range m.Nodes {
		if n.Index == 1 {
			key, err := hex.DecodeString(strings.TrimPrefix(n.NodeKey, "0x"))
			if err != nil {
				t.Fatalf("decode nodekey: %v", err)
			}
			return key
		}
	}
	t.Fatal("no node 1 in preset metadata")
	return nil
}

// waitReceiptOK polls for a mined receipt and asserts status == 0x1.
func waitReceiptOK(t *testing.T, c *rpc.Client, hash string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		raw, err := c.TxReceipt(context.Background(), hash)
		if err == nil && len(raw) > 0 && string(raw) != "null" {
			var r struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(raw, &r) == nil && r.Status != "" {
				if r.Status != "0x1" {
					t.Fatalf("post-fork tx %s status=%s (want 0x1)", hash, r.Status)
				}
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("post-fork tx %s never mined", hash)
		}
		time.Sleep(2 * time.Second)
	}
}
