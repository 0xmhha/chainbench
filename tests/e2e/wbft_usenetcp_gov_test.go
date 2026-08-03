//go:build e2e

// These E2Es port wemix4 GOV-005 (staker registration reflected in the validator
// set) and GOV-009 (a non-NCP staker never becomes a validator) on a standalone
// go-wbft network with useNCP-driven validator selection.
//
// go-wbft's useNCP selection is: candidates = { StakerByOperator[ncp] : ncp in
// govNCP, and that staker is active } — i.e. an NCP member acts as an OPERATOR
// and the staker it registered is the validator candidate. Once the candidate
// count reaches stabilizingStakersThreshold the epoch leaves the stabilizing
// stage and the validator set is decided as the top targetValidators candidates
// by (stake, diligence). The authoritative governance signal is the epoch
// boundary's EpochInfo (istanbul_getWbftExtraInfo): its `stabilizing` flag,
// `stakers` (candidate) list, and decided `validators` list.
//
// The setup keeps the NCP operators (nodes 4,5,6 — a genesis govNCP overlay)
// distinct from the mining validators/stakers (nodes 1,2,3), and uses a small
// targetValidators=3 / stabilizingStakersThreshold=2 so the flow is feasible on a
// generated 8-node preset. Requires the go-wbft bootnode tool to generate the
// preset:
//
//	WBFT_BIN=/path/go-wbft/build/bin/gwemix \
//	BOOTNODE_BIN=/path/go-wbft/build/bin/bootnode \
//	go test -tags e2e -run TestE2E_WbftUseNCP -v ./tests/e2e/
package e2e

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

const (
	govStakingAddr  = "0x0000000000000000000000000000000000001001"
	govConfigAddr   = "0x0000000000000000000000000000000000001000"
	usenetcpEpochLn = 10
)

// TestE2E_WbftUseNCPValidatorGrowth ports GOV-005: registering an NCP staker
// grows the governance-decided validator set. Starting from the stabilizing stage
// (no registered stakers), it registers two NCP stakers to cross the threshold
// (the epoch leaves stabilizing with a 2-validator set), then a third NCP staker
// and asserts the decided set grows to three including the new staker.
func TestE2E_WbftUseNCPValidatorGrowth(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)
	preset := genPresetNV(t, cli, bin, 8, 3)
	nodes := readPresetDir(t, preset)

	n := bootUseNCP(t, cli, bin, preset, nodes)
	url := n.rpcURLFor(1)
	c := rpc.Dial(url)

	minStake := ncpUint(t, c, govConfigAddr, "minimumStaking()")

	// Two NCP stakers (validators 1,2 via NCP operators 4,5) cross threshold 2.
	registerStakerVia(t, n, url, nodes, 1, 4, minStake)
	registerStakerVia(t, n, url, nodes, 2, 5, minStake)
	ei := waitStakerCount(t, c, 2, 90*time.Second)
	if ei.Stabilizing {
		t.Fatalf("still stabilizing with 2 NCP stakers (threshold 2) — should have left the stabilizing stage")
	}
	if len(ei.Validators) != 2 {
		t.Fatalf("decided validator set = %d after 2 NCP stakers, want 2", len(ei.Validators))
	}

	// A third NCP staker (validator 3 via NCP operator 6) grows the set to three.
	registerStakerVia(t, n, url, nodes, 3, 6, minStake)
	ei = waitStakerCount(t, c, 3, 90*time.Second)
	if len(ei.Validators) != 3 {
		t.Fatalf("decided validator set = %d after 3rd NCP staker, want 3 (set did not grow)", len(ei.Validators))
	}
	if !ei.hasValidator(nodes[3].Address) {
		t.Fatalf("newly registered staker node3 (%s) not in decided validator set %v", nodes[3].Address, ei.validatorAddrs())
	}
}

// TestE2E_WbftUseNCPNonNCPExcluded ports GOV-009: a staker registered through a
// NON-NCP operator never becomes a validator, even with a far larger stake than
// the NCP stakers. It first crosses the threshold with two NCP stakers (so the
// set is live, not frozen), then registers a large-stake staker via a non-NCP
// operator and asserts the candidate/validator set is unchanged.
func TestE2E_WbftUseNCPNonNCPExcluded(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)
	preset := genPresetNV(t, cli, bin, 8, 3)
	nodes := readPresetDir(t, preset)

	n := bootUseNCP(t, cli, bin, preset, nodes)
	url := n.rpcURLFor(1)
	c := rpc.Dial(url)

	minStake := ncpUint(t, c, govConfigAddr, "minimumStaking()")

	registerStakerVia(t, n, url, nodes, 1, 4, minStake)
	registerStakerVia(t, n, url, nodes, 2, 5, minStake)
	ei := waitStakerCount(t, c, 2, 90*time.Second)
	if ei.Stabilizing {
		t.Fatalf("still stabilizing with 2 NCP stakers — cannot test exclusion on a frozen set")
	}

	// node8 registered via NON-NCP operator node7, with 5x the NCP stakers' stake.
	bigStake := new(big.Int).Mul(minStake, big.NewInt(5))
	registerStakerVia(t, n, url, nodes, 8, 7, bigStake)

	// Give it two epochs to (not) take effect, then confirm the candidate set is
	// unchanged and node8 is absent from the decided validators.
	time.Sleep((2*usenetcpEpochLn + 5) * time.Second)
	after := latestEpochInfo(t, c)
	if len(after.Stakers) != 2 {
		t.Fatalf("candidate set = %d after a non-NCP staker registered, want 2 (non-NCP staker must not be a candidate)", len(after.Stakers))
	}
	if after.hasValidator(nodes[8].Address) {
		t.Fatalf("non-NCP staker node8 (%s) entered the validator set %v", nodes[8].Address, after.validatorAddrs())
	}
}

// --- useNCP test helpers ---

// bootUseNCP launches a standalone wbft network from an 8-node preset with a
// genesis overlay enabling useNCP (targetValidators=3, stabilizingStakersThreshold=2,
// epochLength=10) and an NCP set of nodes 4,5,6 (the operators), distinct from
// the three mining validators (nodes 1,2,3 — the stakers).
func bootUseNCP(t *testing.T, cli, bin, preset string, nodes map[int]presetRec) *network {
	t.Helper()
	ncps := strings.Join([]string{nodes[4].Address, nodes[5].Address, nodes[6].Address}, ",")
	overlay := filepath.Join(t.TempDir(), "usenetcp.json")
	ov := `{"genesis":{"config":{"croissant":{` +
		`"wBFT":{"useNCP":true,"targetValidators":3,"stabilizingStakersThreshold":2,"epochLength":10},` +
		`"govContracts":{"govNCP":{"params":{"ncps":"` + ncps + `"}}}}}}}`
	if err := os.WriteFile(overlay, []byte(ov), 0o644); err != nil {
		t.Fatalf("write overlay: %v", err)
	}
	n := launchPreset(t, cli, "wbft", bin, preset, 3, 0, []string{"--genesis-overlay", overlay})
	n.waitAdvancing(n.rpcURLFor(1), 60*time.Second)
	return n
}

// registerStakerVia registers node stakerIdx as a GovStaking staker, paid and
// operated by node opIdx, using the staker's preset BLS public key + PoP.
func registerStakerVia(t *testing.T, n *network, url string, nodes map[int]presetRec, stakerIdx, opIdx int, amount *big.Int) {
	t.Helper()
	st, op := nodes[stakerIdx], nodes[opIdx]
	w := n.wallet(mustDecodeHex(t, op.NodeKey), url)
	data := accounts.EncodeCallArgs(
		"registerStaker(uint256,address,address,uint256,bytes,bytes)",
		accounts.Uint(amount), accounts.Address(st.Address), accounts.Address(st.Address),
		accounts.Uint(big.NewInt(0)), accounts.Bytes(mustDecodeHex(t, st.BLSPublic)), accounts.Bytes(mustDecodeHex(t, st.BLSPoP)),
	)
	hash, err := w.Execute(context.Background(), govStakingAddr, mustDecodeHex(t, data), amount)
	if err != nil {
		t.Fatalf("registerStaker node%d via op node%d: %v", stakerIdx, opIdx, err)
	}
	n.waitReceiptSuccess(url, hash)
}

// epochInfo is the wbft EpochInfo subset the useNCP tests assert on.
type epochInfo struct {
	Stabilizing bool `json:"stabilizing"`
	Stakers     []struct {
		Addr string `json:"addr"`
	} `json:"stakers"`
	Validators []struct {
		Addr string `json:"addr"`
	} `json:"validators"`
}

func (e epochInfo) validatorAddrs() []string {
	out := make([]string, 0, len(e.Validators))
	for _, v := range e.Validators {
		out = append(out, v.Addr)
	}
	return out
}

func (e epochInfo) hasValidator(addr string) bool {
	for _, v := range e.Validators {
		if strings.EqualFold(v.Addr, addr) {
			return true
		}
	}
	return false
}

// tryEpochInfo reads the EpochInfo at the most recent completed epoch boundary,
// returning ok=false while no boundary block exists yet or the query transiently
// fails (the boundary block may not be produced at the instant of the call).
func tryEpochInfo(c *rpc.Client) (epochInfo, bool) {
	head, err := c.BlockNumber(context.Background())
	if err != nil || head < usenetcpEpochLn {
		return epochInfo{}, false
	}
	boundary := head / usenetcpEpochLn * usenetcpEpochLn
	var extra struct {
		EpochInfo epochInfo `json:"epochInfo"`
	}
	if err := c.Call(context.Background(), "istanbul_getWbftExtraInfo", &extra, hexBlock(int64(boundary))); err != nil {
		return epochInfo{}, false
	}
	return extra.EpochInfo, true
}

// latestEpochInfo returns the most recent completed epoch's EpochInfo, retrying
// briefly while the boundary block is still being produced.
func latestEpochInfo(t *testing.T, c *rpc.Client) epochInfo {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if ei, ok := tryEpochInfo(c); ok {
			return ei
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("no epoch-boundary EpochInfo available within 30s")
	return epochInfo{}
}

// waitStakerCount polls the latest epoch boundary's EpochInfo until the candidate
// (staker) count reaches want, returning that EpochInfo.
func waitStakerCount(t *testing.T, c *rpc.Client, want int, timeout time.Duration) epochInfo {
	t.Helper()
	deadline := time.Now().Add(timeout)
	last := -1
	for time.Now().Before(deadline) {
		if ei, ok := tryEpochInfo(c); ok {
			last = len(ei.Stakers)
			if last == want {
				return ei
			}
		}
		time.Sleep(3 * time.Second)
	}
	t.Fatalf("candidate (staker) count did not reach %d within %s (last=%d)", want, timeout, last)
	return epochInfo{}
}

func ncpUint(t *testing.T, c *rpc.Client, to, sig string) *big.Int {
	t.Helper()
	out, err := c.EthCall(context.Background(), to, accounts.EncodeCallArgs(sig))
	if err != nil {
		t.Fatalf("eth_call %s: %v", sig, err)
	}
	v, _ := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
	if v == nil {
		t.Fatalf("bad uint from %s: %q", sig, out)
	}
	return v
}

// --- generated-preset helpers ---

type presetRec struct {
	Index     int    `json:"index"`
	NodeKey   string `json:"nodekey"`
	Address   string `json:"address"`
	BLSPublic string `json:"blsPublicKey"`
	BLSPoP    string `json:"blsPoP"`
}

// genPresetNV generates an n-node preset with v validators via `chainbench keys
// generate`, returning the preset dir. Skips when BOOTNODE_BIN is unset.
func genPresetNV(t *testing.T, cli, binary string, n, v int) string {
	t.Helper()
	boot := os.Getenv("BOOTNODE_BIN")
	if boot == "" {
		t.Skip("set BOOTNODE_BIN=/path/to/go-wbft/build/bin/bootnode to generate the useNCP preset")
	}
	if _, err := os.Stat(boot); err != nil {
		t.Skipf("BOOTNODE_BIN=%s not found", boot)
	}
	dir, err := os.MkdirTemp("/tmp", "cbncp")
	if err != nil {
		t.Fatalf("mkdir preset dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	cmd := exec.Command(cli, "keys", "generate",
		"--nodes", itoa(n), "--validators", itoa(v),
		"--bootnode", boot, "--binary", binary, "--out", dir)
	cmd.Dir = repoRoot(t)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("keys generate: %v\n%s", err, b)
	}
	return dir
}

func readPresetDir(t *testing.T, dir string) map[int]presetRec {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("read preset metadata: %v", err)
	}
	var m struct {
		Nodes []presetRec `json:"nodes"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse preset metadata: %v", err)
	}
	out := map[int]presetRec{}
	for _, node := range m.Nodes {
		out[node.Index] = node
	}
	return out
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		t.Fatalf("hex decode %q: %v", s, err)
	}
	return b
}
