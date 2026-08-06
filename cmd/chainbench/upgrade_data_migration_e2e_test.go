//go:build e2e

// This E2E ports the reachable core of wemix4 NODE-002 (wemix -> wbft in-place
// data migration): a go-wemix node's chaindata, once its instance directory is
// bridged from the go-wemix layout (<datadir>/geth) to the go-wbft layout
// (<datadir>/gwemix), is opened by the go-wbft binary on the SAME datadir and
// its pre-fork blocks are recognized — the block height carries over and old
// (wpoa) block state is readable. This is the data-recognition half of NODE-002;
// "continue producing past croissant" (step 7) needs the migrated node to also be
// a wbft validator, which the handoff producer is deliberately not, so it is out
// of scope here.
//
// It reuses the real handoff to produce a genuine go-wemix datadir (the producer,
// node1, mines the pre-fork chain), then stops every node, bridges node1's
// instance dir, and reopens it with go-wbft as an offline reader.
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestWemixDataMigrationE2E -timeout 10m ./cmd/chainbench
package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixDataMigrationE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	ctx := context.Background()

	// 1. Run the handoff, keeping the datadir. node1 (go-wemix producer) mines the
	// pre-fork chain into <dataRoot>/node1/geth.
	dataRoot, node1URL, mgr := runHandoffKeepDatadir(t, fromBin, toBin, template)
	t.Cleanup(func() { _ = os.RemoveAll(dataRoot) })

	// Record node1's pre-fork head before shutting it down.
	preforkHead, err := rpc.Dial(node1URL).BlockNumber(ctx)
	if err != nil || preforkHead == 0 {
		t.Fatalf("read node1 pre-fork head: head=%d err=%v", preforkHead, err)
	}
	t.Logf("go-wemix producer pre-fork head: %d", preforkHead)

	// 2. Stop every handoff node so node1's chaindata is closed and reopenable.
	if leaks := mgr.StopAll(10 * time.Second); len(leaks) > 0 {
		t.Logf("procman: leaked handoff PIDs before migration: %v", leaks)
	}
	// Give the OS a moment to release the DB lock.
	time.Sleep(2 * time.Second)

	// 3. Bridge node1's instance directory: go-wemix writes <datadir>/geth,
	// go-wbft reads <datadir>/gwemix. A relative symlink gwemix -> geth reuses the
	// exact chaindata files in place (the migration NODE-002 performs).
	node1dd := filepath.Join(dataRoot, "node1")
	gethDir := filepath.Join(node1dd, "geth")
	if _, err := os.Stat(filepath.Join(gethDir, "chaindata")); err != nil {
		t.Fatalf("go-wemix chaindata not found at %s: %v", gethDir, err)
	}
	gwemixDir := filepath.Join(node1dd, "gwemix")
	_ = os.RemoveAll(gwemixDir)
	if err := os.Symlink("geth", gwemixDir); err != nil {
		t.Fatalf("symlink gwemix -> geth: %v", err)
	}

	// 4. Align the DB config with the new binary (non-destructive: keeps blocks).
	genesisPath := filepath.Join(dataRoot, "genesis.json")
	if err := driver.InitDatadir(ctx, toBin, node1dd, genesisPath); err != nil {
		t.Fatalf("go-wbft init on migrated datadir: %v", err)
	}

	// 5. Reopen the migrated datadir with go-wbft as an offline reader (no mining,
	// no discovery) on isolated ports.
	readerURL := launchOfflineReader(t, toBin, node1dd)

	// 6. The go-wbft reader must recognize the pre-fork chain: the height carries
	// over and an old wpoa block's state root is readable.
	cr := rpc.Dial(readerURL)
	deadline := time.Now().Add(60 * time.Second)
	var head uint64
	for {
		h, err := cr.BlockNumber(ctx)
		if err == nil && h >= preforkHead {
			head = h
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("go-wbft reader did not recognize migrated chain (head=%d, want >= %d, err=%v)", h, preforkHead, err)
		}
		time.Sleep(2 * time.Second)
	}
	t.Logf("go-wbft reader head on migrated datadir: %d (>= pre-fork %d)", head, preforkHead)

	sample := preforkHead / 2
	if sample < 1 {
		sample = 1
	}
	var blk struct {
		StateRoot string `json:"stateRoot"`
		Hash      string `json:"hash"`
	}
	if err := cr.Call(ctx, "eth_getBlockByNumber", &blk, hexUint(sample), false); err != nil {
		t.Fatalf("eth_getBlockByNumber(%d) on reader: %v", sample, err)
	}
	if blk.StateRoot == "" || blk.Hash == "" {
		t.Fatalf("wpoa block %d missing state on migrated datadir (hash=%q stateRoot=%q)", sample, blk.Hash, blk.StateRoot)
	}
}

// runHandoffKeepDatadir runs the handoff like runGovHandoff but returns the data
// root, the producer (node1) RPC URL, and the procman.Manager tracking the nodes
// (the caller stops them) instead of auto-tearing-down. It retries the flaky
// producer/etcd bootstrap.
func runHandoffKeepDatadir(t *testing.T, fromBin, toBin, template string) (string, string, *procman.Manager) {
	t.Helper()
	var lastOut string
	for attempt := 1; attempt <= govHandoffAttempts; attempt++ {
		dataDir, err := os.MkdirTemp("/tmp", "cbmigrate")
		if err != nil {
			t.Fatalf("mkdir temp datadir: %v", err)
		}
		mgr := procman.New()

		cmd := newUpgradeRunCmd()
		cmd.SetArgs([]string{
			"--profile", "../../profiles/wemix-upgrade.yaml",
			"--preset", "../../keys/preset",
			"--from-binary", fromBin,
			"--to-binary", toBin,
			"--template", template,
			"--data-dir", dataDir,
			"--wait", govHandoffWait,
		})
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		runErr := cmd.Execute()
		mgr.TrackFromOutput(out.String())
		mgr.TrackNodeSet(dataDir)

		if runErr == nil && strings.Contains(out.String(), "handoff confirmed") {
			if attempt > 1 {
				t.Logf("handoff confirmed on attempt %d/%d", attempt, govHandoffAttempts)
			}
			return dataDir, node1RPC(t, out.String()), mgr
		}

		lastOut = out.String()
		if leaks := mgr.StopAll(10 * time.Second); len(leaks) > 0 {
			t.Logf("procman: attempt %d leaked node PIDs %v", attempt, leaks)
		}
		_ = os.RemoveAll(dataDir)
		t.Logf("handoff attempt %d/%d failed (flaky producer/etcd bootstrap); retrying", attempt, govHandoffAttempts)
	}
	t.Fatalf("handoff not confirmed after %d attempts:\n%s", govHandoffAttempts, lastOut)
	return "", "", nil
}

// node1RPC parses the producer (node1) RPC URL from the upgrade run output.
func node1RPC(t *testing.T, out string) string {
	t.Helper()
	m := regexp.MustCompile(`node1\s+(http://\S+)\s+pid=`).FindStringSubmatch(out)
	if len(m) != 2 {
		t.Fatalf("could not find producer (node1) RPC in output:\n%s", out)
	}
	return m[1]
}

// launchOfflineReader starts the go-wbft binary on a datadir as a non-mining,
// non-discovering reader with an isolated RPC and returns its RPC URL. The
// process is killed on test cleanup.
func launchOfflineReader(t *testing.T, binary, dataDir string) string {
	t.Helper()
	const p2pPort, httpPort = 39557, 49557
	cmd := exec.Command(binary,
		"--datadir", dataDir,
		"--port", strconv.Itoa(p2pPort),
		"--nodiscover", "--maxpeers", "0",
		"--http", "--http.addr", "127.0.0.1",
		"--http.port", strconv.Itoa(httpPort),
		"--http.api", "eth,net,web3",
		"--networkid", "8285",
	)
	logf, _ := os.Create(filepath.Join(dataDir, "reader.log"))
	if logf != nil {
		cmd.Stdout, cmd.Stderr = logf, logf
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("launch go-wbft reader: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
		if logf != nil {
			_ = logf.Close()
		}
	})
	return "http://127.0.0.1:" + strconv.Itoa(httpPort)
}
