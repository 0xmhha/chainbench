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

type operatorClaimProof struct {
	Claim                                              claimLedgerProof
	OperatorPreview, MappedStaker, Staked, TotalStaked string
}

func verifyOperatorClaimProof(p operatorClaimProof) error {
	c := p.Claim
	if len(p.OperatorPreview) != 258 || !strings.HasPrefix(p.OperatorPreview, "0x") || p.OperatorPreview[2:130] != strings.Repeat("0", 128) || len(c.Preview) != 258 || p.OperatorPreview[130:] != c.Preview[130:] || p.MappedStaker != c.Staker {
		return fmt.Errorf("original operator preview or registered mapping differs")
	}
	staked, ok := new(big.Int).SetString(p.Staked, 10)
	if !ok || staked.Sign() <= 0 || p.Staked != p.TotalStaked {
		return fmt.Errorf("not solely self-staked reward")
	}
	pClaim := c
	tx, r := pClaim.Transaction, pClaim.Receipt
	input := accounts.EncodeCallArgs("claim(address,bool)", accounts.Address(pClaim.Staker), accounts.Word([]byte{0}))
	if pClaim.Hash == "" || tx["hash"] != pClaim.Hash || r["transactionHash"] != pClaim.Hash || tx["blockHash"] == nil || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] || r["status"] != "0x1" || tx["from"] != pClaim.Caller || tx["to"] != e2eGovStaking || tx["input"] != input || tx["value"] != "0x0" || tx["transactionIndex"] != "0x0" {
		return fmt.Errorf("wrong actual claim input/receipt")
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(r["blockNumber"]), "0x"), 16, 64)
	if err != nil || n == 0 || pClaim.PreviewBlock != fmt.Sprintf("0x%x", n-1) {
		return fmt.Errorf("preview not bound to execution parent")
	}
	addresses := map[string]bool{}
	for _, addr := range []string{pClaim.Caller, pClaim.Staker, pClaim.Rewardee} {
		key := strings.ToLower(addr)
		if key == "" || addresses[key] {
			return fmt.Errorf("claim roles overlap")
		}
		addresses[key] = true
	}
	if !strings.HasPrefix(pClaim.Preview, "0x") || len(pClaim.Preview) != 258 {
		return fmt.Errorf("wrong preview ABI shape")
	}
	gross, ok := new(big.Int).SetString(pClaim.Preview[2:66], 16)
	if !ok || gross.Sign() <= 0 {
		return fmt.Errorf("preview gross missing")
	}
	fee, ok := new(big.Int).SetString(pClaim.Preview[66:130], 16)
	if !ok || fee.Sign() != 0 {
		return fmt.Errorf("preview fee missing")
	}
	rate, ok := new(big.Int).SetString(pClaim.FeeRate, 10)
	if !ok || rate.Sign() != 0 {
		return fmt.Errorf("rate missing")
	}
	precision, ok := new(big.Int).SetString(pClaim.FeePrecision, 10)
	if !ok || precision.Sign() <= 0 || rate.Cmp(precision) >= 0 || new(big.Int).Div(new(big.Int).Mul(gross, rate), precision).Cmp(fee) != 0 {
		return fmt.Errorf("fee differs from configured rate")
	}
	paid := new(big.Int).Sub(gross, fee)
	claimed, feePaid, rewardPaid := 0, 0, 0
	topic := func(addr string) string {
		return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(addr), "0x")
	}
	for _, e := range pClaim.Logs {
		if len(e.Topics) == 0 {
			continue
		}
		switch strings.ToLower(e.Topics[0]) {
		case strings.ToLower(accounts.EventTopic("Claimed(address,address,uint256,bool)")):
			claimed++
			if !strings.EqualFold(e.Address, e2eGovStaking) || len(e.Topics) != 3 || !strings.EqualFold(e.Topics[1], topic(pClaim.Staker)) || !strings.EqualFold(e.Topics[2], topic(pClaim.Caller)) || e.Data != "0x"+fmt.Sprintf("%064x%064x", paid, 0) {
				return fmt.Errorf("wrong Claimed event")
			}
		case strings.ToLower(accounts.EventTopic("RewardPaid(address,uint256)")):
			rewardPaid++
			if !strings.EqualFold(e.Address, pClaim.Rewardee) || len(e.Topics) != 2 || !strings.EqualFold(e.Topics[1], topic(pClaim.Caller)) || e.Data != "0x"+fmt.Sprintf("%064x", paid) {
				return fmt.Errorf("wrong RewardPaid event")
			}
		case strings.ToLower(accounts.EventTopic("FeePaid(address,uint256)")):
			feePaid++
			if !strings.EqualFold(e.Address, pClaim.Rewardee) || len(e.Topics) != 2 || !strings.EqualFold(e.Topics[1], topic(pClaim.FeeRecipient)) || e.Data != "0x"+fmt.Sprintf("%064x", fee) {
				return fmt.Errorf("wrong FeePaid event")
			}
		}
	}
	if claimed != 1 || feePaid != 0 || rewardPaid != 1 {
		return fmt.Errorf("missing/duplicate claim events")
	}
	decode := func(raw string) (*big.Int, bool) {
		if !strings.HasPrefix(raw, "0x") {
			return nil, false
		}
		return new(big.Int).SetString(raw[2:], 16)
	}
	delta := func(address string) (*big.Int, error) {
		before, exists := pClaim.Diff.Pre[address]
		if !exists {
			return nil, fmt.Errorf("missing before account")
		}
		after, exists := pClaim.Diff.Post[address]
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
	want := map[string]*big.Int{pClaim.Caller: new(big.Int).Sub(paid, gasFee), pClaim.Rewardee: new(big.Int).Neg(gross)}
	if want[pClaim.Caller].Sign() <= 0 {
		return fmt.Errorf("caller not paid beyond gas")
	}
	for address, n := range want {
		d, err := delta(address)
		if err != nil || d.Cmp(n) != 0 {
			return fmt.Errorf("exact claim balance mismatch: %s", address)
		}
	}
	nonce, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(tx["nonce"]), "0x"), 16, 64)
	if err != nil || pClaim.Diff.Pre[pClaim.Caller].Nonce != nonce || pClaim.Diff.Post[pClaim.Caller].Nonce != nonce+1 {
		return fmt.Errorf("caller nonce mismatch")
	}
	return nil
}

func operatorClaimFixture() operatorClaimProof {
	c := claimLedgerFixture()
	word := func(n int) string { return fmt.Sprintf("%064x", n) }
	c.Preview = "0x" + word(100) + word(0) + word(0) + word(0)
	c.FeeRate = "0"
	c.FeeRecipient = c.Staker
	c.Transaction["gas"] = "0x5208"
	c.Diff.Post[c.Caller] = claimDiffAccount{Balance: "0xc6", Nonce: 2}
	c.Logs[0].Data = "0x" + word(100) + word(0)
	c.Logs[1] = accounts.Log{Address: c.Rewardee, Topics: []string{accounts.EventTopic("RewardPaid(address,uint256)"), "0x" + strings.Repeat("0", 24) + c.Caller[2:]}, Data: "0x" + word(100)}
	return operatorClaimProof{Claim: c, OperatorPreview: "0x" + strings.Repeat("0", 256), MappedStaker: c.Staker, Staked: "1000", TotalStaked: "1000"}
}

func TestOperatorClaimRejectsIncreaseOnlyAndWrongPreview(t *testing.T) {
	if err := verifyOperatorClaimProof(operatorClaimFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*operatorClaimProof)
	}{
		{"nonzero original operator preview", func(p *operatorClaimProof) {
			p.OperatorPreview = "0x" + fmt.Sprintf("%064x", 1) + strings.Repeat("0", 192)
		}},
		{"missing original operator preview", func(p *operatorClaimProof) { p.OperatorPreview = "" }},
		{"wrong mapped staker", func(p *operatorClaimProof) { p.MappedStaker = p.Claim.Caller }},
		{"delegated stake", func(p *operatorClaimProof) { p.TotalStaked = "1001" }},
		{"latest preview", func(p *operatorClaimProof) { p.Claim.PreviewBlock = "latest" }},
		{"wrong preview gross", func(p *operatorClaimProof) {
			p.Claim.Preview = "0x" + fmt.Sprintf("%064x", 101) + strings.Repeat("0", 192)
		}},
		{"missing events", func(p *operatorClaimProof) { p.Claim.Logs = nil }},
		{"wrong claim amount", func(p *operatorClaimProof) {
			p.Claim.Logs[0].Data = "0x" + fmt.Sprintf("%064x", 99) + strings.Repeat("0", 64)
		}},
		{"wrong reward recipient", func(p *operatorClaimProof) {
			p.Claim.Logs[1].Topics[1] = "0x" + strings.Repeat("0", 24) + p.Claim.Staker[2:]
		}},
		{"unexpected fee", func(p *operatorClaimProof) {
			p.Claim.Logs = append(p.Claim.Logs, accounts.Log{Topics: []string{accounts.EventTopic("FeePaid(address,uint256)")}})
		}},
		{"wrong caller gas credit", func(p *operatorClaimProof) {
			p.Claim.Diff.Post[p.Claim.Caller] = claimDiffAccount{Balance: "0xc7", Nonce: 2}
		}},
		{"wrong rewardee debit", func(p *operatorClaimProof) { p.Claim.Diff.Post[p.Claim.Rewardee] = claimDiffAccount{Balance: "0x385"} }},
		{"wrong nonce", func(p *operatorClaimProof) {
			p.Claim.Diff.Post[p.Claim.Caller] = claimDiffAccount{Balance: "0xc6", Nonce: 3}
		}},
		{"wrong claim input", func(p *operatorClaimProof) { p.Claim.Transaction["input"] = "0x" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := operatorClaimFixture()
			tc.change(&p)
			if verifyOperatorClaimProof(p) == nil {
				t.Fatal("increase-only criterion accepted malformed operator claim")
			}
		})
	}
}
