//go:build e2e

// This E2E ports the remaining reachable wemix4 NCP-governance WRITE flows —
// GOV-007 (remove an NCP by vote of the others) and GOV-008 (immediate self-exit)
// — as one coherent NCP lifecycle on the go-wbft handoff successor. It builds on
// the GOV-006 add flow: the preset validator accounts (raw keys in keys/preset)
// are the NCP electorate, and quorum is ceil(2*ncpCount/3).
//
//	add node2 (quorum 1)  -> ncpCount 1->2
//	add node3 (quorum 2)  -> ncpCount 2->3
//	remove node3 by vote  -> ncpCount 3->2   (GOV-007)
//	node2 self-exit       -> ncpCount 2->1   (GOV-008, immediate, no vote)
//
// Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestWemixGovernanceNCPLifecycleE2E -timeout 8m ./cmd/chainbench
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/procman"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

func TestWemixGovernanceNCPLifecycleE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	url := runGovHandoff(t, fromBin, toBin, template)
	c := rpc.Dial(url)
	ctx := context.Background()

	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	open := func(idx int) accounts.Wallet {
		w, err := ap.OpenWallet(ctx, presetNodeKey(t, idx), url)
		if err != nil {
			t.Fatalf("open wallet node%d: %v", idx, err)
		}
		return w
	}
	// node1 is the sole initial NCP; node2/node3 are the candidates.
	ncp1, ncp2 := open(1), open(2)
	addr2, addr3 := presetNodeAddr(t, 2), presetNodeAddr(t, 3)

	expectCount := func(want int64, what string) {
		if got := ncpCount(t, c); got.Cmp(big.NewInt(want)) != 0 {
			t.Fatalf("%s: ncpCount = %s, want %d", what, got, want)
		}
	}
	expectCount(1, "start")

	// GOV-006: add node2 (quorum 1 -> propose + one vote by node1).
	addNCP(t, c, ncp1, nil, addr2)
	if !ncpIsMember(t, c, addr2) {
		t.Fatal("node2 not an NCP after add")
	}
	expectCount(2, "after add node2")

	// Add node3 (quorum is now ceil(2*2/3)=2 -> node1 and node2 both vote).
	addNCP(t, c, ncp1, []accounts.Wallet{ncp1, ncp2}, addr3)
	if !ncpIsMember(t, c, addr3) {
		t.Fatal("node3 not an NCP after add")
	}
	expectCount(3, "after add node3")

	// GOV-007: remove node3 by vote of the others (quorum 2 -> node1 + node2).
	removeNCPByVote(t, c, ncp1, []accounts.Wallet{ncp1, ncp2}, addr3)
	if ncpIsMember(t, c, addr3) {
		t.Fatal("node3 still an NCP after remove-by-vote")
	}
	expectCount(2, "after remove node3")

	// GOV-008: node2 self-exit (proposing to remove itself executes immediately).
	rc := ncpExecute(t, c, ncp2, accounts.EncodeCallArgs("newProposalToRemoveNCP(address)", accounts.Address(addr2)))
	if rc.Status != "0x1" {
		t.Fatalf("self-exit reverted (status %s)", rc.Status)
	}
	if ncpIsMember(t, c, addr2) {
		t.Fatal("node2 still an NCP after self-exit")
	}
	expectCount(1, "after node2 self-exit")
}

// addNCP proposes adding candidate and votes the ballot through. voters is the
// set of NCP wallets that must vote to meet quorum; nil means the proposer's own
// single vote suffices (quorum 1).
func addNCP(t *testing.T, c *rpc.Client, proposer accounts.Wallet, voters []accounts.Wallet, candidate string) {
	t.Helper()
	passBallot(t, c, proposer, voters, "newProposalToAddNCP(address)", candidate)
}

// removeNCPByVote proposes removing target and votes the ballot through.
func removeNCPByVote(t *testing.T, c *rpc.Client, proposer accounts.Wallet, voters []accounts.Wallet, target string) {
	t.Helper()
	passBallot(t, c, proposer, voters, "newProposalToRemoveNCP(address)", target)
}

// passBallot submits proposalSig(subject), extracts the ballot id from the
// receipt, and casts an accept vote from each voter (or just the proposer when
// voters is nil).
func passBallot(t *testing.T, c *rpc.Client, proposer accounts.Wallet, voters []accounts.Wallet, proposalSig, subject string) {
	t.Helper()
	rc := ncpExecute(t, c, proposer, accounts.EncodeCallArgs(proposalSig, accounts.Address(subject)))
	if rc.Status != "0x1" {
		t.Fatalf("%s(%s) reverted (status %s)", proposalSig, subject, rc.Status)
	}
	if len(rc.Logs) == 0 || len(rc.Logs[0].Topics) < 2 {
		t.Fatalf("no ballot id in receipt for %s(%s): %+v", proposalSig, subject, rc.Logs)
	}
	ballot, ok := new(big.Int).SetString(strings.TrimPrefix(rc.Logs[0].Topics[1], "0x"), 16)
	if !ok {
		t.Fatalf("ballot id not hex: %s", rc.Logs[0].Topics[1])
	}
	if voters == nil {
		voters = []accounts.Wallet{proposer}
	}
	for i, v := range voters {
		vr := ncpExecute(t, c, v, accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(ballot), accounts.Word([]byte{1})))
		if vr.Status != "0x1" {
			t.Fatalf("%s(%s): voter %d reverted (status %s)", proposalSig, subject, i, vr.Status)
		}
	}
}

// govHandoffAttempts is how many times runGovHandoff will (re)launch the handoff
// before giving up. The go-wemix producer's embedded etcd intermittently fails to
// bootstrap (it enters a join-failure loop and the chain halts before the fork),
// so a single launch is flaky; each failed attempt is torn down cleanly (via
// procman, so no orphaned node process survives to hold etcd's ports and poison
// the next attempt) and retried.
const govHandoffAttempts = 4

// govHandoffWait is the per-attempt handoff poll (seconds). A healthy producer
// crosses the fork within ~30s; this bounds how long a halted attempt wastes
// before the retry.
const govHandoffWait = "100"

// runGovHandoff launches the `upgrade run` handoff and returns a successor RPC
// URL, retrying the flaky producer/etcd bootstrap. It uses a SHORT /tmp datadir
// so node1's IPC socket path stays under the ~104-byte unix-socket limit (long
// t.TempDir() paths break the producer's IPC), and a procman.Manager so every
// launched node is tracked and verifiably killed on teardown (between retries and
// at test end) — no orphans.
func runGovHandoff(t *testing.T, fromBin, toBin, template string) string {
	t.Helper()
	var lastOut string
	for attempt := 1; attempt <= govHandoffAttempts; attempt++ {
		dataDir, err := os.MkdirTemp("/tmp", "cbgovlc")
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
		// Track whatever launched (PIDs are printed as `pid=N`, and mirrored into
		// nodeset.json) so we can guarantee teardown either way.
		mgr.TrackFromOutput(out.String())
		mgr.TrackNodeSet(dataDir)

		if runErr == nil && strings.Contains(out.String(), "handoff confirmed") {
			// Success: keep the nodes running for the test, tear down verifiably at
			// the end.
			dir := dataDir
			t.Cleanup(func() {
				if leaks := mgr.StopAll(10 * time.Second); len(leaks) > 0 {
					t.Logf("procman: leaked node PIDs after test: %v", leaks)
				}
				_ = os.RemoveAll(dir)
			})
			if attempt > 1 {
				t.Logf("handoff confirmed on attempt %d/%d", attempt, govHandoffAttempts)
			}
			return successorRPC(t, out.String())
		}

		// Failure: kill this attempt's nodes cleanly (no orphans) before retrying.
		lastOut = out.String()
		if leaks := mgr.StopAll(10 * time.Second); len(leaks) > 0 {
			t.Logf("procman: attempt %d leaked node PIDs %v", attempt, leaks)
		}
		_ = os.RemoveAll(dataDir)
		t.Logf("handoff attempt %d/%d failed (flaky producer/etcd bootstrap); retrying", attempt, govHandoffAttempts)
	}
	t.Fatalf("handoff not confirmed after %d attempts:\n%s", govHandoffAttempts, lastOut)
	return ""
}

// presetNodeKey loads node idx's raw private key from keys/preset/metadata.json.
func presetNodeKey(t *testing.T, idx int) []byte {
	t.Helper()
	_, keyHex := presetNode(t, idx)
	key, err := hex.DecodeString(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		t.Fatalf("decode node%d key: %v", idx, err)
	}
	return key
}

// presetNodeAddr returns node idx's address from keys/preset/metadata.json.
func presetNodeAddr(t *testing.T, idx int) string {
	t.Helper()
	addr, _ := presetNode(t, idx)
	return addr
}

// presetNode returns node idx's (address, nodekey) from the preset metadata.
func presetNode(t *testing.T, idx int) (addr, nodekey string) {
	t.Helper()
	b, err := os.ReadFile("../../keys/preset/metadata.json")
	if err != nil {
		t.Fatalf("read preset metadata: %v", err)
	}
	var m struct {
		Nodes []struct {
			Index   int    `json:"index"`
			NodeKey string `json:"nodekey"`
			Address string `json:"address"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse preset metadata: %v", err)
	}
	for _, n := range m.Nodes {
		if n.Index == idx {
			return n.Address, n.NodeKey
		}
	}
	t.Fatalf("no node %d in preset metadata", idx)
	return "", ""
}
