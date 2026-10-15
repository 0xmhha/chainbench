//go:build e2e

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type unstakeWithdrawalWitness struct {
	Ledger                                                                                   credentialExpiryProof
	Delay                                                                                    uint64
	Created, AfterEarly                                                                      withdrawalCredential
	BeforeCredentialIndex, BeforeWithdrawalIndex, EarlyCredentialIndex, EarlyWithdrawalIndex string
	EarlyTimestamp                                                                           uint64
	Early                                                                                    minedStakingFailure
	Success                                                                                  credentialWithdrawal
}

func verifyUnstakeWithdrawal(p unstakeWithdrawalWitness) error {
	parse := func(s string) (uint64, bool) {
		n, ok := new(big.Int).SetString(s, 10)
		returnValue := uint64(0)
		if ok && n.IsUint64() {
			returnValue = n.Uint64()
			return returnValue, true
		}
		return 0, false
	}
	requested, ok := parse(p.Created.RequestTime)
	if !ok {
		return fmt.Errorf("request time missing")
	}
	mature, ok := parse(p.Created.WithdrawableTime)
	if !ok || p.Delay == 0 || mature != requested+p.Delay || p.Created.Amount != p.Ledger.Amount || p.AfterEarly != p.Created || p.EarlyTimestamp < requested || p.EarlyTimestamp >= mature {
		return fmt.Errorf("early window/credential/configuration differs")
	}
	n := p.Ledger.N
	if p.BeforeCredentialIndex != fmt.Sprint(n+1) || p.BeforeWithdrawalIndex != fmt.Sprint(n) || p.EarlyCredentialIndex != p.BeforeCredentialIndex || p.EarlyWithdrawalIndex != p.BeforeWithdrawalIndex {
		return fmt.Errorf("early refusal changed pointers")
	}
	input := accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1)))
	if e := validateMinedStakingFailure(p.Early, p.Ledger.Caller, input, new(big.Int), "insufficient withdrawable credentials"); e != nil {
		return e
	}
	if p.Success.Timestamp < mature || p.Success.Index != n || len(p.Success.Credentials) != 1 || p.Success.Credentials[0] != (withdrawalCredential{"0", "0", "0"}) || p.Success.CredentialIndex != fmt.Sprint(n+1) || p.Success.WithdrawalIndex != fmt.Sprint(n+1) {
		return fmt.Errorf("mature withdrawal did not drain exact slot")
	}
	return verifyWithdrawalLedger(p.Ledger, p.Success)
}

func unstakeWithdrawalFixture() unstakeWithdrawalWitness {
	ledger, w := withdrawalLedgerFixture()
	w.Index = 0
	w.Timestamp = 116
	w.Credentials = []withdrawalCredential{{"0", "0", "0"}}
	w.CredentialIndex = "1"
	w.WithdrawalIndex = "1"
	logs, _ := json.Marshal([]accounts.Log{{Address: ledger.Contract, Topics: []string{accounts.EventTopic("Withdrawn(address,uint256,uint256)"), "0x000000000000000000000000" + ledger.Caller[2:]}, Data: "0x" + fmt.Sprintf("%064x%064x", 0, 100)}})
	w.LogsJSON = string(logs)
	input := accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1)))
	reason := "insufficient withdrawable credentials"
	early := minedStakingFailure{Hash: "early", Transaction: map[string]any{"hash": "early", "blockHash": "b", "from": ledger.Caller, "to": ledger.Contract, "input": input, "value": "0x0", "gas": "0x7a120"}, Receipt: map[string]any{"transactionHash": "early", "blockHash": "b", "blockNumber": "0x4", "status": "0x0", "gasUsed": "0x8000"}, Trace: stakingFailureTrace{Failed: true, ReturnValue: accounts.EncodeCallArgs("Error(string)", accounts.StringArg(reason))}}
	return unstakeWithdrawalWitness{Ledger: ledger, Delay: 15, Created: withdrawalCredential{"100", "100", "115"}, AfterEarly: withdrawalCredential{"100", "100", "115"}, BeforeCredentialIndex: "1", BeforeWithdrawalIndex: "0", EarlyCredentialIndex: "1", EarlyWithdrawalIndex: "0", EarlyTimestamp: 104, Early: early, Success: w}
}

func TestUnstakeWithdrawalRejectsClockAndPartialProof(t *testing.T) {
	if e := verifyUnstakeWithdrawal(unstakeWithdrawalFixture()); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name   string
		change func(*unstakeWithdrawalWitness)
	}{
		{"early already mature", func(p *unstakeWithdrawalWitness) { p.EarlyTimestamp = 115 }},
		{"wrong configured delay", func(p *unstakeWithdrawalWitness) { p.Delay = 14 }},
		{"wrong principal", func(p *unstakeWithdrawalWitness) { p.Created.Amount = "99" }},
		{"early slot changed", func(p *unstakeWithdrawalWitness) { p.AfterEarly = withdrawalCredential{"0", "0", "0"} }},
		{"early pointer changed", func(p *unstakeWithdrawalWitness) { p.EarlyWithdrawalIndex = "1" }},
		{"missing mined early refusal", func(p *unstakeWithdrawalWitness) { p.Early.Receipt = nil }},
		{"wrong early reason", func(p *unstakeWithdrawalWitness) { p.Early.Trace.ReturnValue = "0x" }},
		{"success still locked", func(p *unstakeWithdrawalWitness) { p.Success.Timestamp = 114 }},
		{"wrong success index", func(p *unstakeWithdrawalWitness) { p.Success.Index = 1 }},
		{"slot not drained", func(p *unstakeWithdrawalWitness) { p.Success.Credentials[0] = p.Created }},
		{"wrong success pointer", func(p *unstakeWithdrawalWitness) { p.Success.WithdrawalIndex = "0" }},
		{"wrong success payout", func(p *unstakeWithdrawalWitness) {
			p.Success.Diff.Post[p.Ledger.Caller] = claimDiffAccount{Balance: "0xc7", Nonce: 2}
		}},
		{"missing withdrawal event", func(p *unstakeWithdrawalWitness) { p.Success.LogsJSON = "[]" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := unstakeWithdrawalFixture()
			tc.change(&p)
			if verifyUnstakeWithdrawal(p) == nil {
				t.Fatal("success-only receipt hid an incorrect early/mature withdrawal", tc.name)
			}
		})
	}
}
