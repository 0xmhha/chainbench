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
