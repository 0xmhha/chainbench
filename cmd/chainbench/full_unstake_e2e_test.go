//go:build e2e

package main

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type fullUnstakeWitness struct {
	Hash, Operator, Staker, Minimum, Amount, Before, After string
	ActiveBefore, ActiveAfter                              bool
	Transaction, Receipt                                   map[string]any
	Logs                                                   []accounts.Log
}

func verifyFullUnstakeWitness(p fullUnstakeWitness) error {
	minimum, ok := new(big.Int).SetString(p.Minimum, 10)
	if !ok || minimum.Sign() <= 0 {
		return fmt.Errorf("minimum missing")
	}
	amount, ok := new(big.Int).SetString(p.Amount, 10)
	if !ok || amount.Cmp(new(big.Int).Mul(minimum, big.NewInt(2))) != 0 || p.Before != p.Amount || p.After != "0" || !p.ActiveBefore || p.ActiveAfter || strings.EqualFold(p.Operator, p.Staker) {
		return fmt.Errorf("not exact two-minimum active-to-inactive full unstake")
	}
	tx, r := p.Transaction, p.Receipt
	input := accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(amount))
	if p.Hash == "" || tx["hash"] != p.Hash || r["transactionHash"] != p.Hash || tx["from"] != p.Operator || tx["to"] != e2eGovStaking || tx["input"] != input || tx["value"] != "0x0" || r["status"] != "0x1" || tx["blockHash"] == nil || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] {
		return fmt.Errorf("wrong actual full unstake input/receipt")
	}
	removed, unstaked := 0, 0
	topic := "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(p.Staker), "0x")
	for _, e := range p.Logs {
		if len(e.Topics) == 0 {
			continue
		}
		kind := ""
		if strings.EqualFold(e.Topics[0], accounts.EventTopic("StakerRemoved(address)")) {
			removed++
			kind = "removed"
		}
		if strings.EqualFold(e.Topics[0], accounts.EventTopic("Unstaked(address,uint256)")) {
			unstaked++
			kind = "unstaked"
		}
		if kind == "" {
			continue
		}
		if !strings.EqualFold(e.Address, e2eGovStaking) || len(e.Topics) != 2 || !strings.EqualFold(e.Topics[1], topic) || kind == "removed" && e.Data != "0x" || kind == "unstaked" && e.Data != "0x"+fmt.Sprintf("%064x", amount) {
			return fmt.Errorf("wrong exact %s event", kind)
		}
	}
	if removed != 1 || unstaked != 1 {
		return fmt.Errorf("missing/duplicate removal or unstake")
	}
	return nil
}

func fullUnstakeFixture() fullUnstakeWitness {
	op := "0x1111111111111111111111111111111111111111"
	staker := "0x2222222222222222222222222222222222222222"
	topic := "0x" + strings.Repeat("0", 24) + staker[2:]
	return fullUnstakeWitness{Hash: "hash", Operator: op, Staker: staker, Minimum: "100", Amount: "200", Before: "200", After: "0", ActiveBefore: true, Transaction: map[string]any{"hash": "hash", "from": op, "to": e2eGovStaking, "input": accounts.EncodeCallArgs("unstake(uint256)", accounts.Uint(big.NewInt(200))), "value": "0x0", "blockHash": "block", "blockNumber": "0x5"}, Receipt: map[string]any{"transactionHash": "hash", "status": "0x1", "blockHash": "block", "blockNumber": "0x5"}, Logs: []accounts.Log{{Address: e2eGovStaking, Topics: []string{accounts.EventTopic("StakerRemoved(address)"), topic}, Data: "0x"}, {Address: e2eGovStaking, Topics: []string{accounts.EventTopic("Unstaked(address,uint256)"), topic}, Data: "0x" + fmt.Sprintf("%064x", 200)}}}
}

func TestFullUnstakeRejectsBalanceOnlyAndMissingEvents(t *testing.T) {
	if e := verifyFullUnstakeWitness(fullUnstakeFixture()); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name   string
		change func(*fullUnstakeWitness)
	}{
		{"minimum instead of twice", func(p *fullUnstakeWitness) { p.Amount = "100" }},
		{"wrong before amount", func(p *fullUnstakeWitness) { p.Before = "199" }},
		{"not active before", func(p *fullUnstakeWitness) { p.ActiveBefore = false }},
		{"still active after", func(p *fullUnstakeWitness) { p.ActiveAfter = true }},
		{"no events", func(p *fullUnstakeWitness) { p.Logs = nil }},
		{"missing removed event", func(p *fullUnstakeWitness) { p.Logs = p.Logs[1:] }},
		{"wrong unstaked amount", func(p *fullUnstakeWitness) { p.Logs[1].Data = "0x" + fmt.Sprintf("%064x", 199) }},
		{"wrong subject", func(p *fullUnstakeWitness) { p.Logs[0].Topics[1] = "0x" + strings.Repeat("0", 64) }},
		{"wrong emitter", func(p *fullUnstakeWitness) { p.Logs[0].Address = p.Operator }},
		{"duplicate removed event", func(p *fullUnstakeWitness) { p.Logs = append(p.Logs, p.Logs[0]) }},
		{"wrong input", func(p *fullUnstakeWitness) { p.Transaction["input"] = "0x" }},
		{"wrong receipt status", func(p *fullUnstakeWitness) { p.Receipt["status"] = "0x0" }},
		{"wrong receipt block", func(p *fullUnstakeWitness) { p.Receipt["blockHash"] = "other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := fullUnstakeFixture()
			tc.change(&p)
			if verifyFullUnstakeWitness(p) == nil {
				t.Fatal("balance-only evidence accepted incomplete full unstake", tc.name)
			}
		})
	}
}
