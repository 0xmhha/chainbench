//go:build e2e

// Package e2e holds the live-verification tier: gated end-to-end tests that boot
// a network from a REAL chain binary and drive it. They are gated behind the
// `e2e` build tag, so `go test ./...` (CI) never compiles or runs them —
// `go test -tags e2e ./tests/e2e/` does, and only when the relevant binary is
// present (else the test t.Skip's).
//
// This replaces the former bash+python `tests/repro/*.sh` scripts: network
// orchestration goes through the chainbench CLI (the same surface a user drives),
// and every assertion is pure Go (pkg/core/rpc + pkg/accounts) — no python, no
// web3.
package e2e

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

// returns42Init is the creation bytecode of a contract whose runtime
// (602a60005260206000f3) returns the 32-byte value 42 for any call. Shared
// fixture (mirrors tests/wbft/accounts/contract_roundtrip.go).
const returns42Init = "600a600c600039600a6000f3602a60005260206000f3"

// deadAddr is a throwaway value-transfer sink.
const deadAddr = "0x000000000000000000000000000000000000dEaD"

// repoRoot returns the repository root (two levels up from tests/e2e).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// requireBinary resolves a chain node binary from env (e.g. GSTABLE_BIN) or
// PATH, skipping the test when it is absent — the standard gate for this tier.
func requireBinary(t *testing.T, env, name string) string {
	t.Helper()
	if p := os.Getenv(env); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		t.Skipf("%s=%s not found", env, p)
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	t.Skipf("no %s binary (set %s=/path/to/%s)", name, env, name)
	return ""
}

// buildCLI returns a built chainbench binary path (CHAINBENCH env or a fresh
// build into a temp dir).
func buildCLI(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("CHAINBENCH"); p != "" {
		return p
	}
	out := filepath.Join(t.TempDir(), "chainbench")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/chainbench")
	cmd.Dir = repoRoot(t)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build chainbench: %v\n%s", err, b)
	}
	return out
}

// network is a running local chain a test drives and tears down.
type network struct {
	t      *testing.T
	cli    string
	dir    string
	chain  string
	binary string
	rpcURL string // node 1
}

// boot launches `validators` validators + `endpoints` endpoints of chain on
// binary via `chainbench setup --launch`, and registers cleanup. extraSet are
// additional `--set key=value` overrides (e.g. "genesis.overrides.bohoBlock=40").
func boot(t *testing.T, cli, chain, binary string, validators, endpoints int, extraSet ...string) *network {
	t.Helper()
	// Use a SHORT datadir under /tmp, not t.TempDir(): a node's IPC endpoint is a
	// unix-domain socket at <datadir>/nodeN/<binary>.ipc, and the ~104-byte socket
	// path limit is easily exceeded by the long t.TempDir() paths (which embed the
	// full test name) — the node then fails to bind its IPC and never produces.
	dir, err := os.MkdirTemp("/tmp", "cbe2e")
	if err != nil {
		t.Fatalf("mkdir temp datadir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	args := []string{"setup", "--launch",
		"--chain", chain, "--binary", binary,
		"--data-dir", dir, "--keys-dir", filepath.Join(repoRoot(t), "keys", "preset"),
		"--validators", itoa(validators), "--endpoints", itoa(endpoints),
	}
	for _, s := range extraSet {
		args = append(args, "--set", s)
	}
	cmd := exec.Command(cli, args...)
	cmd.Dir = repoRoot(t)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("setup --launch: %v\n%s", err, b)
	}
	n := &network{t: t, cli: cli, dir: dir, chain: chain, binary: binary}
	n.rpcURL = n.rpcURLFor(1)
	t.Cleanup(n.stop)
	return n
}

// stop tears the network down (best-effort).
func (n *network) stop() {
	cmd := exec.Command(n.cli, "stop", "--data-dir", n.dir)
	cmd.Dir = repoRoot(n.t)
	_ = cmd.Run()
}

// run execs a chainbench subcommand against this network, failing on error.
func (n *network) run(args ...string) string {
	n.t.Helper()
	cmd := exec.Command(n.cli, args...)
	cmd.Dir = repoRoot(n.t)
	b, err := cmd.CombinedOutput()
	if err != nil {
		n.t.Fatalf("chainbench %v: %v\n%s", args, err, b)
	}
	return string(b)
}

// nodeStop stops node `index`, preserving its datadir for a later nodeStart.
func (n *network) nodeStop(index int) {
	n.run("node", "stop", "--data-dir", n.dir, "--index", itoa(index))
}

// nodeStart relaunches a previously-stopped node `index`.
func (n *network) nodeStart(index int) {
	n.run("node", "start", "--data-dir", n.dir, "--index", itoa(index))
}

// hardfork swaps every node to toBinary (same or different chain) in place at
// `block`, via `chainbench hardfork --dry-run=false`.
func (n *network) hardfork(toChain, toBinary string, block int64) string {
	return n.run("hardfork", "--data-dir", n.dir, "--to-chain", toChain,
		"--to-binary", toBinary, "--block", strconv.FormatInt(block, 10), "--dry-run=false")
}

// rpcURLFor reads a node's RPC URL from the persisted nodeset.json.
func (n *network) rpcURLFor(index int) string {
	n.t.Helper()
	b, err := os.ReadFile(filepath.Join(n.dir, "nodeset.json"))
	if err != nil {
		n.t.Fatalf("read nodeset: %v", err)
	}
	var ns struct {
		Nodes []struct {
			Index  int    `json:"index"`
			RPCURL string `json:"rpc_url"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(b, &ns); err != nil {
		n.t.Fatalf("parse nodeset: %v", err)
	}
	for _, node := range ns.Nodes {
		if node.Index == index {
			return node.RPCURL
		}
	}
	n.t.Fatalf("no node %d in nodeset", index)
	return ""
}

// head returns the current block height at url (-1 on error).
func head(t *testing.T, url string) int64 {
	c := rpc.Dial(url)
	bn, err := c.BlockNumber(context.Background())
	if err != nil {
		return -1
	}
	return int64(bn)
}

// waitAdvancing polls until the head at url grows past its first sample, or
// fails after timeout. WBFT consensus can take ~10-15s to warm up (head sits at
// 1, then accelerates), so a single before/after window is too flaky — poll.
func (n *network) waitAdvancing(url string, timeout time.Duration) {
	n.t.Helper()
	start := head(n.t, url)
	deadline := timeAfter(timeout)
	for !deadline() {
		time.Sleep(3 * time.Second)
		if head(n.t, url) > start {
			return
		}
	}
	n.t.Fatalf("chain not producing blocks at %s (head stuck at %d)", url, start)
}

// grewWithin reports whether the head at url grew over the window — used for the
// negative check that production has HALTED (expects false).
func grewWithin(t *testing.T, url string, window time.Duration) bool {
	before := head(t, url)
	time.Sleep(window)
	return head(t, url) > before
}

// waitCross polls until the head at url exceeds target, or fails after timeout.
func (n *network) waitCross(url string, target int64, timeout time.Duration) {
	n.t.Helper()
	deadline := timeAfter(timeout)
	for !deadline() {
		if head(n.t, url) > target {
			return
		}
		time.Sleep(3 * time.Second)
	}
	n.t.Fatalf("head did not cross %d at %s (last=%d)", target, url, head(n.t, url))
}

// wallet opens an accounts SDK wallet for the chain, funded by key, against url.
func (n *network) wallet(key []byte, url string) accounts.Wallet {
	n.t.Helper()
	ap, err := accounts.ForChain(n.chain)
	if err != nil {
		n.t.Fatalf("accounts.ForChain(%s): %v", n.chain, err)
	}
	w, err := ap.OpenWallet(context.Background(), key, url)
	if err != nil {
		n.t.Fatalf("open wallet: %v", err)
	}
	return w
}

// sendValue transfers wei from key to `to` on url and asserts the receipt is a
// success. Returns nothing; fails the test on any error.
func (n *network) sendValue(key []byte, url, to string, wei *big.Int) {
	n.t.Helper()
	hash, err := n.wallet(key, url).SendCoin(context.Background(), to, wei)
	if err != nil {
		n.t.Fatalf("send value: %v", err)
	}
	n.waitReceiptSuccess(url, hash)
}

// deployReturns42 deploys the returns-42 contract from key on url and returns
// its address (mined confirmation is via waitCallReturns42).
func (n *network) deployReturns42(key []byte, url string) string {
	n.t.Helper()
	code, err := hex.DecodeString(returns42Init)
	if err != nil {
		n.t.Fatalf("decode init: %v", err)
	}
	_, addr, err := n.wallet(key, url).Deploy(context.Background(), code, nil)
	if err != nil {
		n.t.Fatalf("deploy contract: %v", err)
	}
	return addr
}

// waitCallReturns42 polls eth_call on addr until it returns 42 (0x..2a), or
// fails after timeout.
func (n *network) waitCallReturns42(url, addr string, timeout time.Duration) {
	n.t.Helper()
	c := rpc.Dial(url)
	deadline := timeAfter(timeout)
	var last string
	for !deadline() {
		res, err := c.EthCall(context.Background(), addr, "0x")
		if err == nil {
			last = res
			if strings.HasSuffix(res, "2a") {
				return
			}
		}
		time.Sleep(3 * time.Second)
	}
	n.t.Fatalf("contract %s did not return 42 (last=%q)", addr, last)
}

// waitReceiptSuccess polls for a mined receipt and asserts status == 0x1.
func (n *network) waitReceiptSuccess(url, hash string) {
	n.t.Helper()
	c := rpc.Dial(url)
	deadline := timeAfter(90 * time.Second)
	for !deadline() {
		raw, err := c.TxReceipt(context.Background(), hash)
		if err == nil && len(raw) > 0 && string(raw) != "null" {
			var r struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(raw, &r) == nil && r.Status != "" {
				if r.Status == "0x1" {
					return
				}
				n.t.Fatalf("tx %s receipt status=%s (want 0x1)", hash, r.Status)
			}
		}
		time.Sleep(2 * time.Second)
	}
	n.t.Fatalf("tx %s never mined", hash)
}

// hexBlock formats a block height as a 0x-hex ref for eth_getBlockByNumber.
func hexBlock(n int64) string { return "0x" + strconv.FormatInt(n, 16) }

// blockField reads a string field (e.g. "hash", "parentHash", "stateRoot") from
// the block at blockRef ("latest" or a 0x-hex height) via eth_getBlockByNumber.
// Returns "" on error or a missing/non-string field.
func blockField(url, blockRef, field string) string {
	var blk map[string]any
	if err := rpc.Dial(url).Call(context.Background(), "eth_getBlockByNumber", &blk, blockRef, false); err != nil {
		return ""
	}
	if v, ok := blk[field].(string); ok {
		return v
	}
	return ""
}

// balance returns the wei balance of addr at url.
func balance(t *testing.T, url, addr string) *big.Int {
	t.Helper()
	b, err := rpc.Dial(url).BalanceAt(context.Background(), addr)
	if err != nil {
		t.Fatalf("balance %s: %v", addr, err)
	}
	return b
}

// presetFundedKey returns node1's private key from keys/preset — a committed
// TEST fixture (public, local-only) whose address is genesis-funded. NEVER a
// real key; used only to fund local ephemeral test networks.
func presetFundedKey(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "keys", "preset", "metadata.json"))
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
	for _, node := range m.Nodes {
		if node.Index == 1 {
			key, err := hex.DecodeString(strings.TrimPrefix(node.NodeKey, "0x"))
			if err != nil {
				t.Fatalf("decode nodekey: %v", err)
			}
			return key
		}
	}
	t.Fatal("no node 1 in preset metadata")
	return nil
}

// --- small helpers (avoid extra imports) ---

func itoa(n int) string {
	return strconv.Itoa(n)
}

// envOr returns the env var value, or def when unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envInt returns the env var as int64, or def when unset/unparseable.
func envInt(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

// synced reports whether eth_syncing at url is `false` (fully synced). While
// syncing, eth_syncing returns a progress object (non-bool), so anything but a
// literal false means "still syncing".
func synced(url string) bool {
	var s any
	if err := rpc.Dial(url).Call(context.Background(), "eth_syncing", &s); err != nil {
		return false
	}
	b, ok := s.(bool)
	return ok && !b
}

// timeAfter returns a func reporting whether the timeout has elapsed.
func timeAfter(d time.Duration) func() bool {
	end := time.Now().Add(d)
	return func() bool { return time.Now().After(end) }
}
