//go:build e2e

package main

import (
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"math/big"
	"strings"
	"testing"
	"time"
)

type withdrawalCredential struct{ Amount, RequestTime, WithdrawableTime string }
type credentialWithdrawal struct {
	Hash                             string
	Timestamp                        uint64
	Index                            uint64
	Transaction, Receipt             map[string]any
	Credentials                      []withdrawalCredential
	CredentialIndex, WithdrawalIndex string
	Diff                             claimDiff
	LogsJSON                         string
}
type credentialExpiryProof struct {
	Caller, Contract, Amount    string
	N                           uint64
	StakerDelay, DelegatorDelay string
	Created                     []withdrawalCredential
	First, Second               credentialWithdrawal
}

func verifyCredentialExpiry(p credentialExpiryProof) error {
	amount, ok := new(big.Int).SetString(p.Amount, 10)
	if !ok || amount.Sign() <= 0 || len(p.Created) != 2 || len(p.First.Credentials) != 2 || len(p.Second.Credentials) != 2 {
		return fmt.Errorf("missing credential context")
	}
	parse := func(value string) (uint64, error) {
		n, ok := new(big.Int).SetString(value, 10)
		if !ok || !n.IsUint64() {
			return 0, fmt.Errorf("invalid credential time")
		}
		return n.Uint64(), nil
	}
	long, err := parse(p.StakerDelay)
	if err != nil {
		return err
	}
	short, err := parse(p.DelegatorDelay)
	if err != nil || short == 0 || long <= short {
		return fmt.Errorf("unbonding windows not distinct")
	}
	for i, c := range p.Created {
		requested, err := parse(c.RequestTime)
		if err != nil {
			return err
		}
		mature, err := parse(c.WithdrawableTime)
		if err != nil {
			return err
		}
		delay := long
		if i == 1 {
			delay = short
		}
		if c.Amount != p.Amount || mature != requested+delay {
			return fmt.Errorf("creation time/delay mismatch")
		}
	}
	firstMature, _ := parse(p.Created[0].WithdrawableTime)
	secondMature, _ := parse(p.Created[1].WithdrawableTime)
	zero := withdrawalCredential{"0", "0", "0"}
	if p.First.Timestamp < secondMature || p.First.Timestamp >= firstMature || p.Second.Timestamp < firstMature || p.Second.Timestamp <= p.First.Timestamp || p.First.Index != p.N+1 || p.Second.Index != p.N || p.First.Credentials[0] != p.Created[0] || p.First.Credentials[1] != zero || p.Second.Credentials[0] != zero || p.Second.Credentials[1] != zero || p.First.CredentialIndex != fmt.Sprint(p.N+2) || p.Second.CredentialIndex != fmt.Sprint(p.N+2) || p.First.WithdrawalIndex != fmt.Sprint(p.N) || p.Second.WithdrawalIndex != fmt.Sprint(p.N+2) {
		return fmt.Errorf("wrong actual skip/drain/time/pointer")
	}
	return nil
}

func credentialExpiryFixture() credentialExpiryProof {
	p := credentialExpiryProof{Caller: "caller", Contract: e2eGovStaking, Amount: "100", N: 0, StakerDelay: "15", DelegatorDelay: "5", Created: []withdrawalCredential{{"100", "100", "115"}, {"100", "101", "106"}}}
	p.First = credentialWithdrawal{Hash: "first", Timestamp: 110, Index: 1, Receipt: map[string]any{"status": "0x1"}, Credentials: []withdrawalCredential{p.Created[0], {"0", "0", "0"}}, CredentialIndex: "2", WithdrawalIndex: "0"}
	p.Second = credentialWithdrawal{Hash: "second", Timestamp: 116, Index: 0, Receipt: map[string]any{"status": "0x1"}, Credentials: []withdrawalCredential{{"0", "0", "0"}, {"0", "0", "0"}}, CredentialIndex: "2", WithdrawalIndex: "2"}
	return p
}
func TestCredentialExpiryRejectsClockAndDrainAmbiguity(t *testing.T) {
	if err := verifyCredentialExpiry(credentialExpiryFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*credentialExpiryProof)
	}{
		{"staker already mature first", func(p *credentialExpiryProof) { p.First.Timestamp = 115 }},
		{"delegator still locked", func(p *credentialExpiryProof) { p.First.Timestamp = 105 }},
		{"staker drained early", func(p *credentialExpiryProof) { p.First.Credentials[0] = withdrawalCredential{"0", "0", "0"} }},
		{"delegator not drained", func(p *credentialExpiryProof) { p.First.Credentials[1] = p.Created[1] }},
		{"wrong first index", func(p *credentialExpiryProof) { p.First.Index = 0 }},
		{"wrong second index", func(p *credentialExpiryProof) { p.Second.Index = 1 }},
		{"second not mature", func(p *credentialExpiryProof) { p.Second.Timestamp = 114 }},
		{"wrong request plus delay", func(p *credentialExpiryProof) { p.Created[0].WithdrawableTime = "114" }},
		{"wrong oldest live pointer", func(p *credentialExpiryProof) { p.First.WithdrawalIndex = "2" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := credentialExpiryFixture()
			tc.change(&p)
			if err := verifyCredentialExpiry(p); err == nil {
				t.Fatal("ambiguous withdrawal accepted")
			}
		})
	}
}

func verifyWithdrawalLedger(p credentialExpiryProof, w credentialWithdrawal) error {
	tx, r := w.Transaction, w.Receipt
	input := accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1)))
	if w.Hash == "" || tx["hash"] != w.Hash || r["transactionHash"] != w.Hash || tx["blockHash"] == nil || tx["blockHash"] != r["blockHash"] || tx["blockNumber"] != r["blockNumber"] || r["status"] != "0x1" || tx["from"] != p.Caller || tx["to"] != p.Contract || tx["input"] != input || tx["value"] != "0x0" {
		return fmt.Errorf("wrong actual withdrawal input/receipt")
	}
	var logs []accounts.Log
	if err := json.Unmarshal([]byte(w.LogsJSON), &logs); err != nil {
		return err
	}
	count := 0
	amount, ok := new(big.Int).SetString(p.Amount, 10)
	if !ok {
		return fmt.Errorf("amount missing")
	}
	for _, e := range logs {
		if len(e.Topics) == 0 || !strings.EqualFold(e.Topics[0], accounts.EventTopic("Withdrawn(address,uint256,uint256)")) {
			continue
		}
		count++
		if !strings.EqualFold(e.Address, p.Contract) || len(e.Topics) != 2 || !strings.EqualFold(e.Topics[1], "0x"+strings.Repeat("0", 24)+p.Caller[2:]) || e.Data != "0x"+fmt.Sprintf("%064x%064x", w.Index, amount) {
			return fmt.Errorf("wrong Withdrawn event index/amount")
		}
	}
	if count != 1 {
		return fmt.Errorf("missing/duplicate withdrawal event")
	}
	parse := func(s string) (*big.Int, bool) {
		if !strings.HasPrefix(s, "0x") {
			return nil, false
		}
		return new(big.Int).SetString(s[2:], 16)
	}
	used, ok := parse(fmt.Sprint(r["gasUsed"]))
	if !ok || used.Sign() <= 0 {
		return fmt.Errorf("gas missing")
	}
	price, ok := parse(fmt.Sprint(r["effectiveGasPrice"]))
	if !ok || price.Sign() <= 0 {
		return fmt.Errorf("price missing")
	}
	fee := new(big.Int).Mul(used, price)
	expected := map[string]*big.Int{p.Caller: new(big.Int).Sub(amount, fee), p.Contract: new(big.Int).Neg(amount)}
	for address, want := range expected {
		a, ok := parse(w.Diff.Pre[address].Balance)
		if !ok {
			return fmt.Errorf("before balance missing")
		}
		b, ok := parse(w.Diff.Post[address].Balance)
		if !ok || new(big.Int).Sub(b, a).Cmp(want) != 0 {
			return fmt.Errorf("withdrawal amount/fee mismatch")
		}
	}
	if w.Diff.Post[p.Caller].Nonce != w.Diff.Pre[p.Caller].Nonce+1 {
		return fmt.Errorf("nonce mismatch")
	}
	return nil
}
func credentialAt(t *testing.T, c *rpc.Client, caller string, index uint64) withdrawalCredential {
	t.Helper()
	data := accounts.EncodeCallArgs("credentials(address,uint256)", accounts.Address(caller), accounts.Uint(new(big.Int).SetUint64(index)))
	var raw string
	if err := c.Call(t.Context(), "eth_call", &raw, map[string]string{"to": e2eGovStaking, "data": data}, "latest"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(raw, "0x") || len(raw) != 194 {
		t.Fatal("wrong credential ABI", raw)
	}
	values := []string{}
	for i := 2; i < len(raw); i += 64 {
		n, ok := new(big.Int).SetString(raw[i:i+64], 16)
		if !ok {
			t.Fatal("credential word invalid")
		}
		values = append(values, n.String())
	}
	return withdrawalCredential{values[0], values[1], values[2]}
}
func credentialPointers(t *testing.T, c *rpc.Client, caller string) (string, string) {
	t.Helper()
	raw, err := c.EthCall(t.Context(), e2eGovStaking, accounts.EncodeCallArgs("userCredential(address)", accounts.Address(caller)))
	if err != nil || len(raw) != 130 {
		t.Fatal("pointer ABI", err, raw)
	}
	a, ok := new(big.Int).SetString(raw[2:66], 16)
	if !ok {
		t.Fatal("credential pointer invalid")
	}
	b, ok := new(big.Int).SetString(raw[66:130], 16)
	if !ok {
		t.Fatal("withdrawal pointer invalid")
	}
	return a.String(), b.String()
}
func waitCredentialTime(t *testing.T, c *rpc.Client, target uint64) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		block, err := c.BlockByNumber(t.Context(), "latest")
		if err != nil {
			t.Fatal(err)
		}
		if block.Timestamp >= target {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("chain timestamp did not reach maturity")
		}
		select {
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func withdrawalLedgerFixture() (credentialExpiryProof, credentialWithdrawal) {
	p := credentialExpiryProof{Caller: "0x1111111111111111111111111111111111111111", Contract: e2eGovStaking, Amount: "100"}
	w := credentialWithdrawal{Hash: "tx", Index: 1, Transaction: map[string]any{"hash": "tx", "blockHash": "block", "blockNumber": "0x5", "from": p.Caller, "to": p.Contract, "input": accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(1))), "value": "0x0"}, Receipt: map[string]any{"transactionHash": "tx", "blockHash": "block", "blockNumber": "0x5", "status": "0x1", "gasUsed": "0x1", "effectiveGasPrice": "0x2"}, Diff: claimDiff{Pre: map[string]claimDiffAccount{p.Caller: {Balance: "0x64", Nonce: 1}, p.Contract: {Balance: "0x3e8"}}, Post: map[string]claimDiffAccount{p.Caller: {Balance: "0xc6", Nonce: 2}, p.Contract: {Balance: "0x384"}}}}
	encoded, _ := json.Marshal([]accounts.Log{{Address: p.Contract, Topics: []string{accounts.EventTopic("Withdrawn(address,uint256,uint256)"), "0x" + strings.Repeat("0", 24) + p.Caller[2:]}, Data: "0x" + fmt.Sprintf("%064x%064x", 1, 100)}})
	w.LogsJSON = string(encoded)
	return p, w
}
func TestWithdrawalLedgerRejectsThresholdAndWrongSlot(t *testing.T) {
	p, w := withdrawalLedgerFixture()
	if err := verifyWithdrawalLedger(p, w); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*credentialWithdrawal)
	}{
		{"missing event", func(w *credentialWithdrawal) { w.LogsJSON = "[]" }},
		{"wrong slot", func(w *credentialWithdrawal) { w.Index = 0 }},
		{"missing diff", func(w *credentialWithdrawal) { w.Diff = claimDiff{} }},
		{"greater than threshold only", func(w *credentialWithdrawal) { w.Diff.Post[p.Caller] = claimDiffAccount{Balance: "0xc7", Nonce: 2} }},
		{"wrong contract debit", func(w *credentialWithdrawal) { w.Diff.Post[p.Contract] = claimDiffAccount{Balance: "0x385"} }},
		{"wrong withdrawal input", func(w *credentialWithdrawal) {
			w.Transaction["input"] = accounts.EncodeCallArgs("withdraw(uint256)", accounts.Uint(big.NewInt(2)))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, w := withdrawalLedgerFixture()
			tc.change(&w)
			if err := verifyWithdrawalLedger(p, w); err == nil {
				t.Fatal("incorrect withdrawal ledger accepted")
			}
		})
	}
}
