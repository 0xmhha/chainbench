//go:build e2e

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

type autoWithdrawalProof struct {
	Ledger                                  credentialExpiryProof
	Delay                                   uint64
	Created, AfterEarly                     withdrawalCredential
	Early, Success                          credentialWithdrawal
	EarlyContractBefore, EarlyContractAfter string
}

func verifyAutoWithdrawal(p autoWithdrawalProof) error {
	requested, e1 := strconv.ParseUint(p.Created.RequestTime, 10, 64)
	mature, e2 := strconv.ParseUint(p.Created.WithdrawableTime, 10, 64)
	if e1 != nil || e2 != nil || p.Delay == 0 || mature != requested+p.Delay || p.Created.Amount != p.Ledger.Amount || p.AfterEarly != p.Created || p.Early.Timestamp < requested || p.Early.Timestamp >= mature || len(p.Early.Credentials) != 1 || p.Early.Credentials[0] != p.Created || p.Early.CredentialIndex != fmt.Sprint(p.Ledger.N+1) || p.Early.WithdrawalIndex != fmt.Sprint(p.Ledger.N) {
		return fmt.Errorf("auto mode early window/slot/pointers differ")
	}
	tx, r := p.Early.Transaction, p.Early.Receipt
	input := accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(new(big.Int)))
	if p.Early.Hash == "" || tx["hash"] != p.Early.Hash || r["transactionHash"] != p.Early.Hash || tx["blockHash"] == nil || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] || r["status"] != "0x1" || tx["from"] != p.Ledger.Caller || tx["to"] != p.Ledger.Contract || tx["input"] != input || tx["value"] != "0x0" {
		return fmt.Errorf("auto early input/receipt differs")
	}
	var logs []accounts.Log
	if err := json.Unmarshal([]byte(p.Early.LogsJSON), &logs); err != nil || len(logs) != 0 {
		return fmt.Errorf("auto early call emitted events")
	}
	parse := func(v string) (*big.Int, bool) {
		if !strings.HasPrefix(v, "0x") {
			return nil, false
		}
		return new(big.Int).SetString(v[2:], 16)
	}
	principal, valid := new(big.Int).SetString(p.Ledger.Amount, 10)
	beforeContract, bok := parse(p.EarlyContractBefore)
	afterContract, aok := parse(p.EarlyContractAfter)
	if !valid || principal.Sign() <= 0 || !bok || !aok || beforeContract.Cmp(principal) != 0 || afterContract.Cmp(principal) != 0 {
		return fmt.Errorf("pinned early contract balance differs")
	}
	used, ok := parse(fmt.Sprint(r["gasUsed"]))
	if !ok || used.Sign() <= 0 {
		return fmt.Errorf("missing early gas")
	}
	price, ok := parse(fmt.Sprint(r["effectiveGasPrice"]))
	if !ok || price.Sign() <= 0 {
		return fmt.Errorf("missing early price")
	}
	fee := new(big.Int).Mul(used, price)
	for address, want := range map[string]*big.Int{p.Ledger.Caller: new(big.Int).Neg(fee), p.Ledger.Contract: new(big.Int)} {
		if address == p.Ledger.Contract {
			_, hasPre := p.Early.Diff.Pre[address]
			_, hasPost := p.Early.Diff.Post[address]
			if !hasPre && !hasPost {
				continue
			}
		}
		before, ok := parse(p.Early.Diff.Pre[address].Balance)
		if !ok {
			return fmt.Errorf("early pre balance missing")
		}
		after := before
		if a, exists := p.Early.Diff.Post[address]; exists {
			after, ok = parse(a.Balance)
			if !ok {
				return fmt.Errorf("early post balance missing")
			}
		}
		if new(big.Int).Sub(after, before).Cmp(want) != 0 {
			return fmt.Errorf("auto early payout or incorrect fee")
		}
	}
	if p.Early.Diff.Post[p.Ledger.Caller].Nonce != p.Early.Diff.Pre[p.Ledger.Caller].Nonce+1 {
		return fmt.Errorf("early nonce differs")
	}
	if p.Success.Timestamp < mature || p.Success.Index != p.Ledger.N || len(p.Success.Credentials) != 1 || p.Success.Credentials[0] != (withdrawalCredential{"0", "0", "0"}) || p.Success.CredentialIndex != fmt.Sprint(p.Ledger.N+1) || p.Success.WithdrawalIndex != fmt.Sprint(p.Ledger.N+1) {
		return fmt.Errorf("auto mature slot not drained")
	}
	return verifyWithdrawalLedgerCount(p.Ledger, p.Success, 0)
}

func autoWithdrawalFixture() autoWithdrawalProof {
	p := unstakeWithdrawalFixture()
	input := accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(new(big.Int)))
	early := p.Success
	early.Hash = "early"
	early.Timestamp = 104
	early.Transaction = map[string]any{"hash": "early", "blockHash": "before", "blockNumber": "0x4", "from": p.Ledger.Caller, "to": p.Ledger.Contract, "input": input, "value": "0x0"}
	early.Receipt = map[string]any{"transactionHash": "early", "blockHash": "before", "blockNumber": "0x4", "status": "0x1", "gasUsed": "0x1", "effectiveGasPrice": "0x2"}
	early.Credentials = []withdrawalCredential{p.Created}
	early.WithdrawalIndex = "0"
	early.LogsJSON = "[]"
	early.Diff = claimDiff{Pre: map[string]claimDiffAccount{p.Ledger.Caller: {Balance: "0x64", Nonce: 1}, p.Ledger.Contract: {Balance: "0x3e8", Nonce: 1}}, Post: map[string]claimDiffAccount{p.Ledger.Caller: {Balance: "0x62", Nonce: 2}, p.Ledger.Contract: {Balance: "0x3e8", Nonce: 1}}}
	p.Success.Transaction["input"] = input
	return autoWithdrawalProof{Ledger: p.Ledger, Delay: 15, Created: p.Created, AfterEarly: p.Created, Early: early, Success: p.Success, EarlyContractBefore: "0x64", EarlyContractAfter: "0x64"}
}

func TestAutoWithdrawalRejectsEarlyPayoutAndWrongInput(t *testing.T) {
	if err := verifyAutoWithdrawal(autoWithdrawalFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*autoWithdrawalProof)
	}{
		{"explicit count instead", func(p *autoWithdrawalProof) {
			p.Early.Transaction["input"] = accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1)))
		}},
		{"early reverted", func(p *autoWithdrawalProof) { p.Early.Receipt["status"] = "0x0" }},
		{"early already mature", func(p *autoWithdrawalProof) { p.Early.Timestamp = 115 }},
		{"configured delay differs", func(p *autoWithdrawalProof) { p.Delay = 14 }},
		{"early slot drained", func(p *autoWithdrawalProof) { p.AfterEarly = withdrawalCredential{"0", "0", "0"} }},
		{"early pointer advanced", func(p *autoWithdrawalProof) { p.Early.WithdrawalIndex = "1" }},
		{"early payout", func(p *autoWithdrawalProof) {
			p.Early.Diff.Post[p.Ledger.Caller] = claimDiffAccount{Balance: "0xc6", Nonce: 2}
		}},
		{"contract debit early", func(p *autoWithdrawalProof) {
			p.Early.Diff.Post[p.Ledger.Contract] = claimDiffAccount{Balance: "0x384", Nonce: 1}
		}},
		{"early event", func(p *autoWithdrawalProof) { p.Early.LogsJSON = p.Success.LogsJSON }},
		{"early wrong receipt", func(p *autoWithdrawalProof) { p.Early.Receipt["transactionHash"] = "other" }},
		{"mature wrong count", func(p *autoWithdrawalProof) {
			p.Success.Transaction["input"] = accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1)))
		}},
		{"mature still locked", func(p *autoWithdrawalProof) { p.Success.Timestamp = 114 }},
		{"mature slot retained", func(p *autoWithdrawalProof) { p.Success.Credentials[0] = p.Created }},
		{"mature wrong payout", func(p *autoWithdrawalProof) {
			p.Success.Diff.Post[p.Ledger.Caller] = claimDiffAccount{Balance: "0xc7", Nonce: 2}
		}},
		{"mature no event", func(p *autoWithdrawalProof) { p.Success.LogsJSON = "[]" }},
		{"missing pinned contract balance", func(p *autoWithdrawalProof) { p.EarlyContractBefore = "" }},
		{"pinned contract debit", func(p *autoWithdrawalProof) { p.EarlyContractAfter = "0x63" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := autoWithdrawalFixture()
			tc.change(&p)
			if verifyAutoWithdrawal(p) == nil {
				t.Fatal("receipt success hid incorrect auto withdrawal")
			}
		})
	}
}

// The original withdraw(0) input uses automatic scanning. This records the
// current contract outcome, including a successful no-payout early transaction;
// it does not redefine the original early-revert requirement.
func TestWemixGovernanceAutoWithdrawalE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	ctx := context.Background()
	ap, err := accounts.ForChain("wbft")
	if err != nil {
		t.Fatal(err)
	}
	key := mustHexBytes(t, scenarioOpKeys["op1"])
	address, err := ap.AddressForKey(key)
	if err != nil {
		t.Fatal(err)
	}
	url := runGovHandoffOverlay(t, from, to, map[string]any{"alloc": map[string]any{address: map[string]any{"balance": "0x33b2e3c9fd0803ce8000000"}}})
	c := rpc.Dial(url)
	op, err := ap.OpenWallet(ctx, key, url)
	if err != nil {
		t.Fatal(err)
	}
	assertGuardCallerOutsideValidators(t, c, address)
	minimum := govConfigUint(t, c, "minimumStaking()")
	amount := new(big.Int).Mul(minimum, big.NewInt(2))
	staker := presetNodeAddr(t, 1)
	pk, sig := presetNodeBLS(t, 1)
	stakingRegister(t, c, op, staker, pk, sig, amount)
	stakingSendOK(t, c, op, accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(amount)), nil)
	ci, wi := credentialPointers(t, c, address)
	n, err := strconv.ParseUint(wi, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	p := autoWithdrawalProof{Ledger: credentialExpiryProof{Caller: address, Contract: e2eGovStaking, Amount: amount.String(), N: n}, Created: credentialAt(t, c, address, n)}
	delay := govConfigUint(t, c, "unbondingPeriodStaker()")
	if !delay.IsUint64() {
		t.Fatal("invalid configured delay")
	}
	p.Delay = delay.Uint64()
	if ci != fmt.Sprint(n+1) {
		t.Fatal("expected one credential")
	}
	capture := func(hash string) credentialWithdrawal {
		w := credentialWithdrawal{Hash: hash, Index: n}
		if err := c.Call(ctx, "eth_getTransactionByHash", &w.Transaction, hash); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getTransactionReceipt", &w.Receipt, hash); err != nil {
			t.Fatal(err)
		}
		b, err := c.BlockByNumber(ctx, fmt.Sprint(w.Receipt["blockNumber"]))
		if err != nil {
			t.Fatal(err)
		}
		w.Timestamp = b.Timestamp
		w.Credentials = []withdrawalCredential{credentialAt(t, c, address, n)}
		w.CredentialIndex, w.WithdrawalIndex = credentialPointers(t, c, address)
		logs, err := json.Marshal(feeReceipt(t, c, hash))
		if err != nil {
			t.Fatal(err)
		}
		w.LogsJSON = string(logs)
		if err := c.Call(ctx, "debug_traceTransaction", &w.Diff, hash, map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}}); err != nil {
			t.Fatal(err)
		}
		return w
	}
	input := accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(new(big.Int)))
	p.Early = capture(stakingSendOK(t, c, op, input, nil))
	p.AfterEarly = credentialAt(t, c, address, n)
	earlyHeight, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(p.Early.Receipt["blockNumber"]), "0x"), 16, 64)
	if err != nil || earlyHeight == 0 {
		t.Fatal("invalid early block", err)
	}
	if err := c.Call(ctx, "eth_getBalance", &p.EarlyContractBefore, e2eGovStaking, fmt.Sprintf("0x%x", earlyHeight-1)); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(ctx, "eth_getBalance", &p.EarlyContractAfter, e2eGovStaking, fmt.Sprintf("0x%x", earlyHeight)); err != nil {
		t.Fatal(err)
	}

	mature, err := strconv.ParseUint(p.Created.WithdrawableTime, 10, 64)
	if err != nil || p.Early.Timestamp >= mature {
		t.Fatal("auto early call not before maturity", err)
	}
	waitCredentialTime(t, c, mature)
	p.Success = capture(stakingSendOK(t, c, op, input, nil))
	logGovernanceProof(t, "GOV-004-auto-withdrawal", p)
	if err := verifyAutoWithdrawal(p); err != nil {
		t.Fatal(err)
	}
}
