//go:build e2e

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type delegationWithdrawalProof struct {
	Withdrawal                                      autoWithdrawalProof
	Staker, Operator, StakerAmount                  string
	DelegatorDelay, StakerDelay                     string
	DelegatedBefore, DelegatedAfter, DelegatedFinal string
	PowerBefore, PowerAfter, PowerFinal             string
	Delegate, Undelegate                            credentialWithdrawal
}

func verifyDelegationWithdrawal(p delegationWithdrawalProof) error {
	if p.DelegatorDelay != "30" || p.StakerDelay != "70" || p.Withdrawal.Delay != 30 {
		return fmt.Errorf("approved unbonding periods differ")
	}
	caller := p.Withdrawal.Ledger.Caller
	if caller == p.Staker || caller == p.Operator || p.Staker == p.Operator || len(p.Staker) != 42 || len(p.Operator) != 42 {
		return fmt.Errorf("delegator, staker and operator must be distinct")
	}
	amount, ok := new(big.Int).SetString(p.Withdrawal.Ledger.Amount, 10)
	stake, stakeOK := new(big.Int).SetString(p.StakerAmount, 10)
	if !ok || !stakeOK || amount.Sign() <= 0 || stake.Sign() <= 0 {
		return fmt.Errorf("stake/delegation missing")
	}
	if p.DelegatedBefore != "0" || p.DelegatedAfter != amount.String() || p.DelegatedFinal != "0" || p.PowerBefore != stake.String() || p.PowerAfter != new(big.Int).Add(stake, amount).String() || p.PowerFinal != stake.String() {
		return fmt.Errorf("delegated amount or staking power did not change exactly")
	}
	for i, w := range []credentialWithdrawal{p.Delegate, p.Undelegate} {
		signature := "delegate(address,uint256)"
		value := fmt.Sprintf("0x%x", amount)
		event := "Delegated(address,address,uint256)"
		topics := []string{delegationTopic(caller), delegationTopic(p.Staker)}
		contractDelta := new(big.Int).Set(amount)
		if i == 1 {
			signature = "undelegate(address,uint256)"
			value = "0x0"
			event = "Undelegated(address,address,address,uint256)"
			topics = append(topics, delegationTopic(caller))
			contractDelta.SetInt64(0)
		}
		input := accounts.EncodeCallArgs(signature, accounts.Address(p.Staker), accounts.Uint(amount))
		tx, r := w.Transaction, w.Receipt
		if w.Hash == "" || tx["hash"] != w.Hash || r["transactionHash"] != w.Hash || tx["blockHash"] == nil || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] || tx["from"] != caller || tx["to"] != p.Withdrawal.Ledger.Contract || tx["input"] != input || tx["value"] != value || r["status"] != "0x1" {
			return fmt.Errorf("wrong actual delegation input/receipt")
		}
		var logs []accounts.Log
		if err := json.Unmarshal([]byte(w.LogsJSON), &logs); err != nil {
			return err
		}
		count := 0
		for _, log := range logs {
			if len(log.Topics) == 0 || !strings.EqualFold(log.Topics[0], accounts.EventTopic(event)) {
				continue
			}
			count++
			if !strings.EqualFold(log.Address, p.Withdrawal.Ledger.Contract) || len(log.Topics) != len(topics)+1 || log.Data != "0x"+fmt.Sprintf("%064x", amount) {
				return fmt.Errorf("wrong delegation event")
			}
			for n, topic := range topics {
				if !strings.EqualFold(log.Topics[n+1], topic) {
					return fmt.Errorf("wrong delegation event party")
				}
			}
		}
		if count != 1 {
			return fmt.Errorf("missing/duplicate delegation event")
		}
		quantity := func(raw string) (*big.Int, bool) {
			if !strings.HasPrefix(raw, "0x") {
				return nil, false
			}
			return new(big.Int).SetString(raw[2:], 16)
		}
		gas, gasOK := quantity(fmt.Sprint(r["gasUsed"]))
		price, priceOK := quantity(fmt.Sprint(r["effectiveGasPrice"]))
		if !gasOK || !priceOK || gas.Sign() <= 0 || price.Sign() <= 0 {
			return fmt.Errorf("delegation gas/price missing")
		}
		fee := new(big.Int).Mul(gas, price)
		for address, want := range map[string]*big.Int{caller: new(big.Int).Neg(new(big.Int).Add(contractDelta, fee)), p.Withdrawal.Ledger.Contract: contractDelta} {
			a, hasPre := w.Diff.Pre[address]
			b, hasPost := w.Diff.Post[address]
			if address == p.Withdrawal.Ledger.Contract && contractDelta.Sign() == 0 && !hasPre && !hasPost {
				continue
			}
			before, bok := quantity(a.Balance)
			after := before
			aok := bok
			// Diff mode omits unchanged post fields. Only the zero-delta
			// contract balance may inherit pre-state; payer/deposit must change.
			if hasPost && (address == caller || contractDelta.Sign() != 0 || b.Balance != "") {
				after, aok = quantity(b.Balance)
			}
			if !hasPre || !bok || !aok || new(big.Int).Sub(after, before).Cmp(want) != 0 {
				return fmt.Errorf("delegation principal/gas ledger differs")
			}
		}
		if w.Diff.Post[caller].Nonce != w.Diff.Pre[caller].Nonce+1 {
			return fmt.Errorf("delegation nonce differs")
		}
	}
	return verifyAutoWithdrawalBalance(p.Withdrawal, new(big.Int).Add(stake, amount).String())
}

func delegationWithdrawalFixture() delegationWithdrawalProof {
	a := autoWithdrawalFixture()
	a.Delay = 30
	a.Created = withdrawalCredential{"100", "100", "130"}
	a.AfterEarly = a.Created
	a.Early.Timestamp = 105
	a.Early.Credentials = []withdrawalCredential{a.Created}
	a.Success.Timestamp = 131
	a.EarlyContractBefore, a.EarlyContractAfter = "0xc8", "0xc8"
	p := delegationWithdrawalProof{Withdrawal: a, Staker: "0x2222222222222222222222222222222222222222", Operator: "0x3333333333333333333333333333333333333333", StakerAmount: "100", DelegatorDelay: "30", StakerDelay: "70", DelegatedBefore: "0", DelegatedAfter: "100", DelegatedFinal: "0", PowerBefore: "100", PowerAfter: "200", PowerFinal: "100"}
	for i := 0; i < 2; i++ {
		name, value, before, after := "delegate", "0x64", "0x64", "0xc8"
		input := accounts.EncodeCallArgs("delegate(address,uint256)", accounts.Address(p.Staker), accounts.Uint(big.NewInt(100)))
		topics := []string{accounts.EventTopic("Delegated(address,address,uint256)"), delegationTopic(a.Ledger.Caller), delegationTopic(p.Staker)}
		callerAfter := "0x382"
		if i == 1 {
			name, value, before, after = "undelegate", "0x0", "0xc8", "0xc8"
			input = accounts.EncodeCallArgs("undelegate(address,uint256)", accounts.Address(p.Staker), accounts.Uint(big.NewInt(100)))
			topics = []string{accounts.EventTopic("Undelegated(address,address,address,uint256)"), delegationTopic(a.Ledger.Caller), delegationTopic(p.Staker), delegationTopic(a.Ledger.Caller)}
			callerAfter = "0x3e6"
		}
		logs, _ := json.Marshal([]accounts.Log{{Address: a.Ledger.Contract, Topics: topics, Data: "0x" + fmt.Sprintf("%064x", 100)}})
		w := credentialWithdrawal{Hash: name, Transaction: map[string]any{"hash": name, "blockHash": "b", "blockNumber": "0x4", "from": a.Ledger.Caller, "to": a.Ledger.Contract, "input": input, "value": value}, Receipt: map[string]any{"transactionHash": name, "blockHash": "b", "blockNumber": "0x4", "status": "0x1", "gasUsed": "0x1", "effectiveGasPrice": "0x2"}, LogsJSON: string(logs), Diff: claimDiff{Pre: map[string]claimDiffAccount{a.Ledger.Caller: {Balance: "0x3e8", Nonce: 1}, a.Ledger.Contract: {Balance: before}}, Post: map[string]claimDiffAccount{a.Ledger.Caller: {Balance: callerAfter, Nonce: 2}, a.Ledger.Contract: {Balance: after}}}}
		if i == 0 {
			p.Delegate = w
		} else {
			p.Undelegate = w
		}
	}
	return p
}

func delegationTopic(address string) string {
	return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}

func TestDelegationWithdrawalRejectsIncompleteLifecycle(t *testing.T) {
	if err := verifyDelegationWithdrawal(delegationWithdrawalFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*delegationWithdrawalProof)
	}{
		{"old scaled delay", func(p *delegationWithdrawalProof) { p.DelegatorDelay = "5"; p.StakerDelay = "15" }},
		{"wrong staker period", func(p *delegationWithdrawalProof) { p.StakerDelay = "71" }},
		{"early payout", func(p *delegationWithdrawalProof) {
			p.Withdrawal.Early.Diff.Post[p.Withdrawal.Ledger.Caller] = claimDiffAccount{Balance: "0xc6", Nonce: 2}
		}},
		{"early credential drained", func(p *delegationWithdrawalProof) { p.Withdrawal.AfterEarly = withdrawalCredential{"0", "0", "0"} }},
		{"late early call", func(p *delegationWithdrawalProof) { p.Withdrawal.Early.Timestamp = 130 }},
		{"wrong deposit fee", func(p *delegationWithdrawalProof) {
			p.Delegate.Diff.Post[p.Withdrawal.Ledger.Caller] = claimDiffAccount{Balance: "0x383", Nonce: 2}
		}},
		{"missing delegation event", func(p *delegationWithdrawalProof) { p.Delegate.LogsJSON = "[]" }},
		{"wrong undelegation event", func(p *delegationWithdrawalProof) { p.Undelegate.LogsJSON = p.Delegate.LogsJSON }},
		{"undelegate paid principal", func(p *delegationWithdrawalProof) {
			p.Undelegate.Diff.Post[p.Withdrawal.Ledger.Caller] = claimDiffAccount{Balance: "0x44a", Nonce: 2}
		}},
		{"power not increased", func(p *delegationWithdrawalProof) { p.PowerAfter = "100" }},
		{"power not restored", func(p *delegationWithdrawalProof) { p.PowerFinal = "200" }},
		{"delegation not removed", func(p *delegationWithdrawalProof) { p.DelegatedFinal = "100" }},
		{"operator delegated", func(p *delegationWithdrawalProof) { p.Operator = p.Withdrawal.Ledger.Caller }},
		{"mature payout missing", func(p *delegationWithdrawalProof) { p.Withdrawal.Success.LogsJSON = "[]" }},
		{"mature withdraw count changed", func(p *delegationWithdrawalProof) {
			p.Withdrawal.Success.Transaction["input"] = accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1)))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := delegationWithdrawalFixture()
			tc.change(&p)
			if verifyDelegationWithdrawal(p) == nil {
				t.Fatal("incomplete delegation lifecycle accepted")
			}
		})
	}
}

func TestDelegationWithdrawalSparseBalanceDiff(t *testing.T) {
	p := delegationWithdrawalFixture()
	contract := p.Withdrawal.Ledger.Contract
	post := p.Undelegate.Diff.Post[contract]
	post.Balance = ""
	p.Undelegate.Diff.Post[contract] = post
	if err := verifyDelegationWithdrawal(p); err != nil {
		t.Fatalf("unchanged balance omitted by diff-mode tracer must inherit pre-state: %v", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*delegationWithdrawalProof)
	}{
		{"missing changed deposit balance", func(p *delegationWithdrawalProof) {
			a := p.Delegate.Diff.Post[contract]
			a.Balance = ""
			p.Delegate.Diff.Post[contract] = a
		}},
		{"missing payer balance", func(p *delegationWithdrawalProof) {
			caller := p.Withdrawal.Ledger.Caller
			a := p.Undelegate.Diff.Post[caller]
			a.Balance = ""
			p.Undelegate.Diff.Post[caller] = a
		}},
		{"contract actually debited", func(p *delegationWithdrawalProof) {
			a := p.Undelegate.Diff.Post[contract]
			a.Balance = "0x64"
			p.Undelegate.Diff.Post[contract] = a
		}},
		{"contract pre-state absent", func(p *delegationWithdrawalProof) { delete(p.Undelegate.Diff.Pre, contract) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := delegationWithdrawalFixture()
			tc.change(&p)
			if verifyDelegationWithdrawal(p) == nil {
				t.Fatal("missing/changed balance was hidden by sparse diff handling")
			}
		})
	}
}
