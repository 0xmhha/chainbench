//go:build e2e

package main

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type claimLedgerProof struct {
	Hash, Caller, Staker, Rewardee, FeeRecipient, Preview, PreviewBlock string
	FeeRate, FeePrecision                                               string
	Transaction, Receipt                                                map[string]any
	Logs                                                                []accounts.Log
	Diff                                                                claimDiff
}

func verifyClaimLedger(p claimLedgerProof) error {
	tx, r := p.Transaction, p.Receipt
	input := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(p.Staker), accounts.Word([]byte{0}))
	if p.Hash == "" || tx["hash"] != p.Hash || r["transactionHash"] != p.Hash || tx["blockHash"] == nil || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] || r["status"] != "0x1" || tx["from"] != p.Caller || tx["to"] != e2eGovStaking || tx["input"] != input || tx["value"] != "0x0" || tx["transactionIndex"] != "0x0" {
		return fmt.Errorf("wrong actual claim input/receipt")
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(r["blockNumber"]), "0x"), 16, 64)
	if err != nil || n == 0 || p.PreviewBlock != fmt.Sprintf("0x%x", n-1) {
		return fmt.Errorf("preview not bound to execution parent")
	}
	addresses := map[string]bool{}
	for _, addr := range []string{p.Caller, p.Staker, p.Rewardee, p.FeeRecipient} {
		key := strings.ToLower(addr)
		if key == "" || addresses[key] {
			return fmt.Errorf("claim roles overlap")
		}
		addresses[key] = true
	}
	if !strings.HasPrefix(p.Preview, "0x") || len(p.Preview) != 258 {
		return fmt.Errorf("wrong preview ABI shape")
	}
	gross, ok := new(big.Int).SetString(p.Preview[2:66], 16)
	if !ok || gross.Sign() <= 0 {
		return fmt.Errorf("preview gross missing")
	}
	fee, ok := new(big.Int).SetString(p.Preview[66:130], 16)
	if !ok || fee.Sign() <= 0 || fee.Cmp(gross) >= 0 {
		return fmt.Errorf("preview fee missing")
	}
	rate, ok := new(big.Int).SetString(p.FeeRate, 10)
	if !ok || rate.Sign() <= 0 {
		return fmt.Errorf("rate missing")
	}
	precision, ok := new(big.Int).SetString(p.FeePrecision, 10)
	if !ok || precision.Sign() <= 0 || rate.Cmp(precision) >= 0 || new(big.Int).Div(new(big.Int).Mul(gross, rate), precision).Cmp(fee) != 0 {
		return fmt.Errorf("fee differs from configured rate")
	}
	paid := new(big.Int).Sub(gross, fee)
	claimed, feePaid := 0, 0
	topic := func(addr string) string {
		return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(addr), "0x")
	}
	for _, e := range p.Logs {
		if len(e.Topics) == 0 {
			continue
		}
		switch strings.ToLower(e.Topics[0]) {
		case strings.ToLower(accounts.EventTopic("Claimed(address,address,uint256,bool)")):
			claimed++
			if !strings.EqualFold(e.Address, e2eGovStaking) || len(e.Topics) != 3 || !strings.EqualFold(e.Topics[1], topic(p.Staker)) || !strings.EqualFold(e.Topics[2], topic(p.Caller)) || e.Data != "0x"+fmt.Sprintf("%064x%064x", paid, 0) {
				return fmt.Errorf("wrong Claimed event")
			}
		case strings.ToLower(accounts.EventTopic("FeePaid(address,uint256)")):
			feePaid++
			if !strings.EqualFold(e.Address, p.Rewardee) || len(e.Topics) != 2 || !strings.EqualFold(e.Topics[1], topic(p.FeeRecipient)) || e.Data != "0x"+fmt.Sprintf("%064x", fee) {
				return fmt.Errorf("wrong FeePaid event")
			}
		}
	}
	if claimed != 1 || feePaid != 1 {
		return fmt.Errorf("missing/duplicate claim events")
	}
	decode := func(raw string) (*big.Int, bool) {
		if !strings.HasPrefix(raw, "0x") {
			return nil, false
		}
		return new(big.Int).SetString(raw[2:], 16)
	}
	delta := func(address string) (*big.Int, error) {
		before, exists := p.Diff.Pre[address]
		if !exists {
			return nil, fmt.Errorf("missing before account")
		}
		after, exists := p.Diff.Post[address]
		if !exists {
			return nil, fmt.Errorf("missing after account")
		}
		a, ok := decode(before.Balance)
		if !ok {
			return nil, fmt.Errorf("missing before balance")
		}
		b, ok := decode(after.Balance)
		if !ok {
			return nil, fmt.Errorf("missing after balance")
		}
		return new(big.Int).Sub(b, a), nil
	}
	used, ok := decode(fmt.Sprint(r["gasUsed"]))
	if !ok || used.Sign() <= 0 {
		return fmt.Errorf("gas missing")
	}
	price, ok := decode(fmt.Sprint(r["effectiveGasPrice"]))
	if !ok || price.Sign() <= 0 {
		return fmt.Errorf("price missing")
	}
	gasFee := new(big.Int).Mul(used, price)
	want := map[string]*big.Int{p.Caller: new(big.Int).Sub(paid, gasFee), p.Rewardee: new(big.Int).Neg(gross), p.FeeRecipient: fee}
	if want[p.Caller].Sign() <= 0 {
		return fmt.Errorf("caller not paid beyond gas")
	}
	for address, n := range want {
		d, err := delta(address)
		if err != nil || d.Cmp(n) != 0 {
			return fmt.Errorf("exact claim balance mismatch: %s", address)
		}
	}
	nonce, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(tx["nonce"]), "0x"), 16, 64)
	if err != nil || p.Diff.Pre[p.Caller].Nonce != nonce || p.Diff.Post[p.Caller].Nonce != nonce+1 {
		return fmt.Errorf("caller nonce mismatch")
	}
	return nil
}

func claimLedgerFixture() claimLedgerProof {
	caller := "0x1111111111111111111111111111111111111111"
	staker := "0x2222222222222222222222222222222222222222"
	feeRecipient := "0x3333333333333333333333333333333333333333"
	rewardee := "0x4444444444444444444444444444444444444444"
	word := func(n int) string { return fmt.Sprintf("%064x", n) }
	topic := func(a string) string { return "0x" + "000000000000000000000000" + a[2:] }
	return claimLedgerProof{Hash: "tx", Caller: caller, Staker: staker, Rewardee: rewardee, FeeRecipient: feeRecipient, FeeRate: "1000", FeePrecision: "10000", Preview: "0x" + word(100) + word(10) + word(0) + word(0), PreviewBlock: "0x4", Transaction: map[string]any{"hash": "tx", "blockHash": "block", "blockNumber": "0x5", "from": caller, "to": e2eGovStaking, "input": accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(staker), accounts.Word([]byte{0})), "value": "0x0", "nonce": "0x1", "transactionIndex": "0x0"}, Receipt: map[string]any{"transactionHash": "tx", "blockHash": "block", "blockNumber": "0x5", "status": "0x1", "gasUsed": "0x1", "effectiveGasPrice": "0x2"}, Logs: []accounts.Log{
		{Address: e2eGovStaking, Topics: []string{accounts.EventTopic("Claimed(address,address,uint256,bool)"), topic(staker), topic(caller)}, Data: "0x" + word(90) + word(0)},
		{Address: rewardee, Topics: []string{accounts.EventTopic("FeePaid(address,uint256)"), topic(feeRecipient)}, Data: "0x" + word(10)},
	}, Diff: claimDiff{Pre: map[string]claimDiffAccount{caller: {Balance: "0x64", Nonce: 1}, rewardee: {Balance: "0x3e8"}, feeRecipient: {Balance: "0xa"}}, Post: map[string]claimDiffAccount{caller: {Balance: "0xbc", Nonce: 2}, rewardee: {Balance: "0x384"}, feeRecipient: {Balance: "0x14"}}}}
}
func TestClaimLedgerRejectsIncreaseOnlyEvidence(t *testing.T) {
	if err := verifyClaimLedger(claimLedgerFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*claimLedgerProof)
	}{
		{"missing events", func(p *claimLedgerProof) { p.Logs = nil }},
		{"wrong claim amount", func(p *claimLedgerProof) { p.Logs[0].Data = "0x" + fmt.Sprintf("%064x%064x", 91, 0) }},
		{"restake instead of payout", func(p *claimLedgerProof) { p.Logs[0].Data = "0x" + fmt.Sprintf("%064x%064x", 90, 1) }},
		{"wrong fee event", func(p *claimLedgerProof) { p.Logs[1].Data = "0x" + fmt.Sprintf("%064x", 9) }},
		{"duplicate claim event", func(p *claimLedgerProof) { p.Logs = append(p.Logs, p.Logs[0]) }},
		{"wrong rewardee debit", func(p *claimLedgerProof) { p.Diff.Post[p.Rewardee] = claimDiffAccount{Balance: "0x385"} }},
		{"wrong caller gas adjustment", func(p *claimLedgerProof) { p.Diff.Post[p.Caller] = claimDiffAccount{Balance: "0xbd", Nonce: 2} }},
		{"wrong fee recipient credit", func(p *claimLedgerProof) { p.Diff.Post[p.FeeRecipient] = claimDiffAccount{Balance: "0x15"} }},
		{"missing diff", func(p *claimLedgerProof) { p.Diff = claimDiff{} }},
		{"latest preview instead of parent", func(p *claimLedgerProof) { p.PreviewBlock = "latest" }},
		{"wrong fee ratio", func(p *claimLedgerProof) { p.FeeRate = "2000" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := claimLedgerFixture()
			tc.change(&p)
			if err := verifyClaimLedger(p); err == nil {
				t.Fatal("increase-only ledger accepted")
			}
		})
	}
}
