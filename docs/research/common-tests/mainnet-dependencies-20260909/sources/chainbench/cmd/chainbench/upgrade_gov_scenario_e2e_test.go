//go:build e2e

// This E2E ports wemix4's staking-based validator-selection flow — GOV-010
// (stabilization stage), GOV-005 (staker registration reflected in the validator
// set), and GOV-009 (a staker whose operator is not in the permissioned set is
// never selected) — on the real wemix->wbft UPGRADE (handoff) chain configured to
// full wemix4 fidelity via a genesis overlay.
//
// Model (go-wbft, useNCP=true — the current permissioned stage of staking-based
// selection): the govNCP set is a registry of OPERATOR addresses; the validator
// candidates are the stakers those operators registered
// (NCPStakers = { StakerByOperator[op] }). Once the candidate count reaches
// stabilizingStakersThreshold the epoch leaves the stabilizing stage and the top
// targetValidators candidates by stake become the validator set. "NCP" is wemix
// terminology; on wbft this is staking-based validator selection with the govNCP
// layer acting as the current permissioning gate ("약속된 validator") — the
// public/pure-staking mode (useNCP=false) is the future direction.
//
// The overlay makes the permissioned operators (funded ephemeral accounts, in
// govNCP.ncps) distinct from the mining validators/stakers (the four handoff
// successors), matching wemix4's OP_x vs VAL_x split, with targetValidators=4 and
// stabilizingStakersThreshold=2 so the flow is feasible on the 4-validator
// handoff.
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestWemixGovValidatorSelectionScenarioE2E -timeout 15m ./cmd/chainbench
package main

import (
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

// Deterministic operator private keys (TEST fixtures — funded only on the
// ephemeral local handoff via the genesis overlay). op1..op3 are the permissioned
// NCP operators; opX is a non-permissioned operator used for the GOV-009 case.
var scenarioOpKeys = map[string]string{
	"op1": "1111111111111111111111111111111111111111111111111111111111111111",
	"op2": "2222222222222222222222222222222222222222222222222222222222222222",
	"op3": "3333333333333333333333333333333333333333333333333333333333333333",
	"opX": "4444444444444444444444444444444444444444444444444444444444444444",
}

func TestWemixGovValidatorSelectionScenarioE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	ctx := context.Background()
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}

	// Derive operator addresses from the fixed keys.
	opAddr := map[string]string{}
	for name, keyHex := range scenarioOpKeys {
		a, err := ap.AddressForKey(mustHexBytes(t, keyHex))
		if err != nil {
			t.Fatalf("address for %s: %v", name, err)
		}
		opAddr[name] = a
	}

	// Full-fidelity overlay: useNCP on, permissioned operators op1..op3 in govNCP,
	// all four operators funded, targetValidators=4, threshold=2.
	bal := "0x33b2e3c9fd0803ce8000000" // 1e27 wei: covers 5x minimumStaking + gas
	ncps := strings.Join([]string{opAddr["op1"], opAddr["op2"], opAddr["op3"]}, ",")
	allocEntries := make([]string, 0, len(opAddr))
	for _, a := range opAddr {
		allocEntries = append(allocEntries, fmt.Sprintf(`%q:{"balance":%q}`, a, bal))
	}
	overlayJSON := `{"genesis":{` +
		`"config":{"croissant":{` +
		`"wBFT":{"useNCP":true,"targetValidators":4,"stabilizingStakersThreshold":2},` +
		`"govContracts":{"govNCP":{"params":{"ncps":"` + ncps + `"}}}}},` +
		`"alloc":{` + strings.Join(allocEntries, ",") + `}}}`
	overlay := filepath.Join(t.TempDir(), "wemix4-fidelity.json")
	if err := os.WriteFile(overlay, []byte(overlayJSON), 0o644); err != nil {
		t.Fatalf("write overlay: %v", err)
	}

	url := runGovHandoffArgs(t, fromBin, toBin, template, []string{"--genesis-overlay", overlay})
	c := rpc.Dial(url)

	// The four handoff successors (preset nodes 1..4) are the mining validators and
	// the stakers to be registered.
	valAddr := func(i int) string { return presetNodeAddr(t, i) }

	// GOV-010: with no registered stakers (candidates 0 < threshold 2) the epoch is
	// in the stabilizing stage.
	ei := scenarioWaitEpoch(t, c, func(e scenarioEpoch) bool { return true }, 60*time.Second)
	if !ei.Stabilizing {
		t.Fatalf("expected stabilizing=true before any staker registration, got stabilizing=false")
	}

	// GOV-005: register two stakers via permissioned operators op1,op2 -> candidate
	// count reaches threshold 2, the epoch leaves the stabilizing stage.
	openOp := func(name string) accounts.Wallet {
		w, err := ap.OpenWallet(ctx, mustHexBytes(t, scenarioOpKeys[name]), url)
		if err != nil {
			t.Fatalf("open operator %s: %v", name, err)
		}
		return w
	}
	registerVia := func(opName string, valIdx int, amount *big.Int) {
		pk, sig := presetNodeBLS(t, valIdx)
		stakingRegister(t, c, openOp(opName), valAddr(valIdx), pk, sig, amount)
	}
	minStake := govConfigUint(t, c, "minimumStaking()")

	registerVia("op1", 1, minStake)
	registerVia("op2", 2, minStake)
	ei = scenarioWaitEpoch(t, c, func(e scenarioEpoch) bool { return !e.Stabilizing }, 90*time.Second)
	if len(ei.Validators) != 2 {
		t.Fatalf("decided validator set = %d after 2 permissioned stakers, want 2", len(ei.Validators))
	}

	// A third staker via op3 grows the decided set to three.
	registerVia("op3", 3, minStake)
	ei = scenarioWaitEpoch(t, c, func(e scenarioEpoch) bool { return len(e.Stakers) == 3 }, 90*time.Second)
	if len(ei.Validators) != 3 {
		t.Fatalf("decided validator set = %d after 3rd permissioned staker, want 3 (did not grow)", len(ei.Validators))
	}
	if !ei.hasValidator(valAddr(3)) {
		t.Fatalf("newly registered staker node3 (%s) not in decided set %v", valAddr(3), ei.validatorAddrs())
	}

	// GOV-009: register a fourth staker via the NON-permissioned operator opX with a
	// far larger stake. Its operator is not in govNCP, so it is never a candidate.
	bigStake := new(big.Int).Mul(minStake, big.NewInt(5))
	registerVia("opX", 4, bigStake)
	time.Sleep((2*10 + 5) * time.Second) // two epochs
	after := scenarioLatestEpoch(t, c)
	if len(after.Stakers) != 3 {
		t.Fatalf("candidate set = %d after a non-permissioned staker, want 3 (it must not be a candidate)", len(after.Stakers))
	}
	if after.hasValidator(valAddr(4)) {
		t.Fatalf("non-permissioned staker node4 (%s) entered the validator set %v", valAddr(4), after.validatorAddrs())
	}
}

// scenarioEpoch is the wbft EpochInfo subset the scenario asserts on.
type scenarioEpoch struct {
	Stabilizing bool `json:"stabilizing"`
	Stakers     []struct {
		Addr string `json:"addr"`
	} `json:"stakers"`
	Validators []struct {
		Addr string `json:"addr"`
	} `json:"validators"`
}

func (e scenarioEpoch) validatorAddrs() []string {
	out := make([]string, 0, len(e.Validators))
	for _, v := range e.Validators {
		out = append(out, v.Addr)
	}
	return out
}

func (e scenarioEpoch) hasValidator(addr string) bool {
	for _, v := range e.Validators {
		if strings.EqualFold(v.Addr, addr) {
			return true
		}
	}
	return false
}

// scenarioForkBlock/EpochLen mirror the handoff profile + wbft genesis: the fork
// lands at block 20 and epochLength is 10, so EpochInfo exists at 20, 30, 40, ...
const scenarioForkBlock, scenarioEpochLen = 20, 10

// scenarioTryEpoch reads the EpochInfo at the latest completed post-fork epoch
// boundary, returning ok=false while none exists or the query transiently fails.
func scenarioTryEpoch(c *rpc.Client) (scenarioEpoch, bool) {
	head, err := c.BlockNumber(context.Background())
	if err != nil || head <= scenarioForkBlock {
		return scenarioEpoch{}, false
	}
	boundary := head / scenarioEpochLen * scenarioEpochLen
	if boundary < scenarioForkBlock {
		boundary = scenarioForkBlock
	}
	var extra struct {
		EpochInfo scenarioEpoch `json:"epochInfo"`
	}
	if err := c.Call(context.Background(), "istanbul_getWbftExtraInfo", &extra, hexUint(boundary)); err != nil {
		return scenarioEpoch{}, false
	}
	return extra.EpochInfo, true
}

func scenarioLatestEpoch(t *testing.T, c *rpc.Client) scenarioEpoch {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if e, ok := scenarioTryEpoch(c); ok {
			return e
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("no post-fork EpochInfo within 30s")
	return scenarioEpoch{}
}

// scenarioWaitEpoch polls the latest epoch boundary until pred holds.
func scenarioWaitEpoch(t *testing.T, c *rpc.Client, pred func(scenarioEpoch) bool, timeout time.Duration) scenarioEpoch {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last scenarioEpoch
	for time.Now().Before(deadline) {
		if e, ok := scenarioTryEpoch(c); ok {
			last = e
			if pred(e) {
				return e
			}
		}
		time.Sleep(3 * time.Second)
	}
	t.Fatalf("epoch predicate not satisfied within %s (last: stabilizing=%v stakers=%d validators=%d)",
		timeout, last.Stabilizing, len(last.Stakers), len(last.Validators))
	return scenarioEpoch{}
}

func mustHexBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	return b
}

var _ = json.Marshal
