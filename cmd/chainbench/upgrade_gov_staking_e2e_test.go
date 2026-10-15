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
// The BLS pubkey/PoP for each preset node ship in presets/keys/metadata.json
// (blsPublicKey/blsPoP), derived from the committed nodekeys. This test uses
// node2 as the operator and node1 as the staker. Run it with:
//
//	CHAINBENCH_E2E_FROM_BIN=/path/go-wemix/build/bin/gwemix \
//	CHAINBENCH_E2E_TO_BIN=/path/go-wbft/build/bin/gwemix \
//	go test -tags e2e -run TestWemixGovernanceRegisterStakerE2E -timeout 8m ./cmd/chainbench
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixGovernanceRegisterStakerE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	url := runGovHandoff(t, fromBin, toBin)
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

// TestWemixGovernanceUnstakeMinimumGuardE2E verifies the original two-minimum
// stake and minimum-plus-one unstake using an actual mined revert and exact reason.
func TestWemixGovernanceUnstakeMinimumGuardE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	overlay, operatorKey := governanceGuardOverlay(t, false)
	url := runGovHandoffOverlay(t, fromBin, toBin, overlay)
	c := rpc.Dial(url)
	ctx := context.Background()

	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	operator, err := ap.OpenWallet(ctx, operatorKey, url)
	if err != nil {
		t.Fatalf("open operator wallet: %v", err)
	}
	staker := presetNodeAddr(t, 1)
	blsPK, blsSig := presetNodeBLS(t, 1)
	minimum := govConfigUint(t, c, "minimumStaking()")
	amount := new(big.Int).Mul(minimum, big.NewInt(2))
	stakingRegister(t, c, operator, staker, blsPK, blsSig, amount)

	if staked := stakingUint(t, c, "getStakerAmount(address)", staker); staked.Cmp(amount) != 0 {
		t.Fatalf("getStakerAmount = %s after register, want %s", staked, amount)
	}

	assertGuardCallerOutsideValidators(t, c, operator.Address())
	before := readGovernanceGuardState(t, c, staker)
	input := accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(new(big.Int).Add(minimum, big.NewInt(1))))
	reason := "amount must equal balance to deactivate staker"
	failure := stakingRejectedTx(t, c, operator, input, big.NewInt(0), reason)
	cases := []guardCase{{Name: "unstake", Input: input, Sender: operator.Address(), Subject: staker, Value: "0", Reason: reason, Before: before, After: readGovernanceGuardState(t, c, staker), Failure: failure}}
	logGovernanceProof(t, "GOV-015-guards", map[string]any{"minimum": minimum.String(), "cases": cases})
	if err := verifyGovernanceGuards(cases, minimum, false); err != nil {
		t.Fatal(err)
	}
}

// TestWemixGovernanceClaimGuardE2E verifies both original unauthorized callers
// against two registered stakers with accrued rewards, using actual mined failures
// and transaction-level state diffs to separate block reward income from claims.
func TestWemixGovernanceClaimGuardE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set FROM/TO for actual claim guards")
	}
	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"op1", "op2", "opX"}
	keys := [][]byte{}
	alloc := map[string]any{}
	for _, name := range names {
		key := mustHexBytes(t, scenarioOpKeys[name])
		address, err := provider.AddressForKey(key)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		alloc[address] = map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}
	}
	url := runGovHandoffOverlay(t, from, to, map[string]any{"alloc": alloc})
	c := rpc.Dial(url)
	wallets := []accounts.Wallet{}
	for _, key := range keys {
		w, err := provider.OpenWallet(t.Context(), key, url)
		if err != nil {
			t.Fatal(err)
		}
		wallets = append(wallets, w)
		assertGuardCallerOutsideValidators(t, c, w.Address())
	}
	stakers := []string{presetNodeAddr(t, 1), presetNodeAddr(t, 2)}
	rewardees := []string{}
	minimum := govConfigUint(t, c, "minimumStaking()")
	for i, staker := range stakers {
		pk, sig := presetNodeBLS(t, i+1)
		stakingRegister(t, c, wallets[i], staker, pk, sig, minimum)
		if !strings.EqualFold(stakingStakerByOperator(t, c, wallets[i].Address()), staker) {
			t.Fatal("operator mapping absent")
		}
		rewardees = append(rewardees, wordToAddr(stakingInfoWord(t, c, staker, 1)))
	}
	deadline := time.Now().Add(90 * time.Second)
	balances := []string{}
	for {
		balances = nil
		ready := true
		for _, address := range rewardees {
			n, err := c.BalanceAt(t.Context(), address)
			if err != nil {
				t.Fatal(err)
			}
			balances = append(balances, n.String())
			ready = ready && n.Sign() > 0
		}
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("both registered stakers did not accrue reward", balances)
		}
		time.Sleep(200 * time.Millisecond)
	}
	proofs := []claimGuardProof{}
	for _, attempt := range []struct {
		name            string
		caller, subject int
	}{{"operatorA-to-B", 0, 1}, {"third-party-to-A", 2, 0}} {
		caller := wallets[attempt.caller]
		staker := stakers[attempt.subject]
		input := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0}))
		failure := stakingRejectedTx(t, c, caller, input, big.NewInt(0), "no reward to claim")
		p := claimGuardProof{Name: attempt.name, Caller: caller.Address(), Staker: staker, Rewardees: rewardees, RewardBalances: balances, Protected: []string{rewardees[0], rewardees[1], wallets[1].Address(), e2eGovStaking}, Failure: failure}
		if err = c.Call(t.Context(), "debug_traceTransaction", &p.Diff, failure.Hash, map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}}); err != nil {
			t.Fatal("claim guard state diff unavailable", err)
		}
		logGovernanceProof(t, "GOV-022-claim-diff", p)
		if err := verifyClaimGuardDiff(p); err != nil {
			t.Fatal(err)
		}
		proofs = append(proofs, p)
	}
	if len(proofs) != 2 {
		t.Fatal("original caller matrix incomplete")
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

	if stakingUint(t, c, "getDelegatedAmount(address)", staker).Sign() != 0 {
		t.Fatal("GOV-020 fixture unexpectedly has delegators")
	}
	logs := feeReceipt(t, c, hash)
	if !exactFeeEvent(logs, staker, "ChangingFeeRateRequested(address,uint256,uint256)", big.NewInt(0), big.NewInt(newRate)) {
		t.Fatal("GOV-020 missing or incorrect immediate Requested event")
	}
	if _, found := accounts.FindLog(logs, accounts.EventTopic("ChangingFeeRateExecuted(address,uint256)")); found {
		t.Fatal("GOV-020 v1 immediate request emitted unexpected Executed event")
	}
	pending, err := c.EthCall(ctx, e2eGovStaking, accounts.EncodeCallArgs("changingFeeRequests(address)", accounts.Address(staker)))
	if err != nil || pending != "0x"+strings.Repeat("0", 128) {
		t.Fatal("GOV-020 immediate fee left a delayed request", pending, err)
	}
	var transaction map[string]any
	if err := c.Call(ctx, "eth_getTransactionByHash", &transaction, hash); err != nil {
		t.Fatal(err)
	}
	if transaction["hash"] != hash || transaction["input"] != data || transaction["value"] != "0x0" || !strings.EqualFold(fmt.Sprint(transaction["from"]), operator.Address()) || !strings.EqualFold(fmt.Sprint(transaction["to"]), e2eGovStaking) {
		t.Fatal("GOV-020 actual request input differs", transaction)
	}
	logGovernanceProof(t, "GOV-020-immediate", map[string]any{"hash": hash, "staker": staker, "oldRate": 0, "newRate": newRate, "delegatedAmount": "0", "pending": pending, "transaction": transaction})
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
func stakingRegister(t *testing.T, c *rpc.Client, operator accounts.Wallet, staker string, blsPK, blsSig []byte, amount *big.Int) string {
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
	return hash
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
// presets/keys/metadata.json (blsPublicKey / blsPoP).
func presetNodeBLS(t *testing.T, idx int) (pk, pop []byte) {
	t.Helper()
	b, err := os.ReadFile("../../presets/keys/metadata.json")
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

// TestWemixGovernanceReactivateE2E covers GOV-019 reactivation on a decided
// permissioned candidate set. Two supporting stakers keep producing while the
// subject is inactive; epochs must exclude it and later include it again.
// The approved below-minimum criterion requires a mined v1 rejection with its
// exact reason and unchanged inactive state, distinct from eth_call refusal.
func TestWemixGovernanceReactivateE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	ctx := context.Background()
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	operators := make([]string, 3)
	alloc := map[string]any{}
	for i, name := range []string{"op1", "op2", "op3"} {
		operators[i], err = ap.AddressForKey(mustHexBytes(t, scenarioOpKeys[name]))
		if err != nil {
			t.Fatal(err)
		}
		alloc[operators[i]] = map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}
	}
	overlay := map[string]any{
		"alloc": alloc,
		"config": map[string]any{"croissant": map[string]any{
			"wBFT":         map[string]any{"useNCP": true, "targetValidators": 4, "stabilizingStakersThreshold": 2},
			"govContracts": map[string]any{"govNCP": map[string]any{"params": map[string]any{"ncps": strings.Join(operators, ",")}}},
		}},
	}
	url := runGovHandoffOverlay(t, fromBin, toBin, overlay)
	c := rpc.Dial(url)
	amount := govConfigUint(t, c, "minimumStaking()")
	var operator accounts.Wallet
	// Register the supporting validators first, then the subject (node1).
	for _, i := range []int{2, 3, 1} {
		w, err := ap.OpenWallet(ctx, mustHexBytes(t, scenarioOpKeys["op"+strconv.Itoa(i)]), url)
		if err != nil {
			t.Fatal(err)
		}
		pk, sig := presetNodeBLS(t, i)
		stakingRegister(t, c, w, presetNodeAddr(t, i), pk, sig, amount)
		if i == 1 {
			operator = w
		}
	}
	staker := presetNodeAddr(t, 1)
	boundary := assertReactivationSet(t, c, staker, true)
	observations := []validatorObservation{observeValidatorTransition(t, c, url, staker, boundary)}

	stakingSendOK(t, c, operator, accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(amount)), nil)
	if stakingIsStaker(t, c, staker) || stakingUint(t, c, "getStakerAmount(address)", staker).Sign() != 0 {
		t.Fatal("GOV-019 full unstake did not deactivate and clear the subject stake")
	}
	boundary = assertReactivationSet(t, c, staker, false)
	observations = append(observations, observeValidatorTransition(t, c, url, staker, boundary))

	// Preserve the typed call refusal and separately prove the actual mined failure.
	below := new(big.Int).Sub(amount, big.NewInt(1))
	var out string
	err = c.Call(ctx, "eth_call", &out, map[string]string{
		"from": operator.Address(), "to": e2eGovStaking,
		"data":  accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(below)),
		"value": "0x" + below.Text(16),
	}, "latest")
	var refusal *rpc.ServerError
	if !errors.As(err, &refusal) || refusal.Code != 3 || refusal.Message != "execution reverted: amount is less than minimum staking" {
		t.Fatalf("GOV-019 below-minimum call did not return the specific v1 rejection: %v", err)
	}
	if stakingIsStaker(t, c, staker) || stakingUint(t, c, "getStakerAmount(address)", staker).Sign() != 0 {
		t.Fatal("GOV-019 rejected below-minimum call changed the inactive state")
	}
	listBefore, err := c.EthCall(ctx, e2eGovStaking, accounts.EncodeCallArgs("stakers()"))
	if err != nil {
		t.Fatal(err)
	}
	rejected := stakingRejectedTx(t, c, operator, accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(below)), below, "amount is less than minimum staking")
	listAfter, err := c.EthCall(ctx, e2eGovStaking, accounts.EncodeCallArgs("stakers()"))
	if err != nil {
		t.Fatal(err)
	}
	if listAfter != listBefore || stakingIsStaker(t, c, staker) || stakingUint(t, c, "getStakerAmount(address)", staker).Sign() != 0 {
		t.Fatal("GOV-019 mined below-minimum failure changed inactive state or list")
	}
	logGovernanceProof(t, "GOV-019-state", map[string]any{"hash": rejected.Hash, "staker": staker, "minimum": amount.String(), "inputAmount": below.String(), "activeAfter": false, "amountAfter": "0", "listBefore": listBefore, "listAfter": listAfter, "callRefusal": refusal})

	hash := stakingSendOK(t, c, operator, accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(amount)), amount)
	if !stakingIsStaker(t, c, staker) || stakingUint(t, c, "getStakerAmount(address)", staker).Cmp(amount) != 0 {
		t.Fatal("GOV-019 minimum stake did not restore active state and exact amount")
	}
	receipt, err := c.TxReceipt(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status          string         `json:"status"`
		TransactionHash string         `json:"transactionHash"`
		Logs            []accounts.Log `json:"logs"`
	}
	if err := json.Unmarshal(receipt, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "0x1" || !strings.EqualFold(result.TransactionHash, hash) {
		t.Fatalf("GOV-019 wrong or failed receipt: %s", receipt)
	}
	event, found := accounts.FindLog(result.Logs, accounts.EventTopic("Staked(address,uint256)"))
	if !found || !strings.EqualFold(event.Address, e2eGovStaking) || len(event.Topics) != 2 || !strings.EqualFold(accounts.TopicToAddress(event.Topics[1]), staker) {
		t.Fatalf("GOV-019 wrong or missing Staked event: %+v", event)
	}
	value, ok := accounts.WordToBig(event.Data)
	if !ok || len(event.Data) != 66 || value.Cmp(amount) != 0 {
		t.Fatalf("GOV-019 event amount differs: %q", event.Data)
	}
	t.Logf("GOV-019 reactivation receipt: %s", receipt)
	boundary = assertReactivationSet(t, c, staker, true)
	observations = append(observations, observeValidatorTransition(t, c, url, staker, boundary))
	logGovernanceProof(t, "RPC-023-transition", map[string]any{"subject": staker, "epochLength": scenarioEpochLen, "observations": observations})
	if err := verifyValidatorTransition(observations, staker, scenarioEpochLen); err != nil {
		t.Fatal(err)
	}
}

// assertReactivationSet checks the contract's exact active list and the next
// completed epoch after the latest state change. A historical matching epoch
// or the stabilizing stage's inherited validators cannot satisfy this check.
func assertReactivationSet(t *testing.T, c *rpc.Client, subject string, active bool) uint64 {
	t.Helper()
	want := map[string]bool{}
	for _, i := range []int{1, 2, 3} {
		addr := strings.ToLower(presetNodeAddr(t, i))
		if active || !strings.EqualFold(addr, subject) {
			want[addr] = true
		}
	}
	out, err := c.EthCall(context.Background(), e2eGovStaking, accounts.EncodeCallArgs("stakers()"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := hex.DecodeString(strings.TrimPrefix(out, "0x"))
	// A sole address[] result has offset 32, its length, and padded addresses.
	if err != nil || len(data) != 64+32*len(want) || new(big.Int).SetBytes(data[:32]).Cmp(big.NewInt(32)) != 0 || new(big.Int).SetBytes(data[32:64]).Cmp(big.NewInt(int64(len(want)))) != 0 {
		t.Fatalf("GOV-019 malformed or wrong-size stakers() result: %s", out)
	}
	seen := map[string]bool{}
	for i := 64; i < len(data); i += 32 {
		addr := "0x" + hex.EncodeToString(data[i+12:i+32])
		if new(big.Int).SetBytes(data[i:i+12]).Sign() != 0 || !want[addr] || seen[addr] {
			t.Fatalf("GOV-019 unexpected or duplicate contract staker: %s", addr)
		}
		seen[addr] = true
	}
	head, err := c.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	boundary := (head/scenarioEpochLen + 1) * scenarioEpochLen
	waitForHead(t, c, boundary, 90*time.Second)
	epoch := epochInfoAt(t, c, boundary)
	if epoch.Stabilizing || len(epoch.Stakers) != len(want) || len(epoch.Validators) != len(want) {
		t.Fatalf("GOV-019 epoch %d has wrong or inherited candidate set: %+v", boundary, epoch)
	}
	seen = map[string]bool{}
	for _, raw := range epoch.Stakers {
		var candidate struct {
			Addr string `json:"addr"`
		}
		if err := json.Unmarshal(raw, &candidate); err != nil {
			t.Fatal(err)
		}
		addr := strings.ToLower(candidate.Addr)
		if !want[addr] || seen[addr] {
			t.Fatalf("GOV-019 epoch %d unexpected or duplicate candidate: %s", boundary, addr)
		}
		seen[addr] = true
	}
	seen = map[string]bool{}
	for _, validator := range epoch.Validators {
		addr := strings.ToLower(validator.Addr)
		if !want[addr] || seen[addr] {
			t.Fatalf("GOV-019 epoch %d unexpected or duplicate validator: %s", boundary, addr)
		}
		seen[addr] = true
	}
	encoded, err := json.Marshal(epoch)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("GOV-019 active=%v contract stakers=%s subsequent epoch=%d info=%s", active, out, boundary, encoded)
	return boundary
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
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	overlay, operatorKey := governanceGuardOverlay(t, true)
	url := runGovHandoffOverlay(t, fromBin, toBin, overlay)
	c := rpc.Dial(url)
	ctx := context.Background()

	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	operator, err := ap.OpenWallet(ctx, operatorKey, url)
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

	// All original guarded operations must fail in a mined transaction, rather
	// than accepting an SDK preparation/estimation/HTTP error as the guard.
	guardCases := []guardCase{}
	for _, op := range []struct {
		name, input string
		value       *big.Int
	}{
		{"stake", accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(amount)), amount},
		{"unstake", accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(big.NewInt(1))), big.NewInt(0)},
		{"delegate", accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(staker), accounts.Uint(amount)), amount},
	} {
		assertGuardCallerOutsideValidators(t, c, operator.Address())
		before := readGovernanceGuardState(t, c, staker)
		reason := "operation not permitted by council"
		failure := stakingRejectedTx(t, c, operator, op.input, op.value, reason)
		guardCases = append(guardCases, guardCase{Name: op.name, Input: op.input, Sender: operator.Address(), Subject: staker, Value: op.value.String(), Reason: reason, Before: before, After: readGovernanceGuardState(t, c, staker), Failure: failure})
	}
	logGovernanceProof(t, "GOV-017-guards", map[string]any{"minimum": amount.String(), "cases": guardCases, "emergencyMode": ncpBool(t, c, "emergencyMode()")})
	if err := verifyGovernanceGuards(guardCases, amount, true); err != nil {
		t.Fatal(err)
	}

	// Leave emergency mode; the flag clears.
	passNCPBallot(t, c, ncp, accounts.EncodeCallArgs("newProposalEmergencyMode(bool)", accounts.Word([]byte{0})))
	if ncpBool(t, c, "emergencyMode()") {
		t.Fatal("emergencyMode still true after deactivation")
	}
	// GOV-018 requires the guarded operation to resume, beyond flag clearing.
	hash := stakingSendOK(t, c, operator, accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(amount)), amount)
	want := new(big.Int).Mul(amount, big.NewInt(2))
	if got := stakingUint(t, c, "getStakerAmount(address)", staker); got.Cmp(want) != 0 {
		t.Fatalf("GOV-018 staking did not resume after deactivation: amount=%s, want %s", got, want)
	}
	receipt, err := c.TxReceipt(ctx, hash)
	if err != nil {
		t.Fatal("GOV-018 resumed staking receipt", err)
	}
	var result struct {
		Status          string         `json:"status"`
		TransactionHash string         `json:"transactionHash"`
		Logs            []accounts.Log `json:"logs"`
	}
	if err := json.Unmarshal(receipt, &result); err != nil {
		t.Fatal("GOV-018 receipt decoding", err)
	}
	if result.Status != "0x1" || !strings.EqualFold(result.TransactionHash, hash) {
		t.Fatalf("GOV-018 wrong or failed resumed transaction receipt: %s", receipt)
	}
	event, found := accounts.FindLog(result.Logs, accounts.EventTopic("Staked(address,uint256)"))
	if !found || !strings.EqualFold(event.Address, e2eGovStaking) || len(event.Topics) != 2 || !strings.EqualFold(accounts.TopicToAddress(event.Topics[1]), staker) {
		t.Fatalf("GOV-018 Staked event is missing or belongs to another contract/staker: %+v", event)
	}
	value, ok := accounts.WordToBig(event.Data)
	if !ok || len(event.Data) != 66 || value.Cmp(amount) != 0 {
		t.Fatalf("GOV-018 Staked event amount differs: %q, want %s", event.Data, amount)
	}
	t.Logf("GOV-018 resumed staking receipt: %s", receipt)
}

// stakingSendOK sends calldata to GovStaking from w (with optional value) and
// fails unless it mines with status 0x1.
func stakingSendOK(t *testing.T, c *rpc.Client, w accounts.Wallet, data string, value *big.Int) string {
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
	return hash
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

// TestWemixGovernanceBlockRewardE2E binds a sole registered validator rewardee
// to exact fixed-height balances and fifteen actual per-height reward RPC values.
func TestWemixGovernanceBlockRewardE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set FROM/TO for actual block reward ledger")
	}
	url := runGovHandoffOverlay(t, from, to, map[string]any{"config": map[string]any{"brioche": map[string]any{"blockReward": json.Number("1000000000000000000")}}}, rewardNodeLaunchScopes())
	c := rpc.Dial(url)
	var modules map[string]string
	var nodeInfo map[string]any
	if err := c.Call(t.Context(), "rpc_modules", &modules); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(t.Context(), "admin_nodeInfo", &nodeInfo); err != nil {
		t.Fatal(err)
	}
	logGovernanceProof(t, "GOV-012-api-context", map[string]any{"rpcModules": modules, "nodeInfo": nodeInfo})

	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	operator, err := provider.OpenWallet(t.Context(), presetNodeKey(t, 3), url)
	if err != nil {
		t.Fatal(err)
	}
	staker := presetNodeAddr(t, 2)
	pk, sig := presetNodeBLS(t, 2)
	minimum := govConfigUint(t, c, "minimumStaking()")
	hash := stakingRegister(t, c, operator, staker, pk, sig, minimum)
	p := rewardLedgerProof{Staker: staker, Operator: operator.Address(), Rewardee: wordToAddr(stakingInfoWord(t, c, staker, 1)), Minimum: minimum.String(), BLSKey: "0x" + hex.EncodeToString(pk), Logs: feeReceipt(t, c, hash)}
	if p.Start, err = c.BlockNumber(t.Context()); err != nil {
		t.Fatal(err)
	}
	p.End = p.Start + 15
	var beforeHex, afterHex string
	if err = c.Call(t.Context(), "eth_getBalance", &beforeHex, p.Rewardee, hexUint(p.Start)); err != nil {
		t.Fatal(err)
	}
	waitForHead(t, c, p.End, 90*time.Second)
	if err = c.Call(t.Context(), "eth_getBalance", &afterHex, p.Rewardee, hexUint(p.End)); err != nil {
		t.Fatal(err)
	}
	before, ok := new(big.Int).SetString(strings.TrimPrefix(beforeHex, "0x"), 16)
	if !ok {
		t.Fatal("balance missing")
	}
	after, ok := new(big.Int).SetString(strings.TrimPrefix(afterHex, "0x"), 16)
	if !ok {
		t.Fatal("balance missing")
	}
	p.Before = before.String()
	p.After = after.String()
	for n := p.Start + 1; n <= p.End; n++ {
		tag := hexUint(n)
		var h struct {
			Hash, ParentHash string
			Transactions     []string
		}
		if err = c.Call(t.Context(), "eth_getBlockByNumber", &h, tag, false); err != nil {
			t.Fatal(err)
		}
		row := rewardLedgerBlock{Number: n, Hash: h.Hash, ParentHash: h.ParentHash, Transactions: h.Transactions}
		if err = c.Call(t.Context(), "wemix_getBriocheBlockReward", &row.Reward, tag); err != nil {
			t.Fatal("actual reward RPC missing", err)
		}
		if err = c.Call(t.Context(), "istanbul_getValidators", &row.Validators, tag); err != nil {
			t.Fatal(err)
		}
		p.Blocks = append(p.Blocks, row)
	}
	if err = c.Call(t.Context(), "eth_call", &p.ContractStakers, map[string]string{"to": e2eGovStaking, "data": accounts.EncodeCallArgs("stakers()")}, hexUint(p.End)); err != nil {
		t.Fatal(err)
	}
	for user, dest := range map[string]*string{p.Operator: &p.OriginalOperatorPreview, p.Staker: &p.StakerPreview} {
		data := accounts.EncodeCallArgs("previewReward(address,address)", accounts.Address(p.Staker), accounts.Address(user))
		if err = c.Call(t.Context(), "eth_call", dest, map[string]string{"to": e2eGovStaking, "data": data}, hexUint(p.End)); err != nil {
			t.Fatal(err)
		}
	}
	logGovernanceProof(t, "GOV-012-reward-ledger", p)
	if err := verifyBlockRewardLedger(p); err != nil {
		t.Fatal(err)
	}
}

// TestWemixGovernanceOperatorClaimE2E records the approved GOV-013 difference:
// previewReward does not map the operator to the staker, while claim does.
// The staker-share parent preview is bound to exact payout events and ledgers.
func TestWemixGovernanceOperatorClaimE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set FROM/TO for actual operator claim ledger")
	}
	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	key := mustHexBytes(t, scenarioOpKeys["op1"])
	addr, err := provider.AddressForKey(key)
	if err != nil {
		t.Fatal(err)
	}
	url := runGovHandoffOverlay(t, from, to, map[string]any{"alloc": map[string]any{addr: map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}}})
	c := rpc.Dial(url)
	operator, err := provider.OpenWallet(t.Context(), key, url)
	if err != nil {
		t.Fatal(err)
	}
	assertGuardCallerOutsideValidators(t, c, operator.Address())
	staker := presetNodeAddr(t, 2)
	minimum := govConfigUint(t, c, "minimumStaking()")
	pk, sig := presetNodeBLS(t, 2)
	stakingRegister(t, c, operator, staker, pk, sig, minimum)
	rewardee := wordToAddr(stakingInfoWord(t, c, staker, 1))
	start, err := c.BlockNumber(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	waitForHead(t, c, start+10, 90*time.Second)
	hash := stakingSendOK(t, c, operator, accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0})), nil)
	p := operatorClaimProof{Claim: claimLedgerProof{Hash: hash, Caller: operator.Address(), Staker: staker, Rewardee: rewardee, FeePrecision: govConfigUint(t, c, "feePrecision()").String()}}
	if err = c.Call(t.Context(), "eth_getTransactionByHash", &p.Claim.Transaction, hash); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(t.Context(), "eth_getTransactionReceipt", &p.Claim.Receipt, hash); err != nil {
		t.Fatal(err)
	}
	number, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(p.Claim.Receipt["blockNumber"]), "0x"), 16, 64)
	if err != nil || number == 0 {
		t.Fatal("missing mined block", err)
	}
	p.Claim.PreviewBlock = fmt.Sprintf("0x%x", number-1)
	query := func(input string) string {
		var out string
		if err := c.Call(t.Context(), "eth_call", &out, map[string]string{"from": p.Claim.Caller, "to": e2eGovStaking, "data": input}, p.Claim.PreviewBlock); err != nil {
			t.Fatal(err)
		}
		return out
	}
	p.Claim.Preview = query(accounts.EncodeCallArgs("previewReward(address,address)", accounts.Address(staker), accounts.Address(staker)))
	p.OperatorPreview = query(accounts.EncodeCallArgs("previewReward(address,address)", accounts.Address(staker), accounts.Address(p.Claim.Caller)))
	word := func(raw string, index int) *big.Int {
		if !strings.HasPrefix(raw, "0x") || len(raw) < 2+(index+1)*64 {
			t.Fatal("missing ABI word", index)
		}
		n, ok := new(big.Int).SetString(raw[2+index*64:2+(index+1)*64], 16)
		if !ok {
			t.Fatal("invalid ABI word", index)
		}
		return n
	}
	p.MappedStaker = wordToAddr(word(query(accounts.EncodeCallArgs("stakerByOperator(address)", accounts.Address(p.Claim.Caller))), 0))
	info := query(accounts.EncodeCallArgs("stakerInfo(address)", accounts.Address(staker)))
	p.Claim.FeeRecipient = wordToAddr(word(info, 2))
	p.Claim.FeeRate = word(info, 3).String()
	p.TotalStaked = word(info, 5).String()
	p.Staked = word(query(accounts.EncodeCallArgs("userRewardInfo(address,address)", accounts.Address(staker), accounts.Address(staker))), 0).String()
	p.Claim.Logs = feeReceipt(t, c, hash)
	if err = c.Call(t.Context(), "debug_traceTransaction", &p.Claim.Diff, hash, map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}}); err != nil {
		t.Fatal(err)
	}
	logGovernanceProof(t, "GOV-013-operator-ledger", p)
	if err := verifyOperatorClaimProof(p); err != nil {
		t.Fatal(err)
	}
}

// TestWemixGovernanceDelegatorClaimE2E binds the actual payout and fee to
// parent-state preview, exact events and transaction-level balance changes.
func TestWemixGovernanceDelegatorClaimE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set FROM/TO for actual delegator ledger")
	}
	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	keys := [][]byte{}
	alloc := map[string]any{}
	for _, name := range []string{"op1", "op2", "opX"} {
		key := mustHexBytes(t, scenarioOpKeys[name])
		addr, err := provider.AddressForKey(key)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		alloc[addr] = map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}
	}
	url := runGovHandoffOverlay(t, from, to, map[string]any{"alloc": alloc})
	c := rpc.Dial(url)
	wallets := []accounts.Wallet{}
	for _, key := range keys {
		w, err := provider.OpenWallet(t.Context(), key, url)
		if err != nil {
			t.Fatal(err)
		}
		wallets = append(wallets, w)
		assertGuardCallerOutsideValidators(t, c, w.Address())
	}
	operator, delegator := wallets[0], wallets[1]
	feeRecipient := wallets[2].Address()
	staker := presetNodeAddr(t, 2)
	minimum := govConfigUint(t, c, "minimumStaking()")
	rate := big.NewInt(250)
	precision := govConfigUint(t, c, "feePrecision()")
	pk, sig := presetNodeBLS(t, 2)
	input := accounts.EncodeCallArgs("registerStaker(uint256,address,address,uint256,bytes,bytes)", accounts.Uint(minimum), accounts.Address(staker), accounts.Address(feeRecipient), accounts.Uint(rate), accounts.Bytes(pk), accounts.Bytes(sig))
	stakingSendOK(t, c, operator, input, minimum)
	if stakingInfoWord(t, c, staker, 3).Cmp(rate) != 0 {
		t.Fatal("declared fee rate not applied")
	}
	rewardee := wordToAddr(stakingInfoWord(t, c, staker, 1))
	stakingSendOK(t, c, delegator, accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(staker), accounts.Uint(minimum)), minimum)
	start, err := c.BlockNumber(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	waitForHead(t, c, start+10, 90*time.Second)
	hash := stakingSendOK(t, c, delegator, accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0})), nil)
	p := claimLedgerProof{Hash: hash, Caller: delegator.Address(), Staker: staker, Rewardee: rewardee, FeeRecipient: feeRecipient, FeeRate: rate.String(), FeePrecision: precision.String()}
	if err = c.Call(t.Context(), "eth_getTransactionByHash", &p.Transaction, hash); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(t.Context(), "eth_getTransactionReceipt", &p.Receipt, hash); err != nil {
		t.Fatal(err)
	}
	number, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(p.Receipt["blockNumber"]), "0x"), 16, 64)
	if err != nil || number == 0 {
		t.Fatal("receipt block missing", err)
	}
	p.PreviewBlock = fmt.Sprintf("0x%x", number-1)
	previewInput := accounts.EncodeCallArgs("previewReward(address,address)", accounts.Address(staker), accounts.Address(p.Caller))
	if err = c.Call(t.Context(), "eth_call", &p.Preview, map[string]string{"from": p.Caller, "to": e2eGovStaking, "data": previewInput}, p.PreviewBlock); err != nil {
		t.Fatal("execution-parent preview unavailable", err)
	}
	p.Logs = feeReceipt(t, c, hash)
	if err = c.Call(t.Context(), "debug_traceTransaction", &p.Diff, hash, map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}}); err != nil {
		t.Fatal(err)
	}
	logGovernanceProof(t, "GOV-014-claim-ledger", p)
	if err := verifyClaimLedger(p); err != nil {
		t.Fatal(err)
	}
}

// TestWemixGovernanceFeeChangeDelayedE2E ports wemix4 GOV-021 (fee change,
// delayed path): when a staker HAS delegators, requestChangingFee does NOT apply
// the new rate immediately (unlike GOV-020's no-delegator case) — it records a
// pending request (changingFeeRequests[staker].requestTime != 0) and leaves the
// current feeRate unchanged until the chain timestamp reaches changeFeeDelay.
// It checks a mined early revert and exact request/execution events.
//
//	go test -tags e2e -run TestWemixGovernanceFeeChangeDelayedE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceFeeChangeDelayedE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	key := mustHexBytes(t, scenarioOpKeys["op2"])
	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	operatorAddress, err := provider.AddressForKey(key)
	if err != nil {
		t.Fatal(err)
	}
	overlay := map[string]any{"alloc": map[string]any{operatorAddress: map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}}, "config": map[string]any{"croissant": map[string]any{"govContracts": map[string]any{"govConfig": map[string]any{"params": map[string]any{"changeFeeDelay": "30"}}}}}}
	url := runGovHandoffOverlay(t, fromBin, toBin, overlay)
	c := rpc.Dial(url)
	ctx := context.Background()
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatalf("accounts.ForChain(wbft): %v", err)
	}
	operator, err := ap.OpenWallet(ctx, key, url)
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
	requestHash := stakingSendOK(t, c, operator, accounts.EncodeCallArgs("requestChangingFee(uint256)", accounts.Uint(big.NewInt(newRate))), nil)
	requestLogs := feeReceipt(t, c, requestHash)
	if !exactFeeEvent(requestLogs, staker, "ChangingFeeRateRequested(address,uint256,uint256)", big.NewInt(0), big.NewInt(newRate)) {
		t.Fatal("GOV-021 missing or incorrect request event")
	}
	if _, found := accounts.FindLog(requestLogs, accounts.EventTopic("ChangingFeeRateExecuted(address,uint256)")); found {
		t.Fatal("GOV-021 executed before delay")
	}

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

	delay := govConfigUint(t, c, "changeFeeDelay()")
	if delay.Cmp(big.NewInt(30)) != 0 {
		t.Fatal("GOV-021 explicit delay overlay not applied", delay)
	}
	boundary := new(big.Int).Add(reqTime, delay)
	block, err := c.BlockByNumber(ctx, "latest")
	if err != nil || new(big.Int).SetUint64(block.Timestamp).Cmp(boundary) >= 0 {
		t.Fatalf("early check missed chain time window: %v %+v", err, block)
	}
	executeInput := accounts.EncodeCallArgs("executeChangingFee(address)", accounts.Address(staker))
	var callResult string
	callErr := c.Call(ctx, "eth_call", &callResult, map[string]string{"from": operator.Address(), "to": e2eGovStaking, "data": executeInput}, "latest")
	var refusal *rpc.ServerError
	const reason = "the request cannot be executed before delay time"
	if !errors.As(callErr, &refusal) || refusal.Code != 3 || refusal.Message != "execution reverted: "+reason {
		t.Fatal("GOV-021 wrong early call refusal", callErr)
	}
	early := stakingRejectedTx(t, c, operator, executeInput, new(big.Int), reason)
	earlyBlock, err := c.BlockByNumber(ctx, fmt.Sprint(early.Receipt["blockNumber"]))
	if err != nil || new(big.Int).SetUint64(earlyBlock.Timestamp).Cmp(boundary) >= 0 {
		t.Fatal("GOV-021 refusal was not mined before delay", err)
	}
	pendingAfter, err := c.EthCall(ctx, e2eGovStaking, accounts.EncodeCallArgs("changingFeeRequests(address)", accounts.Address(staker)))
	if err != nil || pendingAfter != out || stakingInfoWord(t, c, staker, 3).Sign() != 0 {
		t.Fatal("GOV-021 early failure changed pending request or fee")
	}
	deadline := time.Now().Add(90 * time.Second)
	for {
		block, err = c.BlockByNumber(ctx, "latest")
		if err != nil {
			t.Fatal(err)
		}
		if new(big.Int).SetUint64(block.Timestamp).Cmp(boundary) >= 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("GOV-021 chain timestamp did not reach delay")
		}
		select {
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		case <-time.After(time.Second):
		}
	}
	executionHash := stakingSendOK(t, c, operator, executeInput, nil)
	executionLogs := feeReceipt(t, c, executionHash)
	if !exactFeeEvent(executionLogs, staker, "ChangingFeeRateExecuted(address,uint256)", big.NewInt(newRate)) {
		t.Fatal("GOV-021 missing or incorrect executed event")
	}
	if fee := stakingInfoWord(t, c, staker, 3); fee.Cmp(big.NewInt(newRate)) != 0 {
		t.Fatal("GOV-021 applied rate differs", fee)
	}
	pending, err := c.EthCall(ctx, e2eGovStaking, accounts.EncodeCallArgs("changingFeeRequests(address)", accounts.Address(staker)))
	if err != nil || pending != "0x"+strings.Repeat("0", 128) {
		t.Fatal("GOV-021 applied request not cleared", pending, err)
	}
	logGovernanceProof(t, "GOV-021-delay", map[string]any{"requestHash": requestHash, "executionHash": executionHash, "earlyHash": early.Hash, "staker": staker, "requestTime": reqTime.String(), "delay": delay.String(), "boundary": boundary.String(), "earlyTimestamp": earlyBlock.Timestamp, "readyTimestamp": block.Timestamp, "newRate": newRate, "pendingAfterEarlyRefusal": pendingAfter, "pendingAfterExecution": pending})

}

// TestWemixGovernanceCredentialExpiryE2E proves individual chain-time maturity,
// exact skipped/drained slots, event indices, pointers and withdrawal ledgers.
func TestWemixGovernanceCredentialExpiryE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set FROM/TO for actual credential expiry")
	}
	provider, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	keys := [][]byte{}
	alloc := map[string]any{}
	for _, name := range []string{"op1", "op2"} {
		key := mustHexBytes(t, scenarioOpKeys[name])
		address, err := provider.AddressForKey(key)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		alloc[address] = map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}
	}
	url := runGovHandoffOverlay(t, from, to, map[string]any{"alloc": alloc})
	c := rpc.Dial(url)
	wallets := []accounts.Wallet{}
	for _, key := range keys {
		w, err := provider.OpenWallet(t.Context(), key, url)
		if err != nil {
			t.Fatal(err)
		}
		wallets = append(wallets, w)
		assertGuardCallerOutsideValidators(t, c, w.Address())
	}
	op := wallets[0]
	minimum := govConfigUint(t, c, "minimumStaking()")
	valA, valB := presetNodeAddr(t, 2), presetNodeAddr(t, 4)
	pkA, sigA := presetNodeBLS(t, 2)
	pkB, sigB := presetNodeBLS(t, 4)
	stakingRegister(t, c, op, valA, pkA, sigA, new(big.Int).Mul(minimum, big.NewInt(2)))
	stakingRegister(t, c, wallets[1], valB, pkB, sigB, minimum)
	// Delegation setup precedes creation of the long-lived credential.
	stakingSendOK(t, c, op, accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(valB), accounts.Uint(minimum)), minimum)
	p := credentialExpiryProof{Caller: op.Address(), Contract: e2eGovStaking, Amount: minimum.String(), StakerDelay: govConfigUint(t, c, "unbondingPeriodStaker()").String(), DelegatorDelay: govConfigUint(t, c, "unbondingPeriodDelegator()").String()}
	index, _ := credentialPointers(t, c, p.Caller)
	parsed, ok := new(big.Int).SetString(index, 10)
	if !ok || !parsed.IsUint64() {
		t.Fatal("initial credential pointer invalid")
	}
	p.N = parsed.Uint64()
	longHash := stakingSendOK(t, c, op, accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(minimum)), nil)
	if !stakingIsStaker(t, c, valA) || stakingUint(t, c, "getStakerAmount(address)", valA).Cmp(minimum) != 0 {
		t.Fatal("partial unstake changed active minimum")
	}
	shortHash := stakingSendOK(t, c, op, accounts.EncodeCallArgs("undelegate(address,uint256)", accounts.Address(valB), accounts.Uint(minimum)), nil)
	p.Created = []withdrawalCredential{credentialAt(t, c, p.Caller, p.N), credentialAt(t, c, p.Caller, p.N+1)}
	for i, hash := range []string{longHash, shortHash} {
		var receipt map[string]any
		if err = c.Call(t.Context(), "eth_getTransactionReceipt", &receipt, hash); err != nil {
			t.Fatal(err)
		}
		block, err := c.BlockByNumber(t.Context(), fmt.Sprint(receipt["blockNumber"]))
		if err != nil || fmt.Sprint(block.Timestamp) != p.Created[i].RequestTime {
			t.Fatal("credential request does not match actual creation block", err)
		}
	}
	withdraw := func(index uint64) credentialWithdrawal {
		hash := stakingSendOK(t, c, op, accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1))), nil)
		w := credentialWithdrawal{Hash: hash, Index: index}
		if err = c.Call(t.Context(), "eth_getTransactionByHash", &w.Transaction, hash); err != nil {
			t.Fatal(err)
		}
		if err = c.Call(t.Context(), "eth_getTransactionReceipt", &w.Receipt, hash); err != nil {
			t.Fatal(err)
		}
		block, err := c.BlockByNumber(t.Context(), fmt.Sprint(w.Receipt["blockNumber"]))
		if err != nil {
			t.Fatal(err)
		}
		w.Timestamp = block.Timestamp
		w.Credentials = []withdrawalCredential{credentialAt(t, c, p.Caller, p.N), credentialAt(t, c, p.Caller, p.N+1)}
		w.CredentialIndex, w.WithdrawalIndex = credentialPointers(t, c, p.Caller)
		logs := feeReceipt(t, c, hash)
		encoded, err := json.Marshal(logs)
		if err != nil {
			t.Fatal(err)
		}
		w.LogsJSON = string(encoded)
		if err = c.Call(t.Context(), "debug_traceTransaction", &w.Diff, hash, map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}}); err != nil {
			t.Fatal(err)
		}
		logGovernanceProof(t, "GOV-023-withdrawal", w)
		if err := verifyWithdrawalLedger(p, w); err != nil {
			t.Fatal(err)
		}
		return w
	}
	shortMature, _ := strconv.ParseUint(p.Created[1].WithdrawableTime, 10, 64)
	longMature, _ := strconv.ParseUint(p.Created[0].WithdrawableTime, 10, 64)
	waitCredentialTime(t, c, shortMature)
	latest, err := c.BlockByNumber(t.Context(), "latest")
	if err != nil || latest.Timestamp >= longMature {
		t.Fatal("missed distinct maturity window", err)
	}
	p.First = withdraw(p.N + 1)
	waitCredentialTime(t, c, longMature)
	p.Second = withdraw(p.N)
	logGovernanceProof(t, "GOV-023-expiry", p)
	if err := verifyCredentialExpiry(p); err != nil {
		t.Fatal(err)
	}
}

// stabilizingStakersThreshold is the wbft test genesis'
// croissant.wBFT.stabilizingStakersThreshold — the staker count at or above which
// the epoch leaves the stabilizing stage. Kept in sync with
// internal/chains/wbft/genesis.json, which is a literal there: unlike
// targetValidators and epochLength it is not derived from the validator set,
// because it is a threshold for leaving a stage rather than a size (worklist R7).
const stabilizingStakersThreshold = 5

// TestWemixGovernanceStabilizingE2E ports the reachable core of wemix4 GOV-010
// (stabilization stage): at an epoch boundary the wbft engine publishes an
// EpochInfo whose `stabilizing` flag is true while the active staker count is
// below stabilizingStakersThreshold. The handoff successor starts with the four
// croissant.init validators as its stakers (4 < threshold 5), so every epoch's
// EpochInfo.stabilizing must be true, and the observed staker count must indeed
// be below the configured threshold — tying the flag to the real count rather
// than asserting it in isolation.
//
// EpochInfo is only present on epoch-boundary blocks (multiples of epochLength,
// which is 10 here — this profile declares four validators and epochLength is
// derived as max(10, validators); the fork lands on block 20), so this queries a
// boundary block rather than "latest". The stabilizing->false transition is
// covered separately, by TestWemixGovernanceEpochDecidesTheValidatorSetE2E.
//
// An earlier version of this comment said that branch needed useNCP-driven
// selection and 7 governance NCPs. That was wrong: GovNCP.inspectOperation
// returns !emergencyMode, so registering a staker needs no ballot. What it does
// need is one operator account per staker — GovStaking refuses msg.sender ==
// _staker and refuses an address that already operates another staker — which an
// alloc overlay supplies.
//
//	go test -tags e2e -run TestWemixGovernanceStabilizingE2E -timeout 8m ./cmd/chainbench
func TestWemixGovernanceStabilizingE2E(t *testing.T) {
	fromBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN")
	toBin := os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("set CHAINBENCH_E2E_FROM_BIN and CHAINBENCH_E2E_TO_BIN to run")
	}
	url := runGovHandoff(t, fromBin, toBin)
	c := rpc.Dial(url)
	ctx := context.Background()

	const epochLength, forkBlock = 10, 20

	// Wait until at least one post-fork epoch boundary has been produced.
	deadline := time.Now().Add(90 * time.Second)
	var head uint64
	for {
		h, err := c.BlockNumber(ctx)
		if err == nil {
			head = h
		}
		if head > forkBlock {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("chain did not reach the first post-fork epoch boundary (head=%d)", head)
		}
		time.Sleep(2 * time.Second)
	}

	// Largest epoch boundary strictly below head (so the block is finalized).
	boundary := (head - 1) / epochLength * epochLength
	if boundary < forkBlock {
		boundary = forkBlock
	}

	var extra struct {
		EpochInfo struct {
			Stabilizing bool              `json:"stabilizing"`
			Stakers     []json.RawMessage `json:"stakers"`
		} `json:"epochInfo"`
	}
	if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &extra, hexUint(boundary)); err != nil {
		t.Fatalf("istanbul_getWbftExtraInfo(%d): %v", boundary, err)
	}

	stakerCount := len(extra.EpochInfo.Stakers)
	if stakerCount == 0 {
		t.Fatalf("EpochInfo at block %d has no stakers", boundary)
	}
	if stakerCount >= stabilizingStakersThreshold {
		t.Fatalf("staker count %d >= threshold %d — below-threshold branch not exercised", stakerCount, stabilizingStakersThreshold)
	}
	if !extra.EpochInfo.Stabilizing {
		t.Fatalf("EpochInfo.stabilizing = false at block %d with %d stakers (< threshold %d), want true",
			boundary, stakerCount, stabilizingStakersThreshold)
	}
	t.Logf("epoch boundary %d: stabilizing=true with %d stakers (< threshold %d)", boundary, stakerCount, stabilizingStakersThreshold)
}

// hexUint formats a block number as a 0x-prefixed hex quantity for JSON-RPC.
func hexUint(n uint64) string { return "0x" + strconv.FormatUint(n, 16) }

// balanceOf reads an account's coin balance at the latest block.
func balanceOf(t *testing.T, c *rpc.Client, addr string) *big.Int {
	t.Helper()
	bal, err := c.BalanceAt(context.Background(), addr)
	if err != nil {
		t.Fatalf("BalanceAt(%s): %v", addr, err)
	}
	return bal
}
