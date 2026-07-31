//go:build e2e

// This E2E ports the foundational wemix4 staking WRITE flow (GOV-003:
// GovStaking.registerStaker). On the go-wbft handoff successor the governance
// council (GovNCP) permits operations while not in emergency mode, so an operator
// can register a staker directly, subject to the contract's own guards:
//
//   - msg.value == amount, and minimumStaking <= amount <= maximumStaking
//   - the operator (msg.sender) must differ from the staker
//   - a valid BLS public key + proof-of-possession for the staker identity
//
// The BLS pubkey/PoP for each preset node ship in keys/preset/metadata.json
// (blsPublicKey/blsPoP), derived from the committed nodekeys. This test uses
// node2 as the operator and node1 as the staker. Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	CHAINBENCH_E2E_TEMPLATE=/path/go-wemix/wemix/scripts/genesis-template.json \
//	go test -tags e2e -run TestWemixGovernanceRegisterStakerE2E -timeout 8m ./cmd/chainbench
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

func TestWemixGovernanceRegisterStakerE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	url := runGovHandoff(t, fromBin, toBin, template)
	c := rpc.Dial(url)
	ctx := context.Background()

	// Operator (node2) sends the tx and funds the stake; staker (node1) is the
	// registered validator identity — the two must differ.
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)

	// Sanity pre-state.
	if stakingIsStaker(t, c, staker) {
		t.Fatalf("staker %s already registered before the test", staker)
	}

	// amount = minimumStaking (well within [min, max]).
	amount := govConfigUint(t, c, "minimumStaking()")
	if amount.Sign() <= 0 {
		t.Fatalf("minimumStaking() = %s, want > 0", amount)
	}

	stakingRegister(t, c, operator, staker, blsPK, blsSig, amount)

	// Post-state: the staker is registered and mapped from its operator.
	if !stakingIsStaker(t, c, staker) {
		t.Fatalf("isStaker(%s) is false after registration", staker)
	}
	if got := stakingStakerByOperator(t, c, operator.Address()); !strings.EqualFold(got, staker) {
		t.Fatalf("stakerByOperator(%s) = %s, want %s", operator.Address(), got, staker)
	}
}

// TestWemixGovernanceDelegateE2E ports wemix4 GOV-011 (delegation) on top of the
// registered staker: a delegator (node3) delegates value to the active staker
// (node1), and GovStaking.getDelegatedAmount(staker) grows by the delegated
// amount. `delegate` is payable and requires the staker to be active.
//
//	go test -tags e2e -run TestWemixGovernanceDelegateE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceDelegateE2E(t *testing.T) {
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
	// Register staker node1 via operator node2 (precondition), then delegate from
	// node3.
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	stakingRegister(t, c, operator, staker, blsPK, blsSig, govConfigUint(t, c, "minimumStaking()"))

	delegator, err := ap.OpenWallet(ctx, presetNodeKey(t, 3), url)
	if err != nil {
		t.Fatalf("open delegator wallet: %v", err)
	}
	before := stakingUint(t, c, "getDelegatedAmount(address)", staker)

	// delegate(staker, amount) payable, value == amount. 1e24 wei.
	amount, _ := new(big.Int).SetString("1000000000000000000000000", 10)
	data := accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(staker), accounts.Uint(amount))
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, err := delegator.Execute(ctx, e2eGovStaking, raw, amount)
	if err != nil {
		t.Fatalf("delegate execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, hash); st != "0x1" {
		t.Fatalf("delegate reverted (status %s)", st)
	}

	after := stakingUint(t, c, "getDelegatedAmount(address)", staker)
	if got := new(big.Int).Sub(after, before); got.Cmp(amount) != 0 {
		t.Fatalf("getDelegatedAmount grew by %s, want %s (before=%s after=%s)", got, amount, before, after)
	}
}

// TestWemixGovernanceUnstakeE2E ports wemix4 GOV-004 (unstake) on top of the
// registered staker: the operator unstakes its full stake, which drops the
// staker's amount to zero and deactivates it (a partial unstake below
// minimumStaking is rejected, so a full unstake is the deactivation path). The
// withdrawal credential then matures over the unbonding period, which this test
// does not wait out.
//
//	go test -tags e2e -run TestWemixGovernanceUnstakeE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceUnstakeE2E(t *testing.T) {
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
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	amount := govConfigUint(t, c, "minimumStaking()")
	stakingRegister(t, c, operator, staker, blsPK, blsSig, amount)

	// The registered stake is the staker's own amount.
	if staked := stakingUint(t, c, "getStakerAmount(address)", staker); staked.Cmp(amount) != 0 {
		t.Fatalf("getStakerAmount = %s after register, want %s", staked, amount)
	}
	if !stakingIsStaker(t, c, staker) {
		t.Fatal("staker not registered before unstake")
	}

	// Full unstake by the operator: staker's amount -> 0.
	data := accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(amount))
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, err := operator.Execute(ctx, e2eGovStaking, raw, nil)
	if err != nil {
		t.Fatalf("unstake execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, hash); st != "0x1" {
		t.Fatalf("unstake reverted (status %s)", st)
	}
	if staked := stakingUint(t, c, "getStakerAmount(address)", staker); staked.Sign() != 0 {
		t.Fatalf("getStakerAmount = %s after full unstake, want 0", staked)
	}
}

// TestWemixGovernanceUnstakeMinimumGuardE2E ports wemix4 GOV-015 (unstake below
// minimum is rejected): a staker registered at exactly minimumStaking cannot be
// partially unstaked, because unstake requires the remaining balance to be either
// >= minimumStaking or exactly 0 ("amount must equal balance to deactivate
// staker"). Unstaking 1 wei would leave minimumStaking-1, which is neither, so it
// reverts and the stake is unchanged.
//
//	go test -tags e2e -run TestWemixGovernanceUnstakeMinimumGuardE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceUnstakeMinimumGuardE2E(t *testing.T) {
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
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	amount := govConfigUint(t, c, "minimumStaking()")
	stakingRegister(t, c, operator, staker, blsPK, blsSig, amount)

	if staked := stakingUint(t, c, "getStakerAmount(address)", staker); staked.Cmp(amount) != 0 {
		t.Fatalf("getStakerAmount = %s after register, want %s", staked, amount)
	}

	// Unstaking 1 wei leaves minimumStaking-1 (neither >= minimum nor 0) -> revert.
	data := accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(big.NewInt(1)))
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, execErr := operator.Execute(ctx, e2eGovStaking, raw, nil)
	if execErr == nil {
		if st := stakingWaitStatus(t, c, hash); st == "0x1" {
			t.Fatal("partial unstake below minimum succeeded (status 0x1) — guard missing")
		}
	}
	// The reverted unstake left the stake untouched.
	if staked := stakingUint(t, c, "getStakerAmount(address)", staker); staked.Cmp(amount) != 0 {
		t.Fatalf("getStakerAmount = %s after rejected partial unstake, want %s (unchanged)", staked, amount)
	}
}

// TestWemixGovernanceClaimGuardE2E ports wemix4 GOV-022 (claim theft guard): an
// account that is neither the staker's operator nor a delegator to it cannot
// claim that staker's rewards. GovStaking.claim resolves the caller to _user =
// (operator match ? staker : msg.sender) and requires that user to have a stake
// or pending reward, so a third party's claim reverts with "no reward to claim"
// — the staker's rewardee balance is never touched. This needs no accrued
// rewards: the guard reverts before any transfer.
//
//	go test -tags e2e -run TestWemixGovernanceClaimGuardE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceClaimGuardE2E(t *testing.T) {
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
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	stakingRegister(t, c, operator, staker, blsPK, blsSig, govConfigUint(t, c, "minimumStaking()"))

	// The staker's rewardee (word 1 of the struct) holds any rewards; its balance
	// must not change on a thief's claim.
	rewardee := wordToAddr(stakingInfoWord(t, c, staker, 1))
	balBefore, err := c.BalanceAt(ctx, rewardee)
	if err != nil {
		t.Fatalf("balance rewardee: %v", err)
	}

	// A third party (node3) is neither node1's operator nor a delegator to it.
	thief, err := ap.OpenWallet(ctx, presetNodeKey(t, 3), url)
	if err != nil {
		t.Fatalf("open thief wallet: %v", err)
	}
	data := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0}))
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, execErr := thief.Execute(ctx, e2eGovStaking, raw, nil)
	if execErr == nil {
		// The tx was admitted; it must revert (status 0x0), not succeed.
		if st := stakingWaitStatus(t, c, hash); st == "0x1" {
			t.Fatal("thief's claim succeeded (status 0x1) — reward theft not prevented")
		}
	}
	// execErr != nil (reverted at estimate) or status 0x0 both mean the guard held.

	balAfter, err := c.BalanceAt(ctx, rewardee)
	if err != nil {
		t.Fatalf("balance rewardee after: %v", err)
	}
	if balAfter.Cmp(balBefore) < 0 {
		t.Fatalf("rewardee balance decreased (%s -> %s) — reward stolen", balBefore, balAfter)
	}
}

// wordToAddr converts a 32-byte ABI word (right-aligned address) to a 0x address.
func wordToAddr(w *big.Int) string {
	b := w.Bytes()
	if len(b) > 20 {
		b = b[len(b)-20:]
	}
	return "0x" + hex.EncodeToString(leftPad(b, 20))
}

// leftPad left-pads b to n bytes.
func leftPad(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}

// TestWemixGovernanceFeeChangeE2E ports wemix4 GOV-020 (fee change, immediate
// path). When a staker has no delegators, requestChangingFee applies the new fee
// rate immediately (with delegators it becomes a delayed request). This registers
// a staker with feeRate 0, then the operator requests a new rate and the change
// takes effect at once.
//
//	go test -tags e2e -run TestWemixGovernanceFeeChangeE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceFeeChangeE2E(t *testing.T) {
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
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	stakingRegister(t, c, operator, staker, blsPK, blsSig, govConfigUint(t, c, "minimumStaking()"))

	// stakingRegister sets feeRate 0; confirm, then request a new rate.
	if fee := stakingInfoWord(t, c, staker, 3); fee.Sign() != 0 {
		t.Fatalf("initial feeRate = %s, want 0", fee)
	}
	const newRate = 250 // 2.5% (<= feePrecision 10000)
	data := accounts.EncodeCallArgs("requestChangingFee(uint256)", accounts.Uint(big.NewInt(newRate)))
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, err := operator.Execute(ctx, e2eGovStaking, raw, nil)
	if err != nil {
		t.Fatalf("requestChangingFee execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, hash); st != "0x1" {
		t.Fatalf("requestChangingFee reverted (status %s)", st)
	}

	// No delegators -> the fee change is immediate.
	if fee := stakingInfoWord(t, c, staker, 3); fee.Cmp(big.NewInt(newRate)) != 0 {
		t.Fatalf("feeRate = %s after immediate change, want %d", fee, newRate)
	}
}

// stakingInfoWord reads word `idx` of GovStaking.stakerInfo(staker). The Staker
// struct's leading fields are static (operator, rewardee, feeRecipient, feeRate,
// then an offset to the dynamic blsPubKey), so a static field sits at a fixed
// 32-byte word regardless of the dynamic tail — feeRate is word 3.
func stakingInfoWord(t *testing.T, c *rpc.Client, staker string, idx int) *big.Int {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovStaking, accounts.EncodeCallArgs("stakerInfo(address)", accounts.Address(staker)))
	if err != nil {
		t.Fatalf("eth_call stakerInfo: %v", err)
	}
	h := strings.TrimPrefix(strings.TrimSpace(out), "0x")
	if len(h) < (idx+1)*64 {
		t.Fatalf("stakerInfo result too short (%d hex chars) for word %d", len(h), idx)
	}
	v, ok := new(big.Int).SetString(h[idx*64:(idx+1)*64], 16)
	if !ok {
		t.Fatalf("stakerInfo word %d not hex", idx)
	}
	return v
}

// stakingRegister sends registerStaker(amount, staker, staker, 0, blsPK, blsSig)
// from operator (value=amount) and fails unless it mines successfully.
func stakingRegister(t *testing.T, c *rpc.Client, operator accounts.Wallet, staker string, blsPK, blsSig []byte, amount *big.Int) {
	t.Helper()
	data := accounts.EncodeCallArgs(
		"registerStaker(uint256,address,address,uint256,bytes,bytes)",
		accounts.Uint(amount),
		accounts.Address(staker),
		accounts.Address(staker), // fee recipient (non-zero); the staker itself
		accounts.Uint(big.NewInt(0)),
		accounts.Bytes(blsPK),
		accounts.Bytes(blsSig),
	)
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, err := operator.Execute(context.Background(), e2eGovStaking, raw, amount)
	if err != nil {
		t.Fatalf("registerStaker execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, hash); st != "0x1" {
		t.Fatalf("registerStaker reverted (status %s)", st)
	}
}

// stakingUint reads a GovStaking uint256 getter taking a single address arg.
func stakingUint(t *testing.T, c *rpc.Client, sig, addr string) *big.Int {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovStaking, accounts.EncodeCallArgs(sig, accounts.Address(addr)))
	if err != nil {
		t.Fatalf("eth_call GovStaking %s: %v", sig, err)
	}
	v, ok := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
	if !ok {
		t.Fatalf("GovStaking.%s not hex: %s", sig, out)
	}
	return v
}

// presetNodeBLS returns node idx's BLS public key and proof-of-possession from
// keys/preset/metadata.json (blsPublicKey / blsPoP).
func presetNodeBLS(t *testing.T, idx int) (pk, pop []byte) {
	t.Helper()
	b, err := os.ReadFile("../../keys/preset/metadata.json")
	if err != nil {
		t.Fatalf("read preset metadata: %v", err)
	}
	var m struct {
		Nodes []struct {
			Index        int    `json:"index"`
			BLSPublicKey string `json:"blsPublicKey"`
			BLSPoP       string `json:"blsPoP"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse preset metadata: %v", err)
	}
	for _, n := range m.Nodes {
		if n.Index == idx {
			if n.BLSPublicKey == "" || n.BLSPoP == "" {
				t.Fatalf("node %d missing blsPublicKey/blsPoP in preset metadata", idx)
			}
			pk, err = hex.DecodeString(strings.TrimPrefix(n.BLSPublicKey, "0x"))
			if err != nil {
				t.Fatalf("decode blsPublicKey: %v", err)
			}
			pop, err = hex.DecodeString(strings.TrimPrefix(n.BLSPoP, "0x"))
			if err != nil {
				t.Fatalf("decode blsPoP: %v", err)
			}
			return pk, pop
		}
	}
	t.Fatalf("no node %d in preset metadata", idx)
	return nil, nil
}

// govConfigUint reads a no-arg uint256 getter from GovConfig.
func govConfigUint(t *testing.T, c *rpc.Client, sig string) *big.Int {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovConfig, accounts.EncodeCallArgs(sig))
	if err != nil {
		t.Fatalf("eth_call GovConfig %s: %v", sig, err)
	}
	v, ok := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
	if !ok {
		t.Fatalf("GovConfig.%s not hex: %s", sig, out)
	}
	return v
}

// stakingIsStaker reads GovStaking.isStaker(addr).
func stakingIsStaker(t *testing.T, c *rpc.Client, addr string) bool {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovStaking, accounts.EncodeCallArgs("isStaker(address)", accounts.Address(addr)))
	if err != nil {
		t.Fatalf("eth_call isStaker: %v", err)
	}
	v, _ := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
	return v != nil && v.Sign() > 0
}

// stakingStakerByOperator reads GovStaking.stakerByOperator(op) as a 0x address.
func stakingStakerByOperator(t *testing.T, c *rpc.Client, op string) string {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovStaking, accounts.EncodeCallArgs("stakerByOperator(address)", accounts.Address(op)))
	if err != nil {
		t.Fatalf("eth_call stakerByOperator: %v", err)
	}
	h := strings.TrimPrefix(strings.TrimSpace(out), "0x")
	if len(h) < 40 {
		return "0x" + h
	}
	return "0x" + h[len(h)-40:]
}

// stakingWaitStatus polls for hash's receipt and returns its status field.
func stakingWaitStatus(t *testing.T, c *rpc.Client, hash string) string {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		rc, err := c.TxReceipt(context.Background(), hash)
		if err == nil && len(rc) > 0 && string(rc) != "null" {
			var r struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(rc, &r) == nil && r.Status != "" {
				return r.Status
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("tx %s never mined", hash)
	return ""
}

// TestWemixGovernanceReactivateE2E ports wemix4 GOV-016 (inactive -> active): a
// staker deactivated by a full unstake can be reactivated by staking again.
// isStaker tracks the active set, so it flips true -> false (full unstake) ->
// true (re-stake); the staker stays registered throughout, so stake() (which
// requires a registered staker) reactivates it.
//
//	go test -tags e2e -run TestWemixGovernanceReactivateE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceReactivateE2E(t *testing.T) {
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
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	amount := govConfigUint(t, c, "minimumStaking()")
	stakingRegister(t, c, operator, staker, blsPK, blsSig, amount)
	if !stakingIsStaker(t, c, staker) {
		t.Fatal("staker not active after register")
	}

	// Full unstake -> INACTIVE.
	stakingSendOK(t, c, operator, accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(amount)), nil)
	if stakingIsStaker(t, c, staker) {
		t.Fatal("staker still active after full unstake")
	}
	if v := stakingUint(t, c, "getStakerAmount(address)", staker); v.Sign() != 0 {
		t.Fatalf("getStakerAmount = %s after full unstake, want 0", v)
	}

	// Re-stake -> ACTIVE again (stake is payable, value == amount).
	stakingSendOK(t, c, operator, accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(amount)), amount)
	if !stakingIsStaker(t, c, staker) {
		t.Fatal("staker not reactivated after re-stake")
	}
	if v := stakingUint(t, c, "getStakerAmount(address)", staker); v.Cmp(amount) != 0 {
		t.Fatalf("getStakerAmount = %s after re-stake, want %s", v, amount)
	}
}

// TestWemixGovernanceEmergencyModeE2E ports wemix4 GOV-017 (emergency mode): the
// NCP council can enter emergency mode via a proposal+vote, and while it is on the
// GovStaking operations it guards are blocked (GovNCP.inspectOperation returns
// !emergencyMode, so the inspectWithCouncil modifier rejects them). Deactivating
// emergency mode restores them.
//
//	go test -tags e2e -run TestWemixGovernanceEmergencyModeE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceEmergencyModeE2E(t *testing.T) {
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
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	amount := govConfigUint(t, c, "minimumStaking()")
	stakingRegister(t, c, operator, staker, blsPK, blsSig, amount)

	// node1 is the sole NCP; it drives the emergency-mode ballots.
	ncp, err := ap.OpenWallet(ctx, presetNodeKey(t, 1), url)
	if err != nil {
		t.Fatalf("open NCP wallet: %v", err)
	}
	if ncpBool(t, c, "emergencyMode()") {
		t.Fatal("emergencyMode already true before the test")
	}

	// Enter emergency mode (propose + vote to quorum 1).
	passNCPBallot(t, c, ncp, accounts.EncodeCallArgs("newProposalEmergencyMode(bool)", accounts.Word([]byte{1})))
	if !ncpBool(t, c, "emergencyMode()") {
		t.Fatal("emergencyMode not true after the accepted proposal")
	}

	// A guarded staking op is now blocked by the council.
	raw, _ := hex.DecodeString(strings.TrimPrefix(accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(amount)), "0x"))
	if hash, execErr := operator.Execute(ctx, e2eGovStaking, raw, amount); execErr == nil {
		if st := stakingWaitStatus(t, c, hash); st == "0x1" {
			t.Fatal("stake() succeeded during emergency mode — council guard missing")
		}
	}

	// Leave emergency mode; the flag clears.
	passNCPBallot(t, c, ncp, accounts.EncodeCallArgs("newProposalEmergencyMode(bool)", accounts.Word([]byte{0})))
	if ncpBool(t, c, "emergencyMode()") {
		t.Fatal("emergencyMode still true after deactivation")
	}
}

// stakingSendOK sends calldata to GovStaking from w (with optional value) and
// fails unless it mines with status 0x1.
func stakingSendOK(t *testing.T, c *rpc.Client, w accounts.Wallet, data string, value *big.Int) {
	t.Helper()
	raw, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		t.Fatalf("decode calldata: %v", err)
	}
	hash, err := w.Execute(context.Background(), e2eGovStaking, raw, value)
	if err != nil {
		t.Fatalf("GovStaking execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, hash); st != "0x1" {
		t.Fatalf("GovStaking tx reverted (status %s)", st)
	}
}

// ncpBool reads a no-arg bool getter from GovNCP.
func ncpBool(t *testing.T, c *rpc.Client, sig string) bool {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovNCP, accounts.EncodeCallArgs(sig))
	if err != nil {
		t.Fatalf("eth_call GovNCP %s: %v", sig, err)
	}
	v, _ := new(big.Int).SetString(strings.TrimPrefix(strings.TrimSpace(out), "0x"), 16)
	return v != nil && v.Sign() > 0
}

// passNCPBallot submits a GovNCP proposal (its receipt log carries the ballot id)
// and votes it through as the sole NCP (quorum 1).
func passNCPBallot(t *testing.T, c *rpc.Client, ncp accounts.Wallet, proposalData string) {
	t.Helper()
	rc := ncpExecute(t, c, ncp, proposalData)
	if rc.Status != "0x1" {
		t.Fatalf("NCP proposal reverted (status %s)", rc.Status)
	}
	if len(rc.Logs) == 0 || len(rc.Logs[0].Topics) < 2 {
		t.Fatalf("no ballot id in proposal receipt: %+v", rc.Logs)
	}
	ballot, ok := new(big.Int).SetString(strings.TrimPrefix(rc.Logs[0].Topics[1], "0x"), 16)
	if !ok {
		t.Fatalf("ballot id not hex: %s", rc.Logs[0].Topics[1])
	}
	if vr := ncpExecute(t, c, ncp, accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(ballot), accounts.Word([]byte{1}))); vr.Status != "0x1" {
		t.Fatalf("NCP vote reverted (status %s)", vr.Status)
	}
}

// stakingPreviewPending reads GovStaking.previewReward(staker, user).pending
// (word 0 of the 4-word return) — user's accrued, unclaimed reward on staker
// (pass user == staker for the staker's own reward, or a delegator's address).
func stakingPreviewPending(t *testing.T, c *rpc.Client, staker, user string) *big.Int {
	t.Helper()
	out, err := c.EthCall(context.Background(), e2eGovStaking,
		accounts.EncodeCallArgs("previewReward(address,address)", accounts.Address(staker), accounts.Address(user)))
	if err != nil {
		t.Fatalf("eth_call previewReward: %v", err)
	}
	h := strings.TrimPrefix(strings.TrimSpace(out), "0x")
	if len(h) < 64 {
		t.Fatalf("previewReward result too short: %q", out)
	}
	v, _ := new(big.Int).SetString(h[:64], 16)
	return v
}

// registerProducingStaker registers a currently-producing validator (node2) as a
// staker via a distinct operator (node3), so block rewards accrue to it. Returns
// the staker address and the operator wallet.
func registerProducingStaker(t *testing.T, c *rpc.Client, ap accounts.AccountProvider, url string) (string, accounts.Wallet) {
	t.Helper()
	ctx := context.Background()
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 3), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 2) // a validator in the block-producing set
	blsPK, blsSig := presetNodeBLS(t, 2)
	stakingRegister(t, c, operator, staker, blsPK, blsSig, govConfigUint(t, c, "minimumStaking()"))
	return staker, operator
}

// TestWemixGovernanceBlockRewardE2E ports wemix4 GOV-012 (block reward accrual):
// once a producing validator is a registered staker, its unclaimed reward
// (previewReward) grows block over block as the engine distributes block rewards.
//
//	go test -tags e2e -run TestWemixGovernanceBlockRewardE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceBlockRewardE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	template := os.Getenv("CHAINBENCH_E2E_TEMPLATE")
	if fromBin == "" || toBin == "" || template == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN, CHAINBENCH_E2E_TO_BIN, CHAINBENCH_E2E_TEMPLATE to run")
	}
	url := runGovHandoff(t, fromBin, toBin, template)
	c := rpc.Dial(url)
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	staker, _ := registerProducingStaker(t, c, ap, url)

	before := stakingPreviewPending(t, c, staker, staker)
	start, _ := c.BlockNumber(context.Background())
	// Wait ~15 blocks for rewards to accrue.
	deadline := time.Now().Add(90 * time.Second)
	for {
		if h, _ := c.BlockNumber(context.Background()); h >= start+15 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("chain did not advance 15 blocks")
		}
		time.Sleep(2 * time.Second)
	}
	after := stakingPreviewPending(t, c, staker, staker)
	if after.Cmp(before) <= 0 {
		t.Fatalf("previewReward did not grow: before=%s after=%s", before, after)
	}
}

// TestWemixGovernanceOperatorClaimE2E ports wemix4 GOV-013 (operator claim): the
// operator claims the staker's accrued reward, which resets the pending amount
// (claim sets pendingReward = 0 before transferring).
//
//	go test -tags e2e -run TestWemixGovernanceOperatorClaimE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceOperatorClaimE2E(t *testing.T) {
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
	staker, operator := registerProducingStaker(t, c, ap, url)

	// Wait until a sizeable reward has accrued (rewards grow ~1e18/block, so this
	// clears the claim's gas cost and the read-timing noise).
	const threshold = "5000000000000000000" // 5e18
	want, _ := new(big.Int).SetString(threshold, 10)
	deadline := time.Now().Add(120 * time.Second)
	for stakingPreviewPending(t, c, staker, staker).Cmp(want) < 0 {
		if time.Now().After(deadline) {
			t.Fatal("reward did not accrue past the claim threshold")
		}
		time.Sleep(3 * time.Second)
	}

	// The operator (msg.sender) receives the reward on a no-restake claim, so its
	// balance rises by ~the reward minus gas — a clear net increase.
	opAddr := operator.Address()
	balBefore, err := c.BalanceAt(ctx, opAddr)
	if err != nil {
		t.Fatalf("operator balance: %v", err)
	}
	data := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0}))
	raw, _ := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	hash, err := operator.Execute(ctx, e2eGovStaking, raw, nil)
	if err != nil {
		t.Fatalf("claim execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, hash); st != "0x1" {
		t.Fatalf("claim reverted (status %s)", st)
	}
	balAfter, err := c.BalanceAt(ctx, opAddr)
	if err != nil {
		t.Fatalf("operator balance after: %v", err)
	}
	if balAfter.Cmp(balBefore) <= 0 {
		t.Fatalf("operator balance did not rise after claim: before=%s after=%s", balBefore, balAfter)
	}
}

// TestWemixGovernanceDelegatorClaimE2E ports wemix4 GOV-014 (delegator claim): a
// delegator to a producing staker accrues its share of block rewards and can
// claim them — claiming as a non-operator resolves _user to the caller, so the
// reward (minus fee) is sent to the delegator, raising its balance.
//
//	go test -tags e2e -run TestWemixGovernanceDelegatorClaimE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceDelegatorClaimE2E(t *testing.T) {
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
	staker, _ := registerProducingStaker(t, c, ap, url)

	// A delegator (node4) delegates to the producing staker.
	delegator, err := ap.OpenWallet(ctx, presetNodeKey(t, 4), url)
	if err != nil {
		t.Fatalf("open delegator wallet: %v", err)
	}
	delegAmount, _ := new(big.Int).SetString("1000000000000000000000000", 10) // 1e24
	dData := accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(staker), accounts.Uint(delegAmount))
	dRaw, _ := hex.DecodeString(strings.TrimPrefix(dData, "0x"))
	dHash, err := delegator.Execute(ctx, e2eGovStaking, dRaw, delegAmount)
	if err != nil {
		t.Fatalf("delegate execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, dHash); st != "0x1" {
		t.Fatalf("delegate reverted (status %s)", st)
	}

	// Wait until the delegator's own share of rewards has accrued.
	deleg := delegator.Address()
	want, _ := new(big.Int).SetString("2000000000000000000", 10) // 2e18
	deadline := time.Now().Add(120 * time.Second)
	for stakingPreviewPending(t, c, staker, deleg).Cmp(want) < 0 {
		if time.Now().After(deadline) {
			t.Fatal("delegator reward did not accrue past the claim threshold")
		}
		time.Sleep(3 * time.Second)
	}

	// The delegator claims its share; the reward (minus fee) lands in its balance.
	balBefore, err := c.BalanceAt(ctx, deleg)
	if err != nil {
		t.Fatalf("delegator balance: %v", err)
	}
	cData := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0}))
	cRaw, _ := hex.DecodeString(strings.TrimPrefix(cData, "0x"))
	cHash, err := delegator.Execute(ctx, e2eGovStaking, cRaw, nil)
	if err != nil {
		t.Fatalf("delegator claim execute: %v", err)
	}
	if st := stakingWaitStatus(t, c, cHash); st != "0x1" {
		t.Fatalf("delegator claim reverted (status %s)", st)
	}
	balAfter, err := c.BalanceAt(ctx, deleg)
	if err != nil {
		t.Fatalf("delegator balance after: %v", err)
	}
	if balAfter.Cmp(balBefore) <= 0 {
		t.Fatalf("delegator balance did not rise after claim: before=%s after=%s", balBefore, balAfter)
	}
}

// TestWemixGovernanceFeeChangeDelayedE2E ports wemix4 GOV-021 (fee change,
// delayed path): when a staker HAS delegators, requestChangingFee does NOT apply
// the new rate immediately (unlike GOV-020's no-delegator case) — it records a
// pending request (changingFeeRequests[staker].requestTime != 0) and leaves the
// current feeRate unchanged until executeChangingFee after changeFeeDelay (which
// this test, bounded to seconds, does not wait out).
//
//	go test -tags e2e -run TestWemixGovernanceFeeChangeDelayedE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceFeeChangeDelayedE2E(t *testing.T) {
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
	operator, err := ap.OpenWallet(ctx, presetNodeKey(t, 2), url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	stakingRegister(t, c, operator, staker, blsPK, blsSig, govConfigUint(t, c, "minimumStaking()"))

	// Give the staker a delegator (node3) so the fee change takes the delayed path.
	delegator, err := ap.OpenWallet(ctx, presetNodeKey(t, 3), url)
	if err != nil {
		t.Fatalf("open delegator wallet: %v", err)
	}
	delegAmount, _ := new(big.Int).SetString("1000000000000000000000000", 10)
	stakingSendOK(t, c, delegator, accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(staker), accounts.Uint(delegAmount)), delegAmount)

	// Request a new fee rate; with a delegator present it must NOT apply now.
	const newRate = 250
	stakingSendOK(t, c, operator, accounts.EncodeCallArgs("requestChangingFee(uint256)", accounts.Uint(big.NewInt(newRate))), nil)

	if fee := stakingInfoWord(t, c, staker, 3); fee.Sign() != 0 {
		t.Fatalf("feeRate changed immediately despite a delegator (got %s, want 0 until execute)", fee)
	}
	// changingFeeRequests(staker) returns (newFeeRate, requestTime); a non-zero
	// requestTime (word 1) proves the delayed request was recorded.
	out, err := c.EthCall(ctx, e2eGovStaking, accounts.EncodeCallArgs("changingFeeRequests(address)", accounts.Address(staker)))
	if err != nil {
		t.Fatalf("eth_call changingFeeRequests: %v", err)
	}
	h := strings.TrimPrefix(strings.TrimSpace(out), "0x")
	if len(h) < 128 {
		t.Fatalf("changingFeeRequests result too short: %q", out)
	}
	newFee, _ := new(big.Int).SetString(h[:64], 16)
	reqTime, _ := new(big.Int).SetString(h[64:128], 16)
	if reqTime.Sign() == 0 {
		t.Fatalf("no pending fee-change request recorded (requestTime 0)")
	}
	if newFee.Cmp(big.NewInt(newRate)) != 0 {
		t.Fatalf("pending newFeeRate = %s, want %d", newFee, newRate)
	}

	// After changeFeeDelay elapses (short in the test genesis), executeChangingFee
	// applies the pending rate. The delay is block.timestamp-based, so wait it out.
	delay := govConfigUint(t, c, "changeFeeDelay()")
	time.Sleep(time.Duration(delay.Int64()+3) * time.Second)
	stakingSendOK(t, c, operator, accounts.EncodeCallArgs("executeChangingFee(address)", accounts.Address(staker)), nil)
	if fee := stakingInfoWord(t, c, staker, 3); fee.Cmp(big.NewInt(newRate)) != 0 {
		t.Fatalf("feeRate = %s after executeChangingFee, want %d", fee, newRate)
	}
}
