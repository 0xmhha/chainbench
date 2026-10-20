//go:build e2e

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"testing"
)

const crossSyncRuntimeCode = "0x60003560005500"
const crossSyncInitCode = "0x6007600c60003960076000f360003560005500"

type crossSyncTx struct {
	Phase, Kind, Raw, Hash      string
	Block, Transaction, Receipt map[string]any
}
type crossSyncReplay struct {
	Hash                 string
	Transaction, Receipt map[string]any
}
type crossSyncState struct {
	Tag                              string
	Balances, Nonces, Codes, Storage map[string]string
}
type crossSyncReader struct {
	Mode, Bootnode, PeerCount, Syncing                                 string
	Fork, StartHeight, SourceStart, Reference, Head, SourceHead, Pivot uint64
	Progress                                                           []uint64
	GenesisHash, SourceGenesisHash                                     string
	SourceFork, AtFork, SourceRef, AtRef                               map[string]any
	SourceState, State                                                 crossSyncState
	Transactions                                                       []crossSyncReplay
}
type crossSyncProof struct {
	Sender, Payer, Recipient, ContractA, ContractB, Funding string
	Dataset                                                 []crossSyncTx
	Readers                                                 []crossSyncReader
}

func verifyCrossSync(p crossSyncProof) error {
	if len(p.Dataset) != 10 || len(p.Readers) != 2 || p.Funding != "1000000000000000000000000000" {
		return fmt.Errorf("missing exact ten transactions/readers/funding")
	}
	seen := map[string]bool{}
	for _, a := range []string{p.Sender, p.Payer, p.Recipient, p.ContractA, p.ContractB} {
		a = strings.ToLower(a)
		if !epochBoundaryBytes(a, 20) || seen[a] {
			return fmt.Errorf("invalid independent actor/contracts")
		}
		seen[a] = true
	}
	senderFee, payerFee := big.NewInt(0), big.NewInt(0)
	for i, v := range p.Dataset {
		kind := []string{"legacy", "dynamic", "feeDelegation", "deploy", "write"}[i%5]
		phase, contract := "pre", p.ContractA
		if i >= 5 {
			phase, contract = "post", p.ContractB
		}
		typ := map[string]string{"legacy": "0x0", "dynamic": "0x2", "feeDelegation": "0x16", "deploy": "0x2", "write": "0x2"}[kind]
		raw, err := hex.DecodeString(strings.TrimPrefix(v.Raw, "0x"))
		if err != nil || len(raw) < 2 || v.Hash != rawSubmissionHash(raw) || seen[v.Hash] {
			return fmt.Errorf("missing/duplicate exact raw hash")
		}
		seen[v.Hash] = true
		if (typ == "0x0" && raw[0] < 0xc0) || (typ == "0x2" && raw[0] != 2) || (typ == "0x16" && raw[0] != 0x16) {
			return fmt.Errorf("wrong raw transaction envelope")
		}
		t, r, b := v.Transaction, v.Receipt, v.Block
		h, ok := lifecycleQuantity(t["blockNumber"])
		if !ok || !h.IsUint64() || h.Sign() <= 0 || (i < 5 && h.Uint64() >= 100) || (i >= 5 && h.Uint64() < 100) || v.Phase != phase || v.Kind != kind {
			return fmt.Errorf("missing original pre/post type matrix")
		}
		if t["hash"] != v.Hash || r["transactionHash"] != v.Hash || t["type"] != typ || r["type"] != typ || !strings.EqualFold(fmt.Sprint(t["from"]), p.Sender) || !strings.EqualFold(fmt.Sprint(r["from"]), p.Sender) || t["nonce"] != hexUint(uint64(i)) || r["status"] != "0x1" || b["hash"] != t["blockHash"] || b["hash"] != r["blockHash"] || b["number"] != t["blockNumber"] || b["number"] != r["blockNumber"] || t["transactionIndex"] != r["transactionIndex"] || !epochBoundaryBytes(b["hash"], 32) {
			return fmt.Errorf("wrong source mined transaction or receipt")
		}
		index, ok := lifecycleQuantity(r["transactionIndex"])
		if !ok || !index.IsUint64() {
			return fmt.Errorf("missing source inclusion index")
		}
		txs, ok := b["transactions"].([]any)
		if !ok || index.Uint64() >= uint64(len(txs)) || txs[index.Uint64()] != v.Hash {
			return fmt.Errorf("not exact source block inclusion")
		}
		occurrences := 0
		for _, hash := range txs {
			if hash == v.Hash {
				occurrences++
			}
		}
		if occurrences != 1 {
			return fmt.Errorf("duplicate source inclusion")
		}
		amount := big.NewInt(0)
		if i%5 < 3 {
			n := int64(1 + i%5)
			if i >= 5 {
				n += 3
			}
			amount.Mul(big.NewInt(n), big.NewInt(1000000000000000000))
			if !strings.EqualFold(fmt.Sprint(t["to"]), p.Recipient) || t["input"] != "0x" {
				return fmt.Errorf("wrong transfer destination/data")
			}
		} else if kind == "deploy" {
			if t["to"] != nil || t["input"] != crossSyncInitCode || !strings.EqualFold(fmt.Sprint(r["contractAddress"]), contract) {
				return fmt.Errorf("wrong actual deployment")
			}
		} else {
			n := 42
			if i >= 5 {
				n = 84
			}
			if !strings.EqualFold(fmt.Sprint(t["to"]), contract) || t["input"] != "0x"+fmt.Sprintf("%064x", n) {
				return fmt.Errorf("wrong exact storage write")
			}
		}
		value, ok := lifecycleQuantity(t["value"])
		if !ok || value.Cmp(amount) != 0 || (kind == "feeDelegation" && !strings.EqualFold(fmt.Sprint(t["feePayer"]), p.Payer)) {
			return fmt.Errorf("wrong value or fee payer")
		}
		gas, ok := lifecycleQuantity(r["gasUsed"])
		if !ok || gas.Sign() <= 0 {
			return fmt.Errorf("missing actual gas use")
		}
		price, ok := lifecycleQuantity(r["effectiveGasPrice"])
		if !ok || price.Sign() < 0 {
			return fmt.Errorf("missing actual gas price")
		}
		fee := new(big.Int).Mul(gas, price)
		if kind == "feeDelegation" {
			payerFee.Add(payerFee, fee)
		} else {
			senderFee.Add(senderFee, fee)
		}
	}
	funding, _ := new(big.Int).SetString(p.Funding, 10)
	value := new(big.Int).Mul(big.NewInt(21), big.NewInt(1000000000000000000))
	balances := map[string]*big.Int{p.Sender: new(big.Int).Sub(new(big.Int).Sub(new(big.Int).Set(funding), value), senderFee), p.Payer: new(big.Int).Sub(new(big.Int).Set(funding), payerFee), p.Recipient: value}
	for i, r := range p.Readers {
		mode := []string{"full", "snap"}[i]
		peer, err := url.Parse(r.Bootnode)
		if r.Mode != mode || r.Fork != 100 || r.StartHeight != 0 || r.SourceStart < 200 || r.Reference < 200 || r.Head < r.Reference || r.SourceHead < r.Reference || r.Head > r.SourceHead+2 || r.SourceHead > r.Head+2 || r.Syncing != "false" || len(r.Progress) < 2 || r.Progress[0] != 0 || r.Progress[len(r.Progress)-1] != r.Head || !epochBoundaryBytes(r.GenesisHash, 32) || r.GenesisHash != r.SourceGenesisHash || err != nil || peer.Scheme != "enode" || peer.User == nil || peer.Hostname() != "127.0.0.1" || peer.Port() == "" {
			return fmt.Errorf("not fresh connected completed original sync mode")
		}
		peers, ok := lifecycleQuantity(r.PeerCount)
		if !ok || peers.Sign() <= 0 {
			return fmt.Errorf("peer connection absent")
		}
		for j, h := range r.Progress {
			if j > 0 && h < r.Progress[j-1] {
				return fmt.Errorf("reader progress decreased")
			}
		}
		if (mode == "full" && r.Pivot != 0) || (mode == "snap" && (r.Pivot <= 100 || r.Pivot > r.Reference)) {
			return fmt.Errorf("actual snap pivot not beyond fork and before state query")
		}
		for _, pair := range []struct {
			source, reader map[string]any
			height         uint64
		}{{r.SourceFork, r.AtFork, 100}, {r.SourceRef, r.AtRef, r.Reference}} {
			for _, b := range []map[string]any{pair.source, pair.reader} {
				if b["number"] != hexUint(pair.height) || !epochBoundaryBytes(b["hash"], 32) || !epochBoundaryBytes(b["stateRoot"], 32) {
					return fmt.Errorf("missing exact canonical comparison block")
				}
			}
			if !crossSyncFieldsEqual(pair.source, pair.reader, []string{"number", "hash", "stateRoot", "parentHash", "extraData", "transactions"}) {
				return fmt.Errorf("fork/reference header or root mismatch")
			}
		}
		if r.SourceState.Tag != hexUint(r.Reference) || r.State.Tag != r.SourceState.Tag || len(r.SourceState.Balances) != 3 || len(r.SourceState.Nonces) != 2 || len(r.SourceState.Codes) != 2 || len(r.SourceState.Storage) != 2 {
			return fmt.Errorf("missing pinned state observations")
		}
		for addr, want := range balances {
			got, ok := lifecycleQuantity(r.SourceState.Balances[addr])
			if !ok || got.Cmp(want) != 0 {
				return fmt.Errorf("source value/fee-payer ledger differs")
			}
		}
		if r.SourceState.Nonces[p.Sender] != "0xa" || r.SourceState.Nonces[p.Payer] != "0x0" || r.SourceState.Codes[p.ContractA] != crossSyncRuntimeCode || r.SourceState.Codes[p.ContractB] != crossSyncRuntimeCode || r.SourceState.Storage[p.ContractA] != "0x"+fmt.Sprintf("%064x", 42) || r.SourceState.Storage[p.ContractB] != "0x"+fmt.Sprintf("%064x", 84) {
			return fmt.Errorf("source deploy/write/nonce matrix not actually present")
		}
		a, _ := json.Marshal(r.SourceState)
		b, _ := json.Marshal(r.State)
		if string(a) != string(b) {
			return fmt.Errorf("reader cannot reproduce exact pinned state")
		}
		if len(r.Transactions) != 10 {
			return fmt.Errorf("missing replayed transactions/receipts")
		}
		for j, got := range r.Transactions {
			v := p.Dataset[j]
			if got.Hash != v.Hash || !crossSyncFieldsEqual(v.Transaction, got.Transaction, []string{"hash", "blockHash", "blockNumber", "transactionIndex", "from", "to", "nonce", "type", "value", "input", "gas", "maxFeePerGas", "maxPriorityFeePerGas", "accessList", "feePayer", "v", "r", "s"}) || !crossSyncFieldsEqual(v.Receipt, got.Receipt, []string{"transactionHash", "blockHash", "blockNumber", "transactionIndex", "status", "type", "from", "to", "contractAddress", "gasUsed", "cumulativeGasUsed", "effectiveGasPrice", "logs", "logsBloom"}) {
				return fmt.Errorf("replayed transaction/receipt differs from source")
			}
		}
	}
	return nil
}

func crossSyncFieldsEqual(source, reader map[string]any, fields []string) bool {
	for _, k := range fields {
		a, exists := source[k]
		if !exists {
			continue
		}
		b, present := reader[k]
		if !present {
			return false
		}
		aa, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		if !strings.EqualFold(string(aa), string(bb)) {
			return false
		}
	}
	return true
}

func cloneCrossSync[T any](v T) T {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out T
	if err = json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}
func crossSyncFixture() crossSyncProof {
	p := crossSyncProof{Sender: fmt.Sprintf("0x%040x", 1), Payer: fmt.Sprintf("0x%040x", 2), Recipient: fmt.Sprintf("0x%040x", 3), ContractA: fmt.Sprintf("0x%040x", 4), ContractB: fmt.Sprintf("0x%040x", 5), Funding: "1000000000000000000000000000"}
	senderFee, payerFee := big.NewInt(0), big.NewInt(0)
	for i := 0; i < 10; i++ {
		kind := []string{"legacy", "dynamic", "feeDelegation", "deploy", "write"}[i%5]
		phase := "pre"
		h := uint64(20 + i)
		contract := p.ContractA
		if i >= 5 {
			phase = "post"
			h = uint64(110 + i - 5)
			contract = p.ContractB
		}
		typ, prefix := "0x2", "02"
		if kind == "legacy" {
			typ, prefix = "0x0", "f8"
		}
		if kind == "feeDelegation" {
			typ, prefix = "0x16", "16"
		}
		raw := fmt.Sprintf("0x%s00%02x", prefix, i)
		b, _ := hex.DecodeString(raw[2:])
		hash := rawSubmissionHash(b)
		bh := fmt.Sprintf("0x%064x", 2000+i)
		value := big.NewInt(0)
		input := "0x"
		var to any = p.Recipient
		var created any
		gas := int64(21000)
		if i%5 < 3 {
			v := int64(1 + i%5)
			if i >= 5 {
				v += 3
			}
			value.Mul(big.NewInt(v), big.NewInt(1000000000000000000))
		}
		if kind == "deploy" {
			to = nil
			input = crossSyncInitCode
			created = contract
			gas = 60000
		}
		if kind == "write" {
			to = contract
			v := 42
			if i >= 5 {
				v = 84
			}
			input = "0x" + fmt.Sprintf("%064x", v)
			gas = 45000
		}
		price := big.NewInt(100000000000)
		fee := new(big.Int).Mul(big.NewInt(gas), price)
		if kind == "feeDelegation" {
			payerFee.Add(payerFee, fee)
		} else {
			senderFee.Add(senderFee, fee)
		}
		tx := map[string]any{"hash": hash, "type": typ, "from": p.Sender, "to": to, "value": "0x" + value.Text(16), "input": input, "nonce": hexUint(uint64(i)), "blockHash": bh, "blockNumber": hexUint(h), "transactionIndex": "0x0", "gas": hexUint(uint64(gas)), "gasPrice": "0x" + price.Text(16)}
		if kind == "feeDelegation" {
			tx["feePayer"] = p.Payer
		}
		rc := map[string]any{"transactionHash": hash, "type": typ, "from": p.Sender, "to": to, "contractAddress": created, "status": "0x1", "blockHash": bh, "blockNumber": hexUint(h), "transactionIndex": "0x0", "gasUsed": hexUint(uint64(gas)), "cumulativeGasUsed": hexUint(uint64(gas)), "effectiveGasPrice": "0x" + price.Text(16), "logs": []any{}}
		p.Dataset = append(p.Dataset, crossSyncTx{Phase: phase, Kind: kind, Raw: raw, Hash: hash, Block: map[string]any{"hash": bh, "number": hexUint(h), "transactions": []any{hash}}, Transaction: tx, Receipt: rc})
	}
	funding, _ := new(big.Int).SetString(p.Funding, 10)
	senderAfter := new(big.Int).Sub(new(big.Int).Sub(new(big.Int).Set(funding), big.NewInt(0).Mul(big.NewInt(21), big.NewInt(1000000000000000000))), senderFee)
	payerAfter := new(big.Int).Sub(new(big.Int).Set(funding), payerFee)
	state := crossSyncState{Tag: "0xd2", Balances: map[string]string{p.Sender: "0x" + senderAfter.Text(16), p.Payer: "0x" + payerAfter.Text(16), p.Recipient: "0x" + new(big.Int).Mul(big.NewInt(21), big.NewInt(1000000000000000000)).Text(16)}, Nonces: map[string]string{p.Sender: "0xa", p.Payer: "0x0"}, Codes: map[string]string{p.ContractA: crossSyncRuntimeCode, p.ContractB: crossSyncRuntimeCode}, Storage: map[string]string{p.ContractA: "0x" + fmt.Sprintf("%064x", 42), p.ContractB: "0x" + fmt.Sprintf("%064x", 84)}}
	fork := map[string]any{"number": "0x64", "hash": "0x" + strings.Repeat("ab", 32), "stateRoot": "0x" + strings.Repeat("bc", 32), "extraData": "0x" + strings.Repeat("cd", 40)}
	ref := map[string]any{"number": "0xd2", "hash": "0x" + strings.Repeat("de", 32), "stateRoot": "0x" + strings.Repeat("ef", 32), "extraData": "0x" + strings.Repeat("fa", 40)}
	for _, mode := range []string{"full", "snap"} {
		r := crossSyncReader{Mode: mode, Bootnode: "enode://" + strings.Repeat("11", 64) + "@127.0.0.1:31000", PeerCount: "0x1", Syncing: "false", Fork: 100, StartHeight: 0, SourceStart: 210, Reference: 210, Head: 212, SourceHead: 213, Progress: []uint64{0, 10, 99, 100, 212}, GenesisHash: "0x" + strings.Repeat("ca", 32), SourceGenesisHash: "0x" + strings.Repeat("ca", 32), SourceFork: cloneCrossSync(fork), AtFork: cloneCrossSync(fork), SourceRef: cloneCrossSync(ref), AtRef: cloneCrossSync(ref), SourceState: cloneCrossSync(state), State: cloneCrossSync(state)}
		if mode == "snap" {
			r.Pivot = 146
		}
		for _, tx := range p.Dataset {
			r.Transactions = append(r.Transactions, crossSyncReplay{Hash: tx.Hash, Transaction: cloneCrossSync(tx.Transaction), Receipt: cloneCrossSync(tx.Receipt)})
		}
		p.Readers = append(p.Readers, r)
	}
	return p
}

func TestCrossForkSyncRejectsHeadAndHashOnly(t *testing.T) {
	if err := verifyCrossSync(crossSyncFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*crossSyncProof)
	}{
		{"missing fifth prefork type", func(p *crossSyncProof) { p.Dataset = append(p.Dataset[:4], p.Dataset[5:]...) }},
		{"wrong phase boundary", func(p *crossSyncProof) { p.Dataset[0].Transaction["blockNumber"] = "0x65" }},
		{"wrong prefork FD type", func(p *crossSyncProof) { p.Dataset[2].Transaction["type"] = "0x2" }},
		{"wrong raw hash", func(p *crossSyncProof) { p.Dataset[0].Raw = "0xf800ff" }},
		{"duplicate source nonce", func(p *crossSyncProof) { p.Dataset[1].Transaction["nonce"] = "0x0" }},
		{"failed source receipt", func(p *crossSyncProof) { p.Dataset[0].Receipt["status"] = "0x0" }},
		{"wrong source inclusion", func(p *crossSyncProof) { p.Dataset[0].Block["transactions"] = []any{} }},
		{"reused warm reader", func(p *crossSyncProof) { p.Readers[0].StartHeight = 99 }},
		{"source below200", func(p *crossSyncProof) { p.Readers[0].SourceStart = 199 }},
		{"wrong requested mode", func(p *crossSyncProof) { p.Readers[1].Mode = "full" }},
		{"no bootnode", func(p *crossSyncProof) { p.Readers[0].Bootnode = "" }},
		{"no peer", func(p *crossSyncProof) { p.Readers[0].PeerCount = "0x0" }},
		{"syncing unknown", func(p *crossSyncProof) { p.Readers[0].Syncing = "" }},
		{"syncing not complete", func(p *crossSyncProof) { p.Readers[0].Syncing = "{}" }},
		{"decreasing progress", func(p *crossSyncProof) { p.Readers[0].Progress = []uint64{0, 100, 90, 212} }},
		{"far behind source", func(p *crossSyncProof) { p.Readers[0].SourceHead = 300 }},
		{"wrong genesis", func(p *crossSyncProof) { p.Readers[0].GenesisHash = "0x" + strings.Repeat("bb", 32) }},
		{"wrong fork height", func(p *crossSyncProof) { p.Readers[0].Fork = 20 }},
		{"fork hash differs", func(p *crossSyncProof) { p.Readers[0].AtFork["hash"] = "other" }},
		{"state root differs", func(p *crossSyncProof) { p.Readers[0].AtRef["stateRoot"] = "other" }},
		{"snapshot latest not pinned", func(p *crossSyncProof) { p.Readers[1].State.Tag = "latest" }},
		{"snap pivot absent", func(p *crossSyncProof) { p.Readers[1].Pivot = 0 }},
		{"snap pivot beforefork", func(p *crossSyncProof) { p.Readers[1].Pivot = 99 }},
		{"snap reference before pivot", func(p *crossSyncProof) { p.Readers[1].Pivot = 211 }},
		{"missing replay receipt", func(p *crossSyncProof) { p.Readers[0].Transactions = p.Readers[0].Transactions[:9] }},
		{"wrong replay transaction", func(p *crossSyncProof) { p.Readers[0].Transactions[0].Transaction["value"] = "0x1" }},
		{"wrong replay receipt", func(p *crossSyncProof) { p.Readers[1].Transactions[2].Receipt["gasUsed"] = "0x0" }},
		{"wrong replay code", func(p *crossSyncProof) { p.Readers[1].State.Codes[p.ContractA] = "0x" }},
		{"wrong replay storage", func(p *crossSyncProof) { p.Readers[0].State.Storage[p.ContractB] = "0x0" }},
		{"wrong replay nonce", func(p *crossSyncProof) { p.Readers[0].State.Nonces[p.Sender] = "0x9" }},
		{"wrong replay balance", func(p *crossSyncProof) { p.Readers[1].State.Balances[p.Recipient] = "0x0" }},
		{"wrong source contract state", func(p *crossSyncProof) {
			for i := range p.Readers {
				p.Readers[i].SourceState.Storage[p.ContractA] = "0x0"
				p.Readers[i].State.Storage[p.ContractA] = "0x0"
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := crossSyncFixture()
			tc.change(&p)
			if verifyCrossSync(p) == nil {
				t.Fatal("head/hash hid missing original cross-fork replay and state")
			}
		})
	}
}
