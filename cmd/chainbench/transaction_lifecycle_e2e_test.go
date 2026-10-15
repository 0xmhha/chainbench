//go:build e2e

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

type lifecycleControl struct {
	Node   int
	Method string
	Mining bool
}
type lifecyclePending struct {
	Node              int
	Transaction, Pool map[string]any
}
type transactionLifecycleProof struct {
	Raw, Hash, Sender, Recipient                               string
	HeldHeight, ConfirmedHeight                                uint64
	StableMillis                                               int64
	StableHeads                                                [2][]uint64
	Paused, Resumed                                            []lifecycleControl
	Pending                                                    []lifecyclePending
	HeldBlock, Transaction, Receipt, MinedBlock                map[string]any
	SenderBefore, SenderAfter, RecipientBefore, RecipientAfter string
	NonceBefore, NonceAfter                                    uint64
}

func verifyTransactionLifecycle(p transactionLifecycleProof) error {
	if p.Receipt["status"] != "0x1" || p.Receipt["transactionHash"] != p.Hash || p.HeldHeight <= 20 || p.StableMillis < 2000 || p.NonceBefore != 0 || p.NonceAfter != 1 {
		return fmt.Errorf("not complete successful post-fork lifecycle")
	}
	if !strings.HasPrefix(p.Raw, "0x02") {
		return fmt.Errorf("not local signed dynamic transaction")
	}
	raw, err := hex.DecodeString(p.Raw[2:])
	if err != nil || len(raw) < 2 || rawSubmissionHash(raw) != p.Hash {
		return fmt.Errorf("raw/hash mismatch")
	}
	for i := 0; i < 2; i++ {
		if len(p.StableHeads[i]) != 4 {
			return fmt.Errorf("missing stable endpoint heads")
		}
		for _, h := range p.StableHeads[i] {
			if h != p.HeldHeight {
				return fmt.Errorf("production was not held")
			}
		}
	}
	for phase, controls := range [][]lifecycleControl{p.Paused, p.Resumed} {
		if len(controls) != 4 {
			return fmt.Errorf("missing owned miner control")
		}
		for i, c := range controls {
			method := "miner_stop"
			if phase == 1 {
				method = "miner_start"
			}
			if c.Node != i+1 || c.Method != method || c.Mining != (phase == 1) {
				return fmt.Errorf("wrong miner control/result")
			}
		}
	}
	quantityEqual := func(v any, want int64) bool { n, ok := lifecycleQuantity(v); return ok && n.Cmp(big.NewInt(want)) == 0 }
	checkTx := func(tx map[string]any, pending bool) bool {
		if tx["hash"] != p.Hash || !strings.EqualFold(fmt.Sprint(tx["from"]), p.Sender) || !strings.EqualFold(fmt.Sprint(tx["to"]), p.Recipient) || tx["type"] != "0x2" || !quantityEqual(tx["value"], 1000000000000000000) || !quantityEqual(tx["gas"], 21000) || !quantityEqual(tx["nonce"], 0) {
			return false
		}
		if pending {
			h, hp := tx["blockHash"]
			n, np := tx["blockNumber"]
			return hp && np && h == nil && n == nil
		}
		return tx["blockHash"] == p.Receipt["blockHash"] && tx["blockNumber"] == p.Receipt["blockNumber"]
	}
	if len(p.Pending) != 4 {
		return fmt.Errorf("missing unmined propagation endpoints")
	}
	for i, row := range p.Pending {
		if row.Node != i+1 || !checkTx(row.Transaction, true) {
			return fmt.Errorf("not same pending transaction before mining")
		}
		pending, ok := row.Pool["pending"].(map[string]any)
		if !ok || len(pending) != 1 {
			return fmt.Errorf("transaction not in executable pending pool")
		}
		tx, ok := pending["0"].(map[string]any)
		if !ok || !checkTx(tx, true) {
			return fmt.Errorf("pending pool origin differs")
		}
	}
	if !checkTx(p.Transaction, false) || p.Receipt["type"] != "0x2" || !quantityEqual(p.Receipt["gasUsed"], 21000) {
		return fmt.Errorf("mined normal transfer differs")
	}
	number, ok := lifecycleQuantity(p.Receipt["blockNumber"])
	if !ok || !number.IsUint64() || number.Uint64() != p.HeldHeight+1 || p.ConfirmedHeight < number.Uint64()+3 {
		return fmt.Errorf("not first resumed block and subsequent confirmation")
	}
	if p.HeldBlock["number"] != fmt.Sprintf("0x%x", p.HeldHeight) || p.MinedBlock["number"] != p.Receipt["blockNumber"] || p.MinedBlock["hash"] != p.Receipt["blockHash"] || p.MinedBlock["parentHash"] != p.HeldBlock["hash"] {
		return fmt.Errorf("canonical block origin differs")
	}
	hashes, ok := p.MinedBlock["transactions"].([]any)
	index, iok := lifecycleQuantity(p.Receipt["transactionIndex"])
	if !ok || !iok || !index.IsUint64() || index.Uint64() >= uint64(len(hashes)) || hashes[index.Uint64()] != p.Hash {
		return fmt.Errorf("not receipt-indexed block inclusion")
	}
	count := 0
	for _, h := range hashes {
		if h == p.Hash {
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("not unique block inclusion")
	}
	base, bok := lifecycleQuantity(p.MinedBlock["baseFeePerGas"])
	cap, cok := lifecycleQuantity(p.Transaction["maxFeePerGas"])
	tip, tok := lifecycleQuantity(p.Transaction["maxPriorityFeePerGas"])
	effective, eok := lifecycleQuantity(p.Receipt["effectiveGasPrice"])
	if !bok || !cok || !tok || !eok || cap.Cmp(base) < 0 {
		return fmt.Errorf("missing actual fee model")
	}
	available := new(big.Int).Sub(cap, base)
	if tip.Cmp(available) > 0 {
		tip = available
	}
	if effective.Cmp(new(big.Int).Add(base, tip)) != 0 {
		return fmt.Errorf("not base plus capped tip")
	}
	amount := big.NewInt(1000000000000000000)
	fee := new(big.Int).Mul(big.NewInt(21000), effective)
	balances := []string{p.SenderBefore, p.SenderAfter, p.RecipientBefore, p.RecipientAfter}
	values := []*big.Int{}
	for _, s := range balances {
		n, ok := new(big.Int).SetString(s, 10)
		if !ok || n.Sign() < 0 {
			return fmt.Errorf("invalid balance ledger")
		}
		values = append(values, n)
	}
	if values[0].Sign() == 0 || new(big.Int).Sub(values[0], values[1]).Cmp(new(big.Int).Add(amount, fee)) != 0 || new(big.Int).Sub(values[3], values[2]).Cmp(amount) != 0 {
		return fmt.Errorf("normal transfer value/fee ledger differs")
	}
	return nil
}

func lifecycleQuantity(v any) (*big.Int, bool) {
	s, ok := v.(string)
	if !ok || !strings.HasPrefix(s, "0x") {
		return nil, false
	}
	n, ok := new(big.Int).SetString(s[2:], 16)
	return n, ok && n.Sign() >= 0
}

func transactionLifecycleFixture() transactionLifecycleProof {
	value := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	initial := new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil)
	raw := []byte{2, 1, 2, 3}
	hash := rawSubmissionHash(raw)
	blockHash := "0x" + strings.Repeat("11", 32)
	sender, receiver := "0x"+strings.Repeat("22", 20), "0x"+strings.Repeat("33", 20)
	tx := map[string]any{"hash": hash, "from": sender, "to": receiver, "value": "0x" + value.Text(16), "gas": "0x5208", "nonce": "0x0", "type": "0x2", "maxFeePerGas": "0x96", "maxPriorityFeePerGas": "0x5a", "blockHash": blockHash, "blockNumber": "0x19"}
	receipt := map[string]any{"transactionHash": hash, "from": sender, "to": receiver, "status": "0x1", "type": "0x2", "blockHash": blockHash, "blockNumber": "0x19", "transactionIndex": "0x0", "gasUsed": "0x5208", "effectiveGasPrice": "0x64"}
	p := transactionLifecycleProof{Raw: "0x" + hex.EncodeToString(raw), Hash: hash, Sender: sender, Recipient: receiver, HeldHeight: 24, ConfirmedHeight: 28, StableMillis: 2000, StableHeads: [2][]uint64{{24, 24, 24, 24}, {24, 24, 24, 24}}, HeldBlock: map[string]any{"number": "0x18", "hash": "0x" + strings.Repeat("44", 32)}, Transaction: tx, Receipt: receipt, MinedBlock: map[string]any{"number": "0x19", "hash": blockHash, "parentHash": "0x" + strings.Repeat("44", 32), "baseFeePerGas": "0xa", "transactions": []any{hash}}, SenderBefore: initial.String(), SenderAfter: new(big.Int).Sub(new(big.Int).Sub(initial, value), big.NewInt(2100000)).String(), RecipientBefore: "0", RecipientAfter: value.String(), NonceAfter: 1}
	for i := 1; i <= 4; i++ {
		pending := map[string]any{}
		for k, v := range tx {
			pending[k] = v
		}
		pending["blockHash"], pending["blockNumber"] = nil, nil
		p.Pending = append(p.Pending, lifecyclePending{Node: i, Transaction: pending, Pool: map[string]any{"pending": map[string]any{"0": pending}, "queued": map[string]any{}}})
		p.Paused = append(p.Paused, lifecycleControl{Node: i, Method: "miner_stop", Mining: false})
		p.Resumed = append(p.Resumed, lifecycleControl{Node: i, Method: "miner_start", Mining: true})
	}
	return p
}

func TestTransactionLifecycleRejectsMinedOnlyEvidence(t *testing.T) {
	if err := verifyTransactionLifecycle(transactionLifecycleFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*transactionLifecycleProof)
	}{
		{"bad receipt status", func(p *transactionLifecycleProof) { p.Receipt["status"] = "0x0" }},
		{"mined before pending", func(p *transactionLifecycleProof) { p.Pending[1].Transaction["blockNumber"] = "0x19" }},
		{"wrong relay hash", func(p *transactionLifecycleProof) { p.Pending[1].Transaction["hash"] = "unrelated" }},
		{"missing peer pending", func(p *transactionLifecycleProof) { p.Pending = p.Pending[:3] }},
		{"pool not pending", func(p *transactionLifecycleProof) { p.Pending[1].Pool["pending"] = map[string]any{} }},
		{"wrong pending nonce", func(p *transactionLifecycleProof) { p.Pending[1].Transaction["nonce"] = "0x2" }},
		{"wrong gas", func(p *transactionLifecycleProof) { p.Transaction["gas"] = "0x10000" }},
		{"wrong value", func(p *transactionLifecycleProof) { p.Transaction["value"] = "0x1" }},
		{"missing pause", func(p *transactionLifecycleProof) { p.Paused = p.Paused[:3] }},
		{"still mining", func(p *transactionLifecycleProof) { p.Paused[1].Mining = true }},
		{"wrong resume API", func(p *transactionLifecycleProof) { p.Resumed[1].Method = "eth_mining" }},
		{"no stable height", func(p *transactionLifecycleProof) { p.StableHeads[1][1] = 25 }},
		{"missing stable duration", func(p *transactionLifecycleProof) { p.StableMillis = 0 }},
		{"empty raw", func(p *transactionLifecycleProof) { p.Raw = "0x" }},
		{"wrong raw hash", func(p *transactionLifecycleProof) { p.Hash = "0x" + strings.Repeat("ff", 32) }},
		{"not in block", func(p *transactionLifecycleProof) { p.MinedBlock["transactions"] = []any{} }},
		{"wrong receipt block", func(p *transactionLifecycleProof) { p.Receipt["blockHash"] = "unrelated" }},
		{"wrong transaction index", func(p *transactionLifecycleProof) { p.Receipt["transactionIndex"] = "0x1" }},
		{"no confirmation", func(p *transactionLifecycleProof) { p.ConfirmedHeight = 25 }},
		{"sender fee ignored", func(p *transactionLifecycleProof) {
			p.SenderAfter = new(big.Int).Sub(mustLifecycleDecimal(p.SenderBefore), mustLifecycleDecimal(p.RecipientAfter)).String()
		}},
		{"receiver not funded", func(p *transactionLifecycleProof) { p.RecipientAfter = "0" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := transactionLifecycleFixture()
			tc.change(&p)
			err := verifyTransactionLifecycle(p)
			if err == nil {
				t.Fatal("mined status hid missing transaction lifecycle evidence")
			}
		})
	}
}

func mustLifecycleDecimal(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid fixture decimal")
	}
	return n
}
