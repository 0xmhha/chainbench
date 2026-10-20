//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type stakingFailureTrace struct {
	Failed      bool   `json:"failed"`
	ReturnValue string `json:"returnValue"`
}
type minedStakingFailure struct {
	Hash                 string
	Transaction, Receipt map[string]any
	Trace                stakingFailureTrace
}

// validateMinedStakingFailure binds a typed revert trace to the actual mined input.
func validateMinedStakingFailure(p minedStakingFailure, sender, input string, value *big.Int, reason string) error {
	tx, r := p.Transaction, p.Receipt
	if p.Hash == "" || tx["hash"] != p.Hash || r["transactionHash"] != p.Hash || tx["blockHash"] == nil || tx["blockHash"] == "" || tx["blockHash"] != r["blockHash"] || r["blockNumber"] == nil || r["status"] != "0x0" || tx["from"] != sender || tx["to"] != e2eGovStaking || tx["input"] != input || value == nil || tx["value"] != "0x"+value.Text(16) {
		return fmt.Errorf("missing or mismatched mined staking input/receipt")
	}
	gas, e1 := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(tx["gas"]), "0x"), 16, 64)
	used, e2 := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(r["gasUsed"]), "0x"), 16, 64)
	if e1 != nil || e2 != nil || used == 0 || used >= gas {
		return fmt.Errorf("not a partial-gas contract revert")
	}
	expected := accounts.EncodeCallArgs("Error(string)", accounts.StringArg(reason))
	if !p.Trace.Failed || !strings.EqualFold(strings.TrimPrefix(p.Trace.ReturnValue, "0x"), strings.TrimPrefix(expected, "0x")) {
		return fmt.Errorf("wrong or absent mined revert reason")
	}
	return nil
}

// exactFeeEvent requires one event with the contract, indexed staker and all rates.
func exactFeeEvent(logs []accounts.Log, staker, signature string, values ...*big.Int) bool {
	count := 0
	matched := false
	for _, event := range logs {
		if len(event.Topics) == 0 || !strings.EqualFold(event.Topics[0], accounts.EventTopic(signature)) {
			continue
		}
		count++
		if !strings.EqualFold(event.Address, e2eGovStaking) || len(event.Topics) != 2 || !strings.EqualFold(event.Topics[1], "0x"+strings.Repeat("0", 24)+strings.TrimPrefix(staker, "0x")) || len(event.Data) != 2+64*len(values) || !strings.HasPrefix(event.Data, "0x") {
			continue
		}
		valid := true
		for i, want := range values {
			actual, ok := new(big.Int).SetString(event.Data[2+64*i:2+64*(i+1)], 16)
			if !ok || want == nil || actual.Cmp(want) != 0 {
				valid = false
			}
		}
		matched = valid
	}
	return count == 1 && matched
}

func TestMinedStakingFailureRejectsFalseEvidence(t *testing.T) {
	good := func() minedStakingFailure {
		return minedStakingFailure{Hash: "tx", Transaction: map[string]any{"hash": "tx", "blockHash": "block", "from": "sender", "to": e2eGovStaking, "input": "0x01", "value": "0x1", "gas": "0x7a120"}, Receipt: map[string]any{"transactionHash": "tx", "blockHash": "block", "blockNumber": "0x5", "status": "0x0", "gasUsed": "0x8000"}, Trace: stakingFailureTrace{Failed: true, ReturnValue: accounts.EncodeCallArgs("Error(string)", accounts.StringArg("expected reason"))}}
	}
	if err := validateMinedStakingFailure(good(), "sender", "0x01", big.NewInt(1), "expected reason"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*minedStakingFailure)
	}{
		{"call-only refusal", func(p *minedStakingFailure) { p.Hash = ""; p.Receipt = nil; p.Transaction = nil }},
		{"success receipt", func(p *minedStakingFailure) { p.Receipt["status"] = "0x1" }},
		{"wrong tx", func(p *minedStakingFailure) { p.Receipt["transactionHash"] = "other" }},
		{"wrong block", func(p *minedStakingFailure) { p.Receipt["blockHash"] = "other" }},
		{"wrong sender", func(p *minedStakingFailure) { p.Transaction["from"] = "other" }},
		{"wrong target", func(p *minedStakingFailure) { p.Transaction["to"] = "other" }},
		{"wrong input", func(p *minedStakingFailure) { p.Transaction["input"] = "0x" }},
		{"wrong value", func(p *minedStakingFailure) { p.Transaction["value"] = "0x0" }},
		{"wrong failure", func(p *minedStakingFailure) {
			p.Trace.ReturnValue = accounts.EncodeCallArgs("Error(string)", accounts.StringArg("unrelated"))
		}},
		{"no failure trace", func(p *minedStakingFailure) { p.Trace.Failed = false }},
		{"out of gas", func(p *minedStakingFailure) { p.Receipt["gasUsed"] = "0x7a120" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := good()
			tc.change(&p)
			if validateMinedStakingFailure(p, "sender", "0x01", big.NewInt(1), "expected reason") == nil {
				t.Fatal("invalid mined staking refusal accepted")
			}
		})
	}
}

func TestFeeEventRejectsWrongArguments(t *testing.T) {
	const subject = "0x1111111111111111111111111111111111111111"
	signature := "ChangingFeeRateRequested(address,uint256,uint256)"
	good := func() accounts.Log {
		return accounts.Log{Address: e2eGovStaking, Topics: []string{accounts.EventTopic(signature), "0x000000000000000000000000" + subject[2:]}, Data: "0x" + "0000000000000000000000000000000000000000000000000000000000000000" + "00000000000000000000000000000000000000000000000000000000000000fa"}
	}
	for _, tc := range []struct {
		name   string
		change func(*accounts.Log)
	}{
		{"wrong emitter", func(p *accounts.Log) { p.Address = subject }},
		{"wrong staker", func(p *accounts.Log) {
			p.Topics[1] = "0x" + "0000000000000000000000000000000000000000000000000000000000000000"
		}},
		{"wrong old rate", func(p *accounts.Log) {
			p.Data = "0x" + "0000000000000000000000000000000000000000000000000000000000000001" + p.Data[66:]
		}},
		{"wrong new rate", func(p *accounts.Log) { p.Data = p.Data[:len(p.Data)-2] + "fb" }},
		{"missing data", func(p *accounts.Log) { p.Data = "0x" }},
		{"trailing data", func(p *accounts.Log) { p.Data += "00" }},
		{"extra topic", func(p *accounts.Log) { p.Topics = append(p.Topics, p.Topics[1]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := good()
			tc.change(&p)
			if exactFeeEvent([]accounts.Log{p}, subject, signature, big.NewInt(0), big.NewInt(250)) {
				t.Fatal("incorrect fee event accepted")
			}
		})
	}
	if !exactFeeEvent([]accounts.Log{good()}, subject, signature, big.NewInt(0), big.NewInt(250)) {
		t.Fatal("valid event refused")
	}
	if exactFeeEvent([]accounts.Log{good(), good()}, subject, signature, big.NewInt(0), big.NewInt(250)) {
		t.Fatal("duplicate event accepted")
	}
}

// stakingRejectedTx explicitly submits gas so SDK estimation cannot replace a
// mined failure with a preflight refusal. It preserves the complete RPC facts.
func stakingRejectedTx(t *testing.T, c *rpc.Client, w accounts.Wallet, input string, value *big.Int, reason string) minedStakingFailure {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	data, err := hex.DecodeString(strings.TrimPrefix(input, "0x"))
	if err != nil {
		t.Fatal(err)
	}
	var priceHex string
	if err := c.Call(ctx, "eth_gasPrice", &priceHex); err != nil {
		t.Fatal(err)
	}
	price, ok := new(big.Int).SetString(strings.TrimPrefix(priceHex, "0x"), 16)
	if !ok {
		t.Fatal("invalid gas price", priceHex)
	}
	before, err := c.BalanceAt(ctx, w.Address())
	if err != nil {
		t.Fatal(err)
	}
	hash, err := w.SendLegacyTx(ctx, accounts.LegacyTxArgs{ToHex: e2eGovStaking, Value: value, Data: data, Gas: 500000, GasPrice: price})
	if err != nil {
		t.Fatal(err)
	}
	p := minedStakingFailure{Hash: hash}
	for {
		if err := c.Call(ctx, "eth_getTransactionReceipt", &p.Receipt, hash); err != nil {
			t.Fatal(err)
		}
		if p.Receipt != nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
	if err := c.Call(ctx, "eth_getTransactionByHash", &p.Transaction, hash); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(ctx, "debug_traceTransaction", &p.Trace, hash, map[string]any{"disableStack": true, "disableMemory": true, "disableStorage": true}); err != nil {
		t.Fatal(err)
	}
	logGovernanceProof(t, "mined-refusal", p)
	if err := validateMinedStakingFailure(p, w.Address(), input, value, reason); err != nil {
		t.Fatal(err)
	}
	used, ok := new(big.Int).SetString(strings.TrimPrefix(fmt.Sprint(p.Receipt["gasUsed"]), "0x"), 16)
	if !ok {
		t.Fatal("invalid gasUsed")
	}
	actualPrice, ok := new(big.Int).SetString(strings.TrimPrefix(fmt.Sprint(p.Receipt["effectiveGasPrice"]), "0x"), 16)
	if !ok {
		t.Fatal("invalid effectiveGasPrice")
	}
	after, err := c.BalanceAt(ctx, w.Address())
	if err != nil {
		t.Fatal(err)
	}
	fee := new(big.Int).Mul(used, actualPrice)
	if new(big.Int).Sub(before, fee).Cmp(after) != 0 {
		t.Fatal("rejected staking value was not refunded; fee balance mismatch")
	}
	logGovernanceProof(t, "refusal-fee", map[string]string{"hash": hash, "before": before.String(), "after": after.String(), "fee": fee.String(), "refundedValue": value.String()})
	return p
}

// logGovernanceProof retains public RPC observations in the immutable test log.
func logGovernanceProof(t *testing.T, label string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("governance-proof %s: %s", label, raw)
}

// feeReceipt verifies the status and transaction identity before event decoding.
func feeReceipt(t *testing.T, c *rpc.Client, hash string) []accounts.Log {
	t.Helper()
	raw, err := c.TxReceipt(t.Context(), hash)
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Status string         `json:"status"`
		Hash   string         `json:"transactionHash"`
		Logs   []accounts.Log `json:"logs"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "0x1" || receipt.Hash != hash {
		t.Fatal("wrong fee receipt", string(raw))
	}
	logGovernanceProof(t, "fee-receipt", json.RawMessage(raw))
	return receipt.Logs
}
