//go:build e2e

// This E2E checks NODE-002 against the real handoff: the producer's exact
// pre-fork block, transaction and receipt survive an in-place datadir bridge,
// then the migrated go-wbft reader synchronizes and verifies post-fork blocks.
// It remains a reader, while the four recorded successors keep producing.
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
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixDataMigrationE2E(t *testing.T) {
	fromBin, toBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	workspace, producerURL, producerDir := runHandoffKeepDatadir(t, fromBin, toBin)
	producer := rpc.Dial(producerURL)
	preforkHead, err := producer.BlockNumber(ctx)
	if err != nil || preforkHead == 0 {
		t.Fatalf("producer head=%d: %v", preforkHead, err)
	}
	genesisPath := filepath.Join(workspace, "genesis.json")
	genesis, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Config struct {
			CroissantBlock uint64 `json:"croissantBlock"`
		} `json:"config"`
	}
	if err := json.Unmarshal(genesis, &config); err != nil {
		t.Fatal(err)
	}
	fork := config.Config.CroissantBlock
	if fork == 0 || preforkHead+1 != fork {
		t.Fatalf("producer head=%d, configured fork=%d", preforkHead, fork)
	}
	sample, txHash := migrationTransaction(t, ctx, producer, preforkHead)
	before, err := readMigrationSnapshot(ctx, producer, sample, txHash)
	if err != nil {
		t.Fatal(err)
	}
	successor := rpc.Dial(successorRPC(t, workspace))
	var info struct {
		Enode string `json:"enode"`
	}
	if err := successor.Call(ctx, "admin_nodeInfo", &info); err != nil {
		t.Fatal(err)
	}
	peer, err := url.Parse(info.Enode)
	if err != nil || peer.Port() == "" {
		t.Fatalf("successor peer=%q: %v", info.Enode, err)
	}
	peer.Host = net.JoinHostPort("127.0.0.1", peer.Port())
	records, err := os.ReadFile(filepath.Join(workspace, "chain-record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec struct {
		Nodes []struct {
			Index   int    `json:"index"`
			DataDir string `json:"dataDir"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(records, &rec); err != nil {
		t.Fatal(err)
	}
	index := 0
	for _, n := range rec.Nodes {
		if n.DataDir == producerDir {
			index = n.Index
		}
	}
	if index == 0 {
		t.Fatal("producer index missing from actual node record")
	}
	if err := app.NodeStop(ctx, app.Deps{}, app.NodeStopIn{DataDir: workspace, Index: index}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(producerDir, "geth", "chaindata")); err != nil {
		t.Fatal(err)
	}
	if err := bridgeMigrationData(producerDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(filepath.Join(producerDir, "gwemix")); err != nil {
			t.Error(err)
		}
	})
	if err := process.InitDatadir(ctx, toBin, producerDir, genesisPath); err != nil {
		t.Fatal(err)
	}
	readerURL := launchOfflineReader(t, toBin, producerDir)
	reader := rpc.Dial(readerURL)
	deadline := time.Now().Add(time.Minute)
	for {
		head, err := reader.BlockNumber(ctx)
		if err == nil {
			if head != preforkHead {
				t.Fatalf("offline migrated head=%d, want exactly %d", head, preforkHead)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("migrated reader RPC unavailable", err)
		}
		time.Sleep(time.Second)
	}
	after, err := readMigrationSnapshot(ctx, reader, sample, txHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := compareMigrationSnapshots(before, after); err != nil {
		t.Fatal(err)
	}
	var added bool
	if err := reader.Call(ctx, "admin_addPeer", &added, peer.String()); err != nil || !added {
		t.Fatalf("add successor peer=%v: %v", added, err)
	}
	deadline = time.Now().Add(90 * time.Second)
	observations := []uint64{preforkHead}
	var accepted uint64
	var connected bool
	for {
		head, err := reader.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if head < observations[len(observations)-1] {
			t.Fatal("migrated reader head decreased")
		}
		observations = append(observations, head)
		var peers string
		if err := reader.Call(ctx, "net_peerCount", &peers); err != nil {
			t.Fatal(err)
		}
		if err := validateW3Observation(head, head, peers); err != nil {
			t.Fatal(err)
		}
		connected = connected || peers != "0x0"
		if connected && head >= fork+2 {
			accepted = head
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("migrated reader failed to cross fork=%d; observations=%v connected=%v", fork, observations, connected)
		}
		time.Sleep(time.Second)
	}
	var readerBlock, successorBlock migrationBlock
	if err := reader.Call(ctx, "eth_getBlockByNumber", &readerBlock, hexUint(accepted), false); err != nil {
		t.Fatal(err)
	}
	if err := successor.Call(ctx, "eth_getBlockByNumber", &successorBlock, hexUint(accepted), false); err != nil {
		t.Fatal(err)
	}
	if readerBlock != successorBlock || readerBlock.Hash == "" || readerBlock.StateRoot == "" {
		t.Fatal("post-fork block differs from live successor", readerBlock, successorBlock)
	}
	stable, err := readMigrationSnapshot(ctx, reader, sample, txHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := compareMigrationSnapshots(before, stable); err != nil {
		t.Fatal(err)
	}
	body, err := json.MarshalIndent(map[string]any{"workspace": workspace, "sourceBinary": fromBin, "successorBinary": toBin, "producerIndex": index, "preforkHead": preforkHead, "fork": fork, "transactionHash": txHash, "before": before, "after": after, "afterSync": stable, "readerHeadObservations": observations, "acceptedPostforkBlock": readerBlock, "connected": connected}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	proof := filepath.Join(filepath.Dir(workspace), "reports/migration-proof.json")
	if err := os.WriteFile(proof, body, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("NODE-002 exact data and post-fork synchronization proof:", proof)
}

type migrationBlock struct {
	Hash      string `json:"hash"`
	StateRoot string `json:"stateRoot"`
	Number    string `json:"number"`
}
type migrationSnapshot struct {
	Block       migrationBlock
	Transaction map[string]any
	Receipt     map[string]any
}

func migrationTransaction(t *testing.T, ctx context.Context, c *rpc.Client, head uint64) (uint64, string) {
	t.Helper()
	for n := head; n > 0; n-- {
		var block struct {
			Transactions []string `json:"transactions"`
		}
		if err := c.Call(ctx, "eth_getBlockByNumber", &block, hexUint(n), false); err != nil {
			t.Fatal(err)
		}
		if len(block.Transactions) > 0 {
			return n, block.Transactions[0]
		}
	}
	t.Fatal("producer history has no mined transaction")
	return 0, ""
}
func readMigrationSnapshot(ctx context.Context, c *rpc.Client, number uint64, hash string) (migrationSnapshot, error) {
	var s migrationSnapshot
	if err := c.Call(ctx, "eth_getBlockByNumber", &s.Block, hexUint(number), false); err != nil {
		return s, err
	}
	for _, query := range []struct {
		method string
		out    *map[string]any
		fields []string
	}{
		{"eth_getTransactionByHash", &s.Transaction, []string{"hash", "blockHash", "blockNumber", "transactionIndex", "from", "to", "value", "input", "nonce", "type"}},
		{"eth_getTransactionReceipt", &s.Receipt, []string{"transactionHash", "blockHash", "blockNumber", "transactionIndex", "status", "gasUsed", "cumulativeGasUsed", "effectiveGasPrice", "logs", "logsBloom", "contractAddress", "from", "to"}},
	} {
		var raw map[string]any
		if err := c.Call(ctx, query.method, &raw, hash); err != nil {
			return s, err
		}
		selected := map[string]any{}
		for _, field := range query.fields {
			value, ok := raw[field]
			if !ok {
				return s, fmt.Errorf("%s missing %s", query.method, field)
			}
			selected[field] = value
		}
		*query.out = selected
	}
	if s.Block.Hash == "" || s.Block.StateRoot == "" || s.Block.Number != hexUint(number) || s.Transaction["hash"] != hash || s.Receipt["transactionHash"] != hash || s.Transaction["blockHash"] != s.Block.Hash || s.Receipt["blockHash"] != s.Block.Hash || s.Transaction["blockNumber"] != s.Block.Number || s.Receipt["blockNumber"] != s.Block.Number {
		return s, fmt.Errorf("inconsistent or empty migration RPC data")
	}
	return s, nil
}
func compareMigrationSnapshots(before, after migrationSnapshot) error {
	if before.Block.Hash == "" || before.Block.StateRoot == "" || len(before.Transaction) == 0 || len(before.Receipt) == 0 {
		return fmt.Errorf("migration baseline missing mined data")
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("migrated block, transaction or receipt differs from producer")
	}
	return nil
}
func TestMigrationSnapshotRejectsChangedData(t *testing.T) {
	original := migrationSnapshot{Block: migrationBlock{Hash: "0xblock", StateRoot: "0xstate", Number: "0x9"}, Transaction: map[string]any{"hash": "0xtx", "input": "0x12"}, Receipt: map[string]any{"status": "0x1", "gasUsed": "0x123"}}
	for _, field := range []string{"identical", "block hash", "state root", "transaction input", "receipt gas", "missing transaction"} {
		t.Run(field, func(t *testing.T) {
			body, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			var changed migrationSnapshot
			if err := json.Unmarshal(body, &changed); err != nil {
				t.Fatal(err)
			}
			switch field {
			case "block hash":
				changed.Block.Hash = "0xother"
			case "state root":
				changed.Block.StateRoot = "0xother"
			case "transaction input":
				changed.Transaction["input"] = "0x99"
			case "receipt gas":
				changed.Receipt["gasUsed"] = "0x999"
			case "missing transaction":
				changed.Transaction = nil
			}
			if err := compareMigrationSnapshots(original, changed); (err != nil) != (field != "identical") {
				t.Fatalf("comparison=%v for %s", err, field)
			}
		})
	}
}

func bridgeMigrationData(dataDir string) error {
	alias := filepath.Join(dataDir, "gwemix")
	if _, err := os.Lstat(alias); err == nil {
		return fmt.Errorf("migration destination already exists: %s", alias)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Symlink("geth", alias)
}

func TestMigrationBridgePreservesExistingDestination(t *testing.T) {
	for _, kind := range []string{"directory", "file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "geth", "chaindata"), 0700); err != nil {
				t.Fatal(err)
			}
			alias := filepath.Join(root, "gwemix")
			sentinel := alias
			switch kind {
			case "directory":
				if err := os.Mkdir(alias, 0700); err != nil {
					t.Fatal(err)
				}
				sentinel = filepath.Join(alias, "preserve")
			case "symlink":
				if err := os.Symlink("geth", alias); err != nil {
					t.Fatal(err)
				}
				sentinel = filepath.Join(root, "geth", "preserve")
			}
			if err := os.WriteFile(sentinel, []byte("existing data"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := bridgeMigrationData(root); err == nil {
				t.Fatal("bridge accepted an existing destination")
			}
			if body, err := os.ReadFile(sentinel); err != nil || string(body) != "existing data" {
				t.Fatalf("existing data changed: %q %v", body, err)
			}
		})
	}
}

// runHandoffKeepDatadir composes the handoff like runGovHandoff but hands the
// workspace back instead of tearing it down: this test reads the producer's
// chaindata after the network stops, which is the thing it is checking.
//
// It returns the workspace, the PRODUCER's RPC URL (the node that runs the
// from-binary and stops at the fork) which the caller stops through the workspace record.
func runHandoffKeepDatadir(t *testing.T, fromBin, toBin string) (workspace, producerURL, producerDir string) {
	t.Helper()
	t.Setenv("GWEMIX_BIN", fromBin)
	t.Setenv("GWBFT_BIN", toBin)
	spec := writeHandoffCase(t, nil)

	var lastOut string
	for attempt := 1; attempt <= govHandoffAttempts; attempt++ {
		runtime := createHandoffRuntime(t, spec)
		dataDir := filepath.Join(runtime, "w")
		cmd := newRootCmd()
		cmd.SetArgs([]string{
			"run", spec,
			"--workspace-dir", dataDir,
			"--artifact-root", filepath.Join(runtime, "reports"),
			"--keys", presetKeysDir(t),
			"--read-only-keys",
			"--no-skips",
			"--keep-up",
			"--node-monitor-timeout", "5m",
		})
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		runErr := cmd.Execute()
		preserveHandoffRun(t, runtime, out.String(), runErr)

		if runErr == nil && strings.Contains(out.String(), "fail=0") {
			url, nodeDir := producerRPC(t, dataDir)
			if attempt > 1 {
				t.Logf("handoff composed on attempt %d/%d", attempt, govHandoffAttempts)
			}
			return dataDir, url, nodeDir
		}

		// The error Execute returns is the only account of a refusal cobra
		// never printed. Dropping it is what let "unknown command" read as a
		// flaky chain for fifteen days.
		lastOut = out.String()
		if runErr != nil {
			lastOut += "\nerror: " + runErr.Error()
		}
		if err := closeHandoffRuntime(runtime); err != nil {
			t.Fatalf("failed migration attempt cannot be reclaimed; no retry: %v", err)
		}
		t.Logf("handoff attempt %d/%d did not compose; retrying", attempt, govHandoffAttempts)
	}
	t.Fatalf("handoff not composed after %d attempts:\n%s", govHandoffAttempts, lastOut)
	return "", "", ""
}

// producerRPC answers the from-binary node's RPC URL and data directory. It
// is the mirror of successorRPC: the producer is the node the preset leaves on
// the default binary.
func producerRPC(t *testing.T, dataDir string) (url, nodeDir string) {
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
		if url == "" && n.Binary != "next" && n.HTTP > 0 {
			url, nodeDir = fmt.Sprintf("http://%s:%d", n.Host, n.HTTP), n.DataDir
		}
	}
	if url == "" {
		t.Fatalf("no producer node in %s/chain-record.json", dataDir)
	}
	return url, nodeDir
}

// launchOfflineReader starts the go-wbft binary on a datadir as a non-mining,
// non-discovering reader with an isolated RPC and returns its RPC URL. The
// process is killed on test cleanup.
func launchOfflineReader(t *testing.T, binary, dataDir string) string {
	return launchHandoffReader(t, binary, dataDir, filepath.Join(filepath.Dir(filepath.Dir(dataDir)), "reports/migration-reader.log"))
}
