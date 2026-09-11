//go:build e2e

// This E2E covers the branch every other wbft test leaves untouched: the epoch
// that DECIDES a validator set rather than copying the previous one.
//
// A fresh wbft chain starts its genesis epoch with Stabilizing = true, and while
// stabilizing the engine takes newEpoch.Validators = latestEpochInfo.Validators
// and never calls decideValidators. Leaving that stage needs
// croissant.wBFT.stabilizingStakersThreshold stakers registered in GovStaking,
// and in a handoff nobody stakes there — the producers staked in *wemix*
// governance. So the chain never reached the code that reads targetValidators,
// and a genesis that pinned targetValidators to 1 while declaring fifteen
// validators looked harmless for as long as nobody looked.
//
// TestWemixGovernanceStabilizingE2E covers the other side of the same flag (the
// count stays below the threshold, so the stage persists). This one pushes the
// count over it and checks what the engine then decides.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// epochOpKeys are the operators that register the stakers. They are separate
// accounts because GovStaking refuses msg.sender == _staker and refuses an
// address that already operates another staker, so N stakers need N distinct
// operators. Fixed dev keys, funded by the alloc overlay below — not preset key
// material, which must never be given a role on a shared network.
var epochOpKeys = []string{
	"a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
	"a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2",
	"a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3a3",
	"a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4",
}

// TestWemixGovernanceEpochDecidesTheValidatorSetE2E registers every declared
// validator as a staker, which ends the stabilization stage, and then reads the
// first epoch the engine decided for itself.
//
// The assertion is that the decided set is the set that staked — all four of
// them. Under the genesis this repo shipped until 2026-09-11, targetValidators
// was the literal 1 whatever the validator count was, and decideValidators cuts
// the stake-sorted candidates at it, so this same chain would have come out of
// the stabilization stage with ONE validator.
//
//	go test -tags e2e -run TestWemixGovernanceEpochDecidesTheValidatorSetE2E -timeout 10m ./cmd/chainbench
func TestWemixGovernanceEpochDecidesTheValidatorSetE2E(t *testing.T) {
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

	// The stakers are the profile's four declared validators, which are the nodes
	// actually running. Registering addresses that are NOT running nodes would
	// hand production to nobody and stall the chain at the boundary — the decided
	// set replaces the genesis set outright, it does not extend it.
	const validators = 4

	opAddrs := make([]string, validators)
	for i, k := range epochOpKeys {
		a, err := ap.AddressForKey(mustHexBytes(t, k))
		if err != nil {
			t.Fatalf("address for operator %d: %v", i+1, err)
		}
		opAddrs[i] = a
	}

	// threshold = the validator count, so the stage ends exactly when every
	// declared validator has staked and not before. The shipped default is 5,
	// which four validators can never reach.
	bal := "0x33b2e3c9fd0803ce8000000" // 1e27 wei: minimumStaking is 1e25
	alloc := make([]string, 0, len(opAddrs))
	for _, a := range opAddrs {
		alloc = append(alloc, fmt.Sprintf(`%q:{"balance":%q}`, a, bal))
	}
	overlayJSON := `{"genesis":{` +
		fmt.Sprintf(`"config":{"croissant":{"wBFT":{"stabilizingStakersThreshold":%d}}},`, validators) +
		`"alloc":{` + strings.Join(alloc, ",") + `}}}`
	overlay := filepath.Join(t.TempDir(), "epoch-threshold.json")
	if err := os.WriteFile(overlay, []byte(overlayJSON), 0o644); err != nil {
		t.Fatalf("write overlay: %v", err)
	}

	url := runGovHandoffArgs(t, fromBin, toBin, template, []string{"--genesis-overlay", overlay})
	c := rpc.Dial(url)

	// Pre-state: the stage is on and nothing has staked, which is what makes the
	// transition below attributable to the registrations.
	const epochLength, forkBlock = 10, 20
	if first := epochInfoAt(t, c, forkBlock); !first.Stabilizing {
		t.Fatalf("block %d: stabilizing is already false before any staker registered", forkBlock)
	}

	amount := govConfigUint(t, c, "minimumStaking()")
	if amount.Sign() <= 0 {
		t.Fatalf("minimumStaking() = %s, want > 0", amount)
	}
	staked := map[string]bool{}
	for i := 0; i < validators; i++ {
		operator, err := ap.OpenWallet(ctx, mustHexBytes(t, epochOpKeys[i]), url)
		if err != nil {
			t.Fatalf("open operator %d wallet: %v", i+1, err)
		}
		staker := presetNodeAddr(t, i+1)
		blsPK, blsSig := presetNodeBLS(t, i+1)
		stakingRegister(t, c, operator, staker, blsPK, blsSig, amount)
		staked[strings.ToLower(staker)] = true
	}
	if len(staked) != validators {
		t.Fatalf("registered %d distinct stakers, want %d", len(staked), validators)
	}

	// The first boundary strictly after the last registration is the first epoch
	// that could see them. Wait past it so the block is final.
	head, err := c.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("head after registration: %v", err)
	}
	boundary := (head/epochLength + 1) * epochLength
	waitForHead(t, c, boundary+1, 3*time.Minute)

	decided := epochInfoAt(t, c, boundary)
	if decided.Stabilizing {
		t.Fatalf("block %d: still stabilizing with %d stakers registered (threshold %d) — the engine never decided a set",
			boundary, validators, validators)
	}
	if got := len(decided.Validators); got != validators {
		t.Fatalf("block %d decided %d validator(s), want %d. One is what a targetValidators pinned to 1 produces, whatever the set declares",
			boundary, got, validators)
	}

	// The decided set has to be the set that staked, not merely the right size.
	for _, v := range decided.Validators {
		if !staked[strings.ToLower(v.Addr)] {
			t.Errorf("block %d: decided validator %s did not register as a staker", boundary, v.Addr)
		}
	}

	// And it has to produce. A set that is decided but cannot seal is a halted
	// chain, which the epoch info alone would not show.
	waitForHead(t, c, boundary+epochLength, 3*time.Minute)
	sealers := map[string]bool{}
	for n := boundary + 1; n <= boundary+epochLength; n++ {
		var blk struct {
			Miner string `json:"miner"`
		}
		if err := c.Call(ctx, "eth_getBlockByNumber", &blk, hexUint(n), false); err != nil {
			t.Fatalf("read block %d: %v", n, err)
		}
		m := strings.ToLower(blk.Miner)
		if !staked[m] {
			t.Fatalf("block %d after the decided epoch was sealed by %s, which is not in the decided set", n, m)
		}
		sealers[m] = true
	}
	if len(sealers) < 2 {
		t.Errorf("blocks %d-%d were sealed by %d validator(s); the decided set is not rotating",
			boundary+1, boundary+epochLength, len(sealers))
	}
	t.Logf("epoch %d decided %d validators from %d stakers; blocks %d-%d sealed by %d of them",
		boundary, len(decided.Validators), validators, boundary+1, boundary+epochLength, len(sealers))
}

// epochValidator is one entry of an EpochInfo's decided validator list.
type epochValidator struct {
	Index string `json:"index"`
	Addr  string `json:"addr"`
	BLS   string `json:"bls"`
}

// epochInfo is the part of istanbul_getWbftExtraInfo this test reads. EpochInfo
// is present only on epoch-boundary blocks.
type epochInfo struct {
	Stabilizing bool              `json:"stabilizing"`
	Stakers     []json.RawMessage `json:"stakers"`
	Validators  []epochValidator  `json:"validators"`
}

// epochInfoAt reads the EpochInfo published at an epoch-boundary block.
//
// The block is passed as a hex quantity, never as "latest": go-wbft reads the
// tag's negative sentinel straight into a block number and answers "block is not
// a wbft block", naming the wrong cause (worklist §1s B).
func epochInfoAt(t *testing.T, c *rpc.Client, block uint64) epochInfo {
	t.Helper()
	var extra struct {
		EpochInfo *epochInfo `json:"epochInfo"`
	}
	if err := c.Call(context.Background(), "istanbul_getWbftExtraInfo", &extra, hexUint(block)); err != nil {
		t.Fatalf("istanbul_getWbftExtraInfo(%d): %v", block, err)
	}
	if extra.EpochInfo == nil {
		t.Fatalf("block %d carries no EpochInfo, so it is not an epoch boundary", block)
	}
	return *extra.EpochInfo
}

// waitForHead blocks until the chain reaches want, or fails saying how far it got.
func waitForHead(t *testing.T, c *rpc.Client, want uint64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var head uint64
	for time.Now().Before(deadline) {
		if h, err := c.BlockNumber(context.Background()); err == nil {
			head = h
			if head >= want {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("head stalled at %d, never reached %d within %s", head, want, timeout)
}
