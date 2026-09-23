//go:build e2e

// This E2E ports the remaining reachable wemix4 NCP-governance WRITE flows —
// GOV-007 (remove an NCP by vote of the others) and GOV-008 (immediate self-exit)
// — as one coherent NCP lifecycle on the go-wbft handoff successor. It builds on
// the GOV-006 add flow: the preset validator accounts (raw keys in presets/keys)
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
//	go test -tags e2e -run TestWemixGovernanceNCPLifecycleE2E -timeout 8m ./cmd/chainbench
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixGovernanceNCPLifecycleE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	url := runGovHandoff(t, fromBin, toBin)
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

// govHandoffAttempts is how many times the handoff is (re)composed before
// giving up. The go-wemix producer's embedded etcd intermittently fails to
// bootstrap, so a single bring-up is flaky; each failed attempt is torn down
// cleanly before the retry.
const govHandoffAttempts = 4

// handoffPresetPath is the declaration the handoff is composed from.
//
// `chainbench upgrade run` used to do this, and #419 retired it: the handoff no
// longer has a composer of its own, it goes through the ordinary composition
// path like any other network. What the command took as flags this file
// declares — both binaries, the croissant fork and the block it lands on.
const handoffPresetPath = "../../presets/chain/wemix-to-wbft.json"

// presetKeysDir answers the shipped key set as an absolute path.
//
// The preset names it "presets/keys", which resolves against the process
// working directory — this package's own directory when a test runs, where
// there is no such thing.
func presetKeysDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "presets", "keys"))
	if err != nil {
		t.Fatalf("resolve preset keys: %v", err)
	}
	return abs
}

// handoffProducer is the account the producer mines and stakes with before the
// fork. presets/chain/wemix-upgrade.yaml names it as the from-chain member, and
// the preset's node5 keystore holds it rather than node5's own address.
const handoffProducer = "0xf9593d358b373d354a348c00887b914b408f6984"

// handoffBalance funds an account past what any of these flows spends:
// minimumStaking is 1e25 and a test may stake several times.
const handoffBalance = "0x33b2e3c9fd0803ce8000000" // 1e27 wei

// baseOverlay is the genesis state every one of these flows starts from.
//
// The retired command took both of these from its profile. The chain-preset
// declares neither, because it describes a network crossing a fork and not a
// governance exercise — so the exercise brings its own, the way the two tests
// that already declared an overlay did.
//
//   - Funding. Without it a flow fails at its first transaction with
//     "insufficient funds for gas * price + value", which says nothing about
//     what it was testing.
//   - One NCP. The wbft genesis template seeds govNCP from the member CSV, so
//     the four validators all start as NCPs and a proposal needs three votes.
//     These flows are written against a SOLE initial NCP that can carry a
//     proposal alone, which is what the handoff produced. A test that wants a
//     different council declares it and wins.
func baseOverlay(t *testing.T) map[string]any {
	t.Helper()
	alloc := map[string]any{
		handoffProducer: map[string]any{"balance": handoffBalance},
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "presets", "keys", "metadata.json"))
	if err != nil {
		t.Fatalf("read preset metadata: %v", err)
	}
	var m struct {
		Nodes []struct {
			Address string `json:"address"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse preset metadata: %v", err)
	}
	for _, n := range m.Nodes {
		if n.Address != "" {
			alloc[n.Address] = map[string]any{"balance": handoffBalance}
		}
	}
	return map[string]any{
		"alloc": alloc,
		"config": map[string]any{
			"croissant": map[string]any{
				"govContracts": map[string]any{
					"govNCP": map[string]any{
						"params": map[string]any{"ncps": presetNodeAddr(t, 1)},
					},
				},
			},
		},
	}
}

// mergeOverlay lays a test's overlay over the state above, leaf by leaf. Two
// maps at the same key are merged rather than replaced, so a test that tunes
// one wBFT parameter does not thereby drop the funding or the council it did
// not mention.
func mergeOverlay(base, over map[string]any) map[string]any {
	for k, v := range over {
		bm, bok := base[k].(map[string]any)
		vm, vok := v.(map[string]any)
		if bok && vok {
			base[k] = mergeOverlay(bm, vm)
			continue
		}
		base[k] = v
	}
	return base
}

// overlayFromJSON reads the {"genesis": {...}} document a test builds and
// returns the genesis object inside it, which is what the chain-preset declares.
func overlayFromJSON(t *testing.T, doc string) map[string]any {
	t.Helper()
	var wrapper struct {
		Genesis map[string]any `json:"genesis"`
	}
	if err := json.Unmarshal([]byte(doc), &wrapper); err != nil {
		t.Fatalf("parse genesis overlay: %v\n%s", err, doc)
	}
	return wrapper.Genesis
}

// writeHandoffCase writes the case the handoff is composed from: the preset
// INLINE, the genesis state these flows need, and a wait past the fork block so
// the successors have taken production over before a test asks them anything.
//
// The preset is inlined rather than named by id because the case is written to
// a temp directory, and a named preset is looked for beside the case or under
// presets/chain — neither of which is there. Reading the committed file keeps
// the test on the same declaration the shipped case uses.
func writeHandoffCase(t *testing.T, overlay map[string]any) string {
	t.Helper()
	b, err := os.ReadFile(handoffPresetPath)
	if err != nil {
		t.Fatalf("read handoff preset: %v", err)
	}
	var preset map[string]any
	if err := json.Unmarshal(b, &preset); err != nil {
		t.Fatalf("parse handoff preset: %v", err)
	}
	merged := mergeOverlay(baseOverlay(t), overlay)
	withForkBlock(t, preset, merged)
	preset["genesis"] = map[string]any{"overlay": merged}
	doc := map[string]any{
		"schemaVersion": "2",
		"kind":          "case",
		"id":            "gov-handoff",
		"description":   "composes the wemix->wbft handoff and waits past the fork",
		"requires":      []string{"rpc", "consensus"},
		"chainPreset":   preset,
		"steps": []map[string]any{
			{"do": "waitBlock", "target": 22, "timeout": "300s"},
			{"expect": "blockNumber", "compare": "Greater", "is": "20"},
		},
	}
	enc, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		t.Fatalf("encode handoff case: %v", err)
	}
	path := filepath.Join(t.TempDir(), "gov-handoff.json")
	if err := os.WriteFile(path, enc, 0o644); err != nil {
		t.Fatalf("write handoff case: %v", err)
	}
	return path
}

// withForkBlock puts the fork block beside a croissant section the overlay
// adds. The genesis refuses "croissantBlock and the croissant config section
// must be set together", and an overlay that tunes wBFT parameters brings the
// section without the block — the preset's upgrade.at is where the block comes
// from, so the test does not repeat a number the declaration already owns.
func withForkBlock(t *testing.T, preset, overlay map[string]any) {
	t.Helper()
	cfg, ok := overlay["config"].(map[string]any)
	if !ok {
		return
	}
	if _, has := cfg["croissant"]; !has {
		return
	}
	if _, has := cfg["croissantBlock"]; has {
		return
	}
	up, ok := preset["upgrade"].(map[string]any)
	if !ok {
		t.Fatalf("preset declares no upgrade block, so there is no fork block to set")
	}
	at, ok := up["at"].(float64)
	if !ok {
		t.Fatalf("preset upgrade.at is %T, want a number", up["at"])
	}
	cfg["croissantBlock"] = int(at)
}

// handoffNodes is what the workspace records about each node the run launched.
type handoffNodes struct {
	Nodes []struct {
		Label   string `json:"label"`
		Host    string `json:"host"`
		HTTP    int    `json:"http"`
		PID     int    `json:"pid"`
		Binary  string `json:"binary"`
		DataDir string `json:"dataDir"`
	} `json:"nodes"`
}

// successorRPC reads the record the run wrote and answers a successor's RPC
// URL, with every launched pid beside it for teardown.
//
// The retired command printed both, and the tests parsed its stdout. The record
// is the better source: it is what resume, status and stop already read, so a
// test asks the same question the product does.
func successorRPC(t *testing.T, dataDir string) (string, []int) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dataDir, "chain-record.json"))
	if err != nil {
		t.Fatalf("read chain record: %v", err)
	}
	var rec handoffNodes
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatalf("parse chain record: %v", err)
	}
	var url string
	var pids []int
	for _, n := range rec.Nodes {
		if n.PID > 0 {
			pids = append(pids, n.PID)
		}
		// "next" is the successor binary the preset declares; the producer runs
		// "default" and stops at the fork, so it can answer nothing after it.
		if url == "" && n.Binary == "next" && n.HTTP > 0 {
			url = fmt.Sprintf("http://%s:%d", n.Host, n.HTTP)
		}
	}
	if url == "" {
		t.Fatalf("no successor node in %s/chain-record.json", dataDir)
	}
	return url, pids
}

// runGovHandoff composes the handoff, leaves it up, and returns a successor's
// RPC URL. It uses a SHORT /tmp workspace so node1's IPC socket path stays under
// the ~104-byte unix-socket limit, and tears every launched node down at the end.
func runGovHandoff(t *testing.T, fromBin, toBin string) string {
	return runGovHandoffOverlay(t, fromBin, toBin, nil)
}

// runGovHandoffOverlay is runGovHandoff with a genesis overlay deep-merged into
// the handoff genesis — the shape `upgrade run --genesis-overlay` took, declared
// on the chain-preset instead of passed as a flag.
func runGovHandoffOverlay(t *testing.T, fromBin, toBin string, overlay map[string]any) string {
	t.Helper()
	// The preset names its binaries through these, so a test points them at the
	// builds it was given rather than at whatever is on PATH.
	t.Setenv("GWEMIX_BIN", fromBin)
	t.Setenv("GWBFT_BIN", toBin)
	spec := writeHandoffCase(t, overlay)

	var lastOut string
	for attempt := 1; attempt <= govHandoffAttempts; attempt++ {
		dataDir, err := os.MkdirTemp("/tmp", "cbgov")
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
			url, pids := successorRPC(t, dataDir)
			t.Cleanup(func() {
				if leaks := stopPIDs(pids, 10*time.Second); len(leaks) > 0 {
					t.Logf("process: leaked node PIDs after test: %v", leaks)
				}
				_ = os.RemoveAll(dataDir)
			})
			if attempt > 1 {
				t.Logf("handoff composed on attempt %d/%d", attempt, govHandoffAttempts)
			}
			return url
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
	return ""
}

// recordedPIDs reads the pids a failed attempt left behind, so teardown can
// reach nodes the run launched before it gave up. A workspace with no record
// launched nothing.
func recordedPIDs(dataDir string) (bool, []int) {
	b, err := os.ReadFile(filepath.Join(dataDir, "chain-record.json"))
	if err != nil {
		return false, nil
	}
	var rec handoffNodes
	if err := json.Unmarshal(b, &rec); err != nil {
		return false, nil
	}
	var pids []int
	for _, n := range rec.Nodes {
		if n.PID > 0 {
			pids = append(pids, n.PID)
		}
	}
	return true, pids
}

// presetNodeKey loads node idx's raw private key from presets/keys/metadata.json.
func presetNodeKey(t *testing.T, idx int) []byte {
	t.Helper()
	_, keyHex := presetNode(t, idx)
	key, err := hex.DecodeString(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		t.Fatalf("decode node%d key: %v", idx, err)
	}
	return key
}

// presetNodeAddr returns node idx's address from presets/keys/metadata.json.
func presetNodeAddr(t *testing.T, idx int) string {
	t.Helper()
	addr, _ := presetNode(t, idx)
	return addr
}

// presetNode returns node idx's (address, nodekey) from the preset metadata.
func presetNode(t *testing.T, idx int) (addr, nodekey string) {
	t.Helper()
	b, err := os.ReadFile("../../presets/keys/metadata.json")
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

// presetNode1Key loads node 1's private key from presets/keys — a committed TEST
// fixture (public, local-only) whose address is genesis-funded.

func presetNode1Key(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "presets", "keys", "metadata.json"))
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
