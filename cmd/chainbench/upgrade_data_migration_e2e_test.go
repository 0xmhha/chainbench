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
//	go test -tags e2e -run TestWemixDataMigrationE2E -timeout 10m ./cmd/chainbench
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixDataMigrationE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	ctx := context.Background()

	// 1. Compose the handoff and keep the workspace. The go-wemix producer — the
	// node the preset leaves on the from-binary — mines the pre-fork chain into
	// its own <datadir>/geth. Which node that is belongs to the declaration, so
	// the record answers it rather than this test assuming node1.
	dataRoot, producerURL, producerDir, pids := runHandoffKeepDatadir(t, fromBin, toBin)
	t.Cleanup(func() { _ = os.RemoveAll(dataRoot) })

	// Record the producer's pre-fork head before shutting it down.
	preforkHead, err := rpc.Dial(producerURL).BlockNumber(ctx)
	if err != nil || preforkHead == 0 {
		t.Fatalf("read producer pre-fork head: head=%d err=%v", preforkHead, err)
	}
	t.Logf("go-wemix producer pre-fork head: %d", preforkHead)

	// 2. Stop every handoff node so node1's chaindata is closed and reopenable.
	if leaks := stopPIDs(pids, 10*time.Second); len(leaks) > 0 {
		t.Logf("process: leaked handoff PIDs before migration: %v", leaks)
	}
	// Give the OS a moment to release the DB lock.
	time.Sleep(2 * time.Second)

	// 3. Bridge the producer's instance directory: go-wemix writes <datadir>/geth,
	// go-wbft reads <datadir>/gwemix. A relative symlink gwemix -> geth reuses the
	// exact chaindata files in place (the migration NODE-002 performs).
	node1dd := producerDir
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
	if err := process.InitDatadir(ctx, toBin, node1dd, genesisPath); err != nil {
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

// runHandoffKeepDatadir composes the handoff like runGovHandoff but hands the
// workspace back instead of tearing it down: this test reads the producer's
// chaindata after the network stops, which is the thing it is checking.
//
// It returns the workspace, the PRODUCER's RPC URL (the node that runs the
// from-binary and stops at the fork) and every launched pid, which the caller
// stops.
func runHandoffKeepDatadir(t *testing.T, fromBin, toBin string) (workspace, producerURL, producerDir string, pids []int) {
	t.Helper()
	t.Setenv("GWEMIX_BIN", fromBin)
	t.Setenv("GWBFT_BIN", toBin)
	spec := writeHandoffCase(t, nil)

	var lastOut string
	for attempt := 1; attempt <= govHandoffAttempts; attempt++ {
		dataDir, err := os.MkdirTemp("/tmp", "cbmigrate")
		if err != nil {
			t.Fatalf("mkdir temp workspace: %v", err)
		}
		cmd := newRootCmd()
		cmd.SetArgs([]string{
			"run", spec,
			"--workspace-dir", dataDir,
			"--keys", presetKeysDir(t),
			"--keep-up",
			"--node-monitor-timeout", "5m",
		})
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		runErr := cmd.Execute()

		if runErr == nil && strings.Contains(out.String(), "fail=0") {
			url, nodeDir, launched := producerRPC(t, dataDir)
			if attempt > 1 {
				t.Logf("handoff composed on attempt %d/%d", attempt, govHandoffAttempts)
			}
			return dataDir, url, nodeDir, launched
		}

		// The error Execute returns is the only account of a refusal cobra
		// never printed. Dropping it is what let "unknown command" read as a
		// flaky chain for fifteen days.
		lastOut = out.String()
		if runErr != nil {
			lastOut += "\nerror: " + runErr.Error()
		}
		if _, pids := recordedPIDs(dataDir); len(pids) > 0 {
			if leaks := stopPIDs(pids, 10*time.Second); len(leaks) > 0 {
				t.Logf("process: attempt %d leaked node PIDs %v", attempt, leaks)
			}
		}
		_ = os.RemoveAll(dataDir)
		t.Logf("handoff attempt %d/%d did not compose; retrying", attempt, govHandoffAttempts)
	}
	t.Fatalf("handoff not composed after %d attempts:\n%s", govHandoffAttempts, lastOut)
	return "", "", "", nil
}

// producerRPC answers the from-binary node's RPC URL and every launched pid. It
// is the mirror of successorRPC: the producer is the node the preset leaves on
// the default binary.
func producerRPC(t *testing.T, dataDir string) (url, nodeDir string, pids []int) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dataDir, "chain-record.json"))
	if err != nil {
		t.Fatalf("read chain record: %v", err)
	}
	var rec handoffNodes
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatalf("parse chain record: %v", err)
	}
	for _, n := range rec.Nodes {
		if n.PID > 0 {
			pids = append(pids, n.PID)
		}
		if url == "" && n.Binary != "next" && n.HTTP > 0 {
			url, nodeDir = fmt.Sprintf("http://%s:%d", n.Host, n.HTTP), n.DataDir
		}
	}
	if url == "" {
		t.Fatalf("no producer node in %s/chain-record.json", dataDir)
	}
	return url, nodeDir, pids
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
