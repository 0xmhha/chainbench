//go:build e2e

package main

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type claimDiffAccount struct {
	Balance string
	Nonce   uint64
}
type claimDiff struct{ Pre, Post map[string]claimDiffAccount }
type claimGuardProof struct {
	Name, Caller, Staker string
	Rewardees            []string
	RewardBalances       []string
	Protected            []string
	Failure              minedStakingFailure
	Diff                 claimDiff
}

func verifyClaimGuardDiff(p claimGuardProof) error {
	if p.Caller == "" || p.Staker == "" || strings.EqualFold(p.Caller, p.Staker) || len(p.Rewardees) != 2 || len(p.RewardBalances) != 2 || len(p.Protected) != 4 {
		return fmt.Errorf("incomplete registered/reward context")
	}
	for _, balance := range p.RewardBalances {
		n, ok := new(big.Int).SetString(balance, 10)
		if !ok || n.Sign() <= 0 {
			return fmt.Errorf("no accrued reward")
		}
	}
	protected := map[string]bool{}
	for _, addr := range p.Protected {
		key := strings.ToLower(addr)
		if key == "" || protected[key] || strings.EqualFold(addr, p.Caller) {
			return fmt.Errorf("invalid protected accounts")
		}
		protected[key] = true
	}
	if !protected[strings.ToLower(e2eGovStaking)] || strings.EqualFold(p.Rewardees[0], p.Rewardees[1]) {
		return fmt.Errorf("missing protected contract or distinct rewardees")
	}
	for _, address := range p.Rewardees {
		if !protected[strings.ToLower(address)] {
			return fmt.Errorf("missing protected rewardee")
		}
	}
	input := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(p.Staker), accounts.Word([]byte{0}))
	if err := validateMinedStakingFailure(p.Failure, p.Caller, input, big.NewInt(0), "no reward to claim"); err != nil {
		return err
	}
	for _, side := range []map[string]claimDiffAccount{p.Diff.Pre, p.Diff.Post} {
		if len(side) == 0 {
			return fmt.Errorf("missing actual prestate diff")
		}
		for address := range side {
			if protected[strings.ToLower(address)] {
				return fmt.Errorf("protected claim state changed")
			}
		}
	}
	before, exists := p.Diff.Pre[p.Caller]
	if !exists {
		return fmt.Errorf("actual caller before state absent")
	}
	after, exists := p.Diff.Post[p.Caller]
	if !exists || after.Nonce != before.Nonce+1 {
		return fmt.Errorf("actual caller nonce did not advance")
	}
	decode := func(value string) (*big.Int, bool) {
		if !strings.HasPrefix(value, "0x") {
			return nil, false
		}
		return new(big.Int).SetString(value[2:], 16)
	}
	first, ok := decode(before.Balance)
	if !ok {
		return fmt.Errorf("caller balance absent")
	}
	last, ok := decode(after.Balance)
	if !ok {
		return fmt.Errorf("caller balance absent")
	}
	used, ok := decode(fmt.Sprint(p.Failure.Receipt["gasUsed"]))
	if !ok {
		return fmt.Errorf("gas missing")
	}
	price, ok := decode(fmt.Sprint(p.Failure.Receipt["effectiveGasPrice"]))
	if !ok {
		return fmt.Errorf("price missing")
	}
	if new(big.Int).Sub(first, last).Cmp(new(big.Int).Mul(used, price)) != 0 {
		return fmt.Errorf("caller balance did not change by exact fee only")
	}
	return nil
}

func claimGuardFixture() claimGuardProof {
	caller := "0x1111111111111111111111111111111111111111"
	staker := "0x2222222222222222222222222222222222222222"
	input := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0}))
	return claimGuardProof{Name: "operatorA-to-B", Caller: caller, Staker: staker, Rewardees: []string{"rewardA", "rewardB"}, RewardBalances: []string{"10", "20"}, Protected: []string{"rewardA", "rewardB", "operatorB", e2eGovStaking}, Failure: minedStakingFailure{Hash: "tx", Transaction: map[string]any{"hash": "tx", "blockHash": "block", "from": caller, "to": e2eGovStaking, "input": input, "value": "0x0", "gas": "0x7a120"}, Receipt: map[string]any{"transactionHash": "tx", "blockHash": "block", "blockNumber": "0x5", "status": "0x0", "gasUsed": "0x64", "effectiveGasPrice": "0x2"}, Trace: stakingFailureTrace{Failed: true, ReturnValue: accounts.EncodeCallArgs("Error(string)", accounts.StringArg("no reward to claim"))}}, Diff: claimDiff{Pre: map[string]claimDiffAccount{caller: {Balance: "0x3e8", Nonce: 1}}, Post: map[string]claimDiffAccount{caller: {Balance: "0x320", Nonce: 2}}}}
}
func TestClaimGuardDiffRejectsMissingOrTransferredState(t *testing.T) {
	if err := verifyClaimGuardDiff(claimGuardFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*claimGuardProof)
	}{
		{"missing actual diff", func(p *claimGuardProof) { p.Diff = claimDiff{} }},
		{"rewardee changed", func(p *claimGuardProof) { p.Diff.Post["rewardB"] = claimDiffAccount{Balance: "0x1"} }},
		{"operatorB changed", func(p *claimGuardProof) { p.Diff.Post["operatorB"] = claimDiffAccount{Balance: "0x1"} }},
		{"caller received reward", func(p *claimGuardProof) { p.Diff.Post[p.Caller] = claimDiffAccount{Balance: "0x3e9", Nonce: 2} }},
		{"no nonce effect", func(p *claimGuardProof) { p.Diff.Post[p.Caller] = claimDiffAccount{Balance: "0x320", Nonce: 1} }},
		{"no accrued reward", func(p *claimGuardProof) { p.RewardBalances[1] = "0" }},
		{"wrong mined reason", func(p *claimGuardProof) {
			p.Failure.Trace.ReturnValue = accounts.EncodeCallArgs("Error(string)", accounts.StringArg("other"))
		}},
		{"incomplete protection", func(p *claimGuardProof) { p.Protected = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := claimGuardFixture()
			tc.change(&p)
			if err := verifyClaimGuardDiff(p); err == nil {
				t.Fatal("weak claim theft proof accepted")
			}
		})
	}
}
