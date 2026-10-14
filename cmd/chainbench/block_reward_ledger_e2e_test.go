//go:build e2e

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/dsl"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type rewardLedgerBlock struct {
	Number                   uint64
	Hash, ParentHash, Reward string
	Validators               []string
	Transactions             []string
}
type rewardLedgerProof struct {
	Staker, Operator, Rewardee, Before, After, Minimum, BLSKey string
	Start, End                                                 uint64
	Logs                                                       []accounts.Log
	Blocks                                                     []rewardLedgerBlock
	ContractStakers, OriginalOperatorPreview, StakerPreview    string
}

func verifyBlockRewardLedger(p rewardLedgerProof) error {
	if p.End != p.Start+15 || len(p.Blocks) != 15 || p.Staker == "" || p.Operator == "" || p.Rewardee == "" || strings.EqualFold(p.Staker, p.Operator) || strings.EqualFold(p.Rewardee, p.Operator) || strings.EqualFold(p.Rewardee, p.Staker) {
		return fmt.Errorf("wrong reward interval/roles")
	}
	minimum, ok := new(big.Int).SetString(p.Minimum, 10)
	if !ok || minimum.Sign() <= 0 {
		return fmt.Errorf("minimum absent")
	}
	contract := "0x" + fmt.Sprintf("%064x%064x", 32, 1) + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(p.Staker), "0x")
	if strings.ToLower(p.ContractStakers) != contract {
		return fmt.Errorf("not sole registered staker")
	}
	count := 0
	for _, e := range p.Logs {
		if len(e.Topics) == 0 || !strings.EqualFold(e.Topics[0], accounts.EventTopic("StakerRegistered(address,address,address,address,uint256,uint256,bytes)")) {
			continue
		}
		count++
		if !strings.EqualFold(e.Address, e2eGovStaking) || len(e.Topics) != 2 || !strings.EqualFold(e.Topics[1], "0x"+strings.Repeat("0", 24)+strings.TrimPrefix(p.Staker, "0x")) {
			return fmt.Errorf("wrong registered subject")
		}
		raw, err := hex.DecodeString(strings.TrimPrefix(e.Data, "0x"))
		if err != nil || len(raw) != 288 {
			return fmt.Errorf("wrong registration ABI")
		}
		addr := func(i int) string { return "0x" + hex.EncodeToString(raw[i*32+12:(i+1)*32]) }
		if !strings.EqualFold(addr(0), p.Operator) || !strings.EqualFold(addr(1), p.Rewardee) || !strings.EqualFold(addr(2), p.Staker) || new(big.Int).SetBytes(raw[96:128]).Sign() != 0 || new(big.Int).SetBytes(raw[128:160]).Cmp(minimum) != 0 || new(big.Int).SetBytes(raw[160:192]).Cmp(big.NewInt(192)) != 0 || new(big.Int).SetBytes(raw[192:224]).Cmp(big.NewInt(48)) != 0 || !strings.EqualFold("0x"+hex.EncodeToString(raw[224:272]), p.BLSKey) {
			return fmt.Errorf("registration fields/keys mismatch")
		}
		for _, b := range raw[272:] {
			if b != 0 {
				return fmt.Errorf("BLS padding invalid")
			}
		}
	}
	if count != 1 {
		return fmt.Errorf("missing/duplicate registration")
	}
	total := big.NewInt(0)
	for i, b := range p.Blocks {
		if b.Number != p.Start+uint64(i)+1 || b.Hash == "" || len(b.Transactions) != 0 || i > 0 && b.ParentHash != p.Blocks[i-1].Hash {
			return fmt.Errorf("unbound blocks or unrelated writes")
		}
		seen := map[string]bool{}
		for _, v := range b.Validators {
			v = strings.ToLower(v)
			if v == "" || seen[v] {
				return fmt.Errorf("invalid validator set")
			}
			seen[v] = true
		}
		if !seen[strings.ToLower(p.Staker)] {
			return fmt.Errorf("staker outside eligible validators")
		}
		if !strings.HasPrefix(b.Reward, "0x") {
			return fmt.Errorf("reward RPC invalid")
		}
		reward, ok := new(big.Int).SetString(b.Reward[2:], 16)
		if !ok || reward.Sign() <= 0 {
			return fmt.Errorf("reward RPC invalid")
		}
		total.Add(total, reward)
	}
	before, ok := new(big.Int).SetString(p.Before, 10)
	if !ok || before.Sign() < 0 {
		return fmt.Errorf("balance absent")
	}
	after, ok := new(big.Int).SetString(p.After, 10)
	if !ok || new(big.Int).Sub(after, before).Cmp(total) != 0 {
		return fmt.Errorf("reward balance differs from exact RPC sum")
	}
	return nil
}

func rewardLedgerFixture() rewardLedgerProof {
	staker := "0x1111111111111111111111111111111111111111"
	operator := "0x2222222222222222222222222222222222222222"
	rewardee := "0x3333333333333333333333333333333333333333"
	key := strings.Repeat("ab", 48)
	encoded := accounts.EncodeCallArgs("ignored(address,address,address,uint256,uint256,bytes)", accounts.Address(operator), accounts.Address(rewardee), accounts.Address(staker), accounts.Uint(big.NewInt(0)), accounts.Uint(big.NewInt(100)), accounts.Bytes(bytes.Repeat([]byte{0xab}, 48)))
	p := rewardLedgerProof{Staker: staker, Operator: operator, Rewardee: rewardee, Before: "10", After: "25", Minimum: "100", BLSKey: "0x" + key, Start: 30, End: 45, Logs: []accounts.Log{{Address: e2eGovStaking, Topics: []string{accounts.EventTopic("StakerRegistered(address,address,address,address,uint256,uint256,bytes)"), "0x" + strings.Repeat("0", 24) + staker[2:]}, Data: "0x" + encoded[10:]}}, ContractStakers: "0x" + fmt.Sprintf("%064x%064x", 32, 1) + strings.Repeat("0", 24) + staker[2:]}
	for n := uint64(31); n <= 45; n++ {
		p.Blocks = append(p.Blocks, rewardLedgerBlock{Number: n, Hash: fmt.Sprint(n), ParentHash: fmt.Sprint(n - 1), Reward: "0x1", Validators: []string{staker}})
	}
	return p
}
func TestBlockRewardLedgerRejectsUnboundGrowth(t *testing.T) {
	if err := verifyBlockRewardLedger(rewardLedgerFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*rewardLedgerProof)
	}{
		{"short interval", func(p *rewardLedgerProof) { p.Blocks = p.Blocks[:2] }},
		{"wrong reward sum", func(p *rewardLedgerProof) { p.After = "26" }},
		{"bad reward RPC", func(p *rewardLedgerProof) { p.Blocks[14].Reward = "oops" }},
		{"nonlinked blocks", func(p *rewardLedgerProof) { p.Blocks[14].ParentHash = "other" }},
		{"not eligible validator", func(p *rewardLedgerProof) { p.Blocks[14].Validators = nil }},
		{"unrelated transaction", func(p *rewardLedgerProof) { p.Blocks[14].Transactions = []string{"claim"} }},
		{"wrong registered rewardee", func(p *rewardLedgerProof) { p.Rewardee = p.Operator }},
		{"no registration event", func(p *rewardLedgerProof) { p.Logs = nil }},
		{"more stakers", func(p *rewardLedgerProof) { p.ContractStakers = "0x00" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := rewardLedgerFixture()
			tc.change(&p)
			if err := verifyBlockRewardLedger(p); err == nil {
				t.Fatal("unbound reward increase accepted")
			}
		})
	}
}

const rewardHTTPModules = "admin,eth,debug,miner,net,txpool,personal,web3,istanbul,wemix"

func TestRewardHandoffRequiresExplicitBriocheAndHTTPNamespace(t *testing.T) {
	overlay := map[string]any{"config": map[string]any{"brioche": map[string]any{"blockReward": json.Number("1000000000000000000")}}}
	path := writeHandoffCase(t, overlay, rewardNodeLaunchScopes())
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := dsl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	config := spec.Chain.GenesisOverlay["config"].(map[string]any)
	if _, ok := config["brioche"].(map[string]any); !ok {
		t.Fatal("explicit Brioche object missing")
	}
	got := strings.Join(spec.EnvLaunch["node1"], ";")
	if !strings.Contains(got, "http.api="+rewardHTTPModules) {
		t.Fatal("reward RPC HTTP namespace not declared", got)
	}
}

func TestDefaultHandoffHasNoExtraLaunchOverrides(t *testing.T) {
	path := writeHandoffCase(t, nil)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := dsl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.EnvLaunch) != 0 {
		t.Fatal("optional proof settings leaked into default handoff")
	}
}

func rewardNodeLaunchScopes() map[string]map[string]any {
	result := map[string]map[string]any{}
	for i := 1; i <= 4; i++ {
		result[fmt.Sprint("node", i)] = map[string]any{"http": true, "http.api": rewardHTTPModules}
	}
	return result
}
