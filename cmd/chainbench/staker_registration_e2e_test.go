//go:build e2e

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type registrationWitness struct {
	BeforeCode                                                  string
	Reward                                                      rewardLedgerProof
	Hash, PoP, Mapped, Code, Implementation, ImplementationCode string
	Active                                                      bool
	Transaction, Receipt                                        map[string]any
}

func verifyRegistrationWitness(w registrationWitness) error {
	p := w.Reward
	if w.BeforeCode != "0x" {
		return fmt.Errorf("rewardee existed before registration")
	}
	minimum, ok := new(big.Int).SetString(p.Minimum, 10)
	if !ok || minimum.Sign() <= 0 {
		return fmt.Errorf("minimum missing")
	}
	pk, e := hex.DecodeString(strings.TrimPrefix(p.BLSKey, "0x"))
	if e != nil || len(pk) != 48 {
		return fmt.Errorf("wrong BLS public key")
	}
	pop, e := hex.DecodeString(strings.TrimPrefix(w.PoP, "0x"))
	if e != nil || len(pop) != 96 {
		return fmt.Errorf("wrong proof of possession")
	}
	input := accounts.EncodeCallArgs("registerStaker(uint256,address,address,uint256,bytes,bytes)", accounts.Uint(minimum), accounts.Address(p.Staker), accounts.Address(p.Staker), accounts.Uint(big.NewInt(0)), accounts.Bytes(pk), accounts.Bytes(pop))
	tx, r := w.Transaction, w.Receipt
	if w.Hash == "" || tx["hash"] != w.Hash || r["transactionHash"] != w.Hash || tx["from"] != p.Operator || tx["to"] != e2eGovStaking || tx["input"] != input || tx["value"] != "0x"+minimum.Text(16) || tx["blockHash"] == nil || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] || r["status"] != "0x1" || !w.Active || !strings.EqualFold(w.Mapped, p.Staker) || strings.EqualFold(p.Staker, p.Operator) || strings.EqualFold(p.Rewardee, p.Staker) || strings.EqualFold(p.Rewardee, p.Operator) {
		return fmt.Errorf("wrong registration receipt/input/roles/mapping")
	}
	code, e := hex.DecodeString(strings.TrimPrefix(w.Code, "0x"))
	if e != nil || len(code) == 0 {
		return fmt.Errorf("new rewardee code missing")
	}
	impl, e := hex.DecodeString(strings.TrimPrefix(w.ImplementationCode, "0x"))
	if e != nil || len(impl) == 0 || w.Implementation != "0x"+strings.Repeat("0", 60)+"1002" {
		return fmt.Errorf("new rewardee implementation link missing")
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
	return nil
}

func registrationFixture() registrationWitness {
	staker := "0x1111111111111111111111111111111111111111"
	operator := "0x2222222222222222222222222222222222222222"
	rewardee := "0x3333333333333333333333333333333333333333"
	pk := make([]byte, 48)
	pop := make([]byte, 96)
	minimum := big.NewInt(100)
	data := "0x" + strings.Repeat("0", 24) + operator[2:] + strings.Repeat("0", 24) + rewardee[2:] + strings.Repeat("0", 24) + staker[2:] + fmt.Sprintf("%064x%064x%064x%064x", 0, minimum, 192, 48) + hex.EncodeToString(pk) + strings.Repeat("0", 32)
	r := rewardLedgerProof{Staker: staker, Operator: operator, Rewardee: rewardee, Minimum: "100", BLSKey: "0x" + hex.EncodeToString(pk), Logs: []accounts.Log{{Address: e2eGovStaking, Topics: []string{accounts.EventTopic("StakerRegistered(address,address,address,address,uint256,uint256,bytes)"), "0x" + strings.Repeat("0", 24) + staker[2:]}, Data: data}}}
	input := accounts.EncodeCallArgs("registerStaker(uint256,address,address,uint256,bytes,bytes)", accounts.Uint(minimum), accounts.Address(staker), accounts.Address(staker), accounts.Uint(big.NewInt(0)), accounts.Bytes(pk), accounts.Bytes(pop))
	return registrationWitness{Reward: r, BeforeCode: "0x", Hash: "hash", PoP: "0x" + hex.EncodeToString(pop), Mapped: staker, Code: "0x1234", Implementation: "0x" + strings.Repeat("0", 60) + "1002", ImplementationCode: "0x5678", Active: true, Transaction: map[string]any{"hash": "hash", "from": operator, "to": e2eGovStaking, "input": input, "value": "0x64", "blockHash": "block", "blockNumber": "0x5"}, Receipt: map[string]any{"transactionHash": "hash", "status": "0x1", "blockHash": "block", "blockNumber": "0x5"}}
}

func TestRegistrationWitnessRejectsActiveOnlyEvidence(t *testing.T) {
	if e := verifyRegistrationWitness(registrationFixture()); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name   string
		change func(*registrationWitness)
	}{
		{"already deployed before registration", func(w *registrationWitness) { w.BeforeCode = "0x1234" }},
		{"no deployed code", func(w *registrationWitness) { w.Code = "0x" }},
		{"no implementation code", func(w *registrationWitness) { w.ImplementationCode = "0x" }},
		{"wrong implementation", func(w *registrationWitness) { w.Implementation = "0x" + strings.Repeat("0", 64) }},
		{"wrong mapping", func(w *registrationWitness) { w.Mapped = w.Reward.Operator }},
		{"missing event", func(w *registrationWitness) { w.Reward.Logs = nil }},
		{"wrong event emitter", func(w *registrationWitness) { w.Reward.Logs[0].Address = w.Reward.Rewardee }},
		{"wrong event subject", func(w *registrationWitness) { w.Reward.Logs[0].Topics[1] = "0x" + strings.Repeat("0", 64) }},
		{"duplicate event", func(w *registrationWitness) { w.Reward.Logs = append(w.Reward.Logs, w.Reward.Logs[0]) }},
		{"wrong BLS key", func(w *registrationWitness) { w.Reward.BLSKey = "0x" + strings.Repeat("11", 48) }},
		{"wrong minimum", func(w *registrationWitness) { w.Reward.Minimum = "101" }},
		{"wrong input", func(w *registrationWitness) { w.Transaction["input"] = "0x" }},
		{"wrong value", func(w *registrationWitness) { w.Transaction["value"] = "0x63" }},
		{"wrong receipt", func(w *registrationWitness) { w.Receipt["status"] = "0x0" }},
		{"wrong block", func(w *registrationWitness) { w.Receipt["blockHash"] = "other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := registrationFixture()
			tc.change(&w)
			if verifyRegistrationWitness(w) == nil {
				t.Fatal("active-only registration accepted invalid deployment/input/event", tc.name)
			}
		})
	}
}
