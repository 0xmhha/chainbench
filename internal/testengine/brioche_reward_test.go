package testengine_test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

type briocheRewardRow struct {
	Height                                     uint64
	Parent, Block                              map[string]any
	RPCQueryTag, BeforeTag, AfterTag, Proposer string
	Reward, Before, After, StakerCount         string
}
type briocheRewardProof struct {
	WBFTBase, BriocheBase              string
	First, Period, Finish, Times, Rate uint64
	ActualNodes                        []string
	Rows                               []briocheRewardRow
}

func verifyBriocheReward(p briocheRewardProof) error {
	if p.WBFTBase != "2000000000000000000" || p.BriocheBase != p.WBFTBase || p.First != 10 || p.Period != 10 || p.Finish != 1000 || p.Times != 4 || p.Rate != 50 || len(p.Rows) != 4 || len(p.ActualNodes) != 4 {
		return fmt.Errorf("not declared exact2ETH/first10/50percent fixture")
	}
	nodes := map[string]bool{}
	for _, a := range p.ActualNodes {
		a = strings.ToLower(a)
		if !previousSealerHex(a, 20) || nodes[a] {
			return fmt.Errorf("invalid actual proposer identities")
		}
		nodes[a] = true
	}
	for i, r := range p.Rows {
		h := []uint64{5, 9, 10, 15}[i]
		tag, parent := fmt.Sprintf("0x%x", h), fmt.Sprintf("0x%x", h-1)
		if r.Height != h || r.Block["number"] != tag || r.Parent["number"] != parent || r.RPCQueryTag != tag || r.BeforeTag != parent || r.AfterTag != tag || !previousSealerHex(r.Block["hash"], 32) || !previousSealerHex(r.Parent["hash"], 32) || r.Block["parentHash"] != r.Parent["hash"] || r.Block["hash"] == r.Parent["hash"] || !strings.EqualFold(fmt.Sprint(r.Block["miner"]), r.Proposer) || !nodes[strings.ToLower(r.Proposer)] {
			return fmt.Errorf("wrong exact mined reward, parent or proposer origin")
		}
		txs, ok := r.Block["transactions"].([]any)
		if !ok || len(txs) != 0 || r.Block["gasUsed"] != "0x0" || r.StakerCount != "0x"+strings.Repeat("0", 64) {
			return fmt.Errorf("not isolated empty block without registered stakers")
		}
		expected := big.NewInt(2000000000000000000)
		if h >= 10 {
			expected.Div(expected, big.NewInt(2))
		}
		reward, valid := briocheQuantity(r.Reward)
		before, validBefore := briocheQuantity(r.Before)
		after, validAfter := briocheQuantity(r.After)
		if !valid || !validBefore || !validAfter || reward.Cmp(expected) != 0 || new(big.Int).Sub(after, before).Cmp(expected) != 0 {
			return fmt.Errorf("RPC/configured half and exact proposer mint ledger differ")
		}
	}
	return nil
}

func briocheQuantity(s string) (*big.Int, bool) {
	if !strings.HasPrefix(s, "0x") || len(s) <= 2 {
		return nil, false
	}
	for _, r := range s[2:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return nil, false
		}
	}
	return new(big.Int).SetString(s[2:], 16)
}
func briocheRewardFixture() briocheRewardProof {
	p := briocheRewardProof{WBFTBase: "2000000000000000000", BriocheBase: "2000000000000000000", First: 10, Period: 10, Finish: 1000, Times: 4, Rate: 50}
	for i := 1; i <= 4; i++ {
		p.ActualNodes = append(p.ActualNodes, fmt.Sprintf("0x%040x", i))
	}
	for i, h := range []uint64{5, 9, 10, 15} {
		reward := big.NewInt(2000000000000000000)
		if h >= 10 {
			reward.Div(reward, big.NewInt(2))
		}
		before := big.NewInt(5000000000000000000)
		after := new(big.Int).Add(before, reward)
		parentHash := fmt.Sprintf("0x%064x", 100+i*2)
		hash := fmt.Sprintf("0x%064x", 101+i*2)
		p.Rows = append(p.Rows, briocheRewardRow{Height: h, Parent: map[string]any{"number": fmt.Sprintf("0x%x", h-1), "hash": parentHash}, Block: map[string]any{"number": fmt.Sprintf("0x%x", h), "hash": hash, "parentHash": parentHash, "miner": p.ActualNodes[i], "transactions": []any{}, "gasUsed": "0x0"}, RPCQueryTag: fmt.Sprintf("0x%x", h), BeforeTag: fmt.Sprintf("0x%x", h-1), AfterTag: fmt.Sprintf("0x%x", h), Proposer: p.ActualNodes[i], Reward: "0x" + reward.Text(16), Before: "0x" + before.Text(16), After: "0x" + after.Text(16), StakerCount: "0x" + strings.Repeat("0", 64)})
	}
	return p
}

func TestBriocheRewardRejectsDecreaseOnly(t *testing.T) {
	if err := verifyBriocheReward(briocheRewardFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*briocheRewardProof)
	}{
		{"wrong WBFT baseline", func(p *briocheRewardProof) { p.WBFTBase = "1000000000000000000" }},
		{"wrong Brioche baseline", func(p *briocheRewardProof) { p.BriocheBase = "1000000000000000000" }},
		{"wrong first halving", func(p *briocheRewardProof) { p.First = 11 }},
		{"wrong period", func(p *briocheRewardProof) { p.Period = 20 }},
		{"wrong coefficient", func(p *briocheRewardProof) { p.Rate = 75 }},
		{"missing boundary rows", func(p *briocheRewardProof) { p.Rows = []briocheRewardRow{p.Rows[0], p.Rows[3]} }},
		{"RPC latest", func(p *briocheRewardProof) { p.Rows[0].RPCQueryTag = "latest" }},
		{"before tag not parent", func(p *briocheRewardProof) { p.Rows[0].BeforeTag = p.Rows[0].AfterTag }},
		{"after latest", func(p *briocheRewardProof) { p.Rows[0].AfterTag = "latest" }},
		{"wrong mined height", func(p *briocheRewardProof) { p.Rows[0].Block["number"] = "0x6" }},
		{"wrong parent link", func(p *briocheRewardProof) { p.Rows[0].Block["parentHash"] = "other" }},
		{"same parent and child hash", func(p *briocheRewardProof) { p.Rows[0].Block["hash"] = p.Rows[0].Parent["hash"] }},
		{"different balance recipient", func(p *briocheRewardProof) { p.Rows[0].Proposer = p.ActualNodes[1] }},
		{"unowned proposer", func(p *briocheRewardProof) {
			a := "0x" + strings.Repeat("ff", 20)
			p.Rows[0].Proposer = a
			p.Rows[0].Block["miner"] = a
		}},
		{"wrong pre amount still positive", func(p *briocheRewardProof) { p.Rows[0].Reward = "0x" + big.NewInt(3000000000000000000).Text(16) }},
		{"wrong post amount still lower", func(p *briocheRewardProof) { p.Rows[3].Reward = "0x" + big.NewInt(750000000000000000).Text(16) }},
		{"halving boundary unchanged", func(p *briocheRewardProof) { p.Rows[2].Reward = p.Rows[1].Reward }},
		{"wrong actual balance delta", func(p *briocheRewardProof) { p.Rows[0].After = "0x1" }},
		{"invalid balance", func(p *briocheRewardProof) { p.Rows[0].Before = "0x-1" }},
		{"transaction fees not excluded", func(p *briocheRewardProof) { p.Rows[0].Block["transactions"] = []any{"tx"} }},
		{"gas not zero", func(p *briocheRewardProof) { p.Rows[0].Block["gasUsed"] = "0x5208" }},
		{"registered staker distribution", func(p *briocheRewardProof) { p.Rows[0].StakerCount = "0x" + fmt.Sprintf("%064x", 1) }},
		{"duplicate node identities", func(p *briocheRewardProof) { p.ActualNodes[3] = p.ActualNodes[0] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := briocheRewardFixture()
			tc.change(&p)
			if verifyBriocheReward(p) == nil {
				t.Fatal("decrease hid wrong configured reward or actual proposer ledger")
			}
		})
	}
}
