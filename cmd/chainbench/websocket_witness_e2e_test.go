//go:build e2e

package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type wsObservedPayload struct {
	ReceivedUnixNano int64
	Payload          map[string]any
}
type websocketWitness struct {
	HeadID, LogID, Contract, OtherContract, Topic, TargetHash string
	PeriodNanos, SubmittedUnixNano                            int64
	StartBlock, EndBlock                                      map[string]any
	Headers, Logs                                             []wsObservedPayload
	CanonicalBlocks                                           []map[string]any
	Receipt, OtherReceipt                                     map[string]any
	Filter                                                    map[string]any
}

func verifyWebsocketWitness(p websocketWitness) error {
	start, sok := lifecycleQuantity(p.StartBlock["number"])
	end, eok := lifecycleQuantity(p.EndBlock["number"])
	if p.HeadID == "" || p.LogID == "" || !sok || !eok || !start.IsUint64() || !end.IsUint64() || start.Uint64() <= 20 || end.Cmp(start) <= 0 || end.Uint64()-start.Uint64() < 3 || uint64(len(p.Headers)) != end.Uint64()-start.Uint64() || len(p.CanonicalBlocks) != len(p.Headers) {
		return fmt.Errorf("not exact acknowledged block window")
	}
	parent := p.StartBlock["hash"]
	if !epochBoundaryBytes(parent, 32) {
		return fmt.Errorf("missing window parent")
	}
	for i, row := range p.Headers {
		h, c := row.Payload, p.CanonicalBlocks[i]
		number := fmt.Sprintf("0x%x", start.Uint64()+1+uint64(i))
		if row.ReceivedUnixNano <= 0 || h["number"] != number || !epochBoundaryBytes(h["hash"], 32) || h["parentHash"] != parent || c["number"] != number || c["hash"] != h["hash"] || c["parentHash"] != parent {
			return fmt.Errorf("WS head missing, duplicate or not canonical")
		}
		parent = h["hash"]
	}
	if p.EndBlock["hash"] != parent {
		return fmt.Errorf("end snapshot not bound")
	}
	if p.PeriodNanos <= 0 || p.SubmittedUnixNano <= 0 || len(p.Logs) != 1 || !epochBoundaryBytes(p.Contract, 20) || !epochBoundaryBytes(p.OtherContract, 20) || strings.EqualFold(p.Contract, p.OtherContract) || !epochBoundaryBytes(p.Topic, 32) {
		return fmt.Errorf("missing exact filtered log control")
	}
	if !strings.EqualFold(fmt.Sprint(p.Filter["address"]), p.Contract) || !reflect.DeepEqual(p.Filter["topics"], []any{p.Topic}) {
		return fmt.Errorf("not actual address/topic filter")
	}
	row := p.Logs[0]
	log := row.Payload
	delay := row.ReceivedUnixNano - p.SubmittedUnixNano
	if delay <= 0 || delay >= p.PeriodNanos {
		return fmt.Errorf("submit-to-log upper bound not below one block")
	}
	if !strings.EqualFold(fmt.Sprint(log["address"]), p.Contract) || !reflect.DeepEqual(log["topics"], []any{p.Topic}) || log["transactionHash"] != p.TargetHash || log["removed"] != false || log["data"] != "0x" {
		return fmt.Errorf("received log does not match filter/transaction")
	}
	number, ok := lifecycleQuantity(log["blockNumber"])
	if !ok || !number.IsUint64() || number.Cmp(start) <= 0 || number.Cmp(end) > 0 || log["blockHash"] != p.CanonicalBlocks[number.Uint64()-start.Uint64()-1]["hash"] {
		return fmt.Errorf("log not in observed canonical window")
	}
	logs, ok := p.Receipt["logs"].([]any)
	if !ok || len(logs) != 1 || !reflect.DeepEqual(logs[0], log) || p.Receipt["transactionHash"] != p.TargetHash || p.Receipt["status"] != "0x1" || p.Receipt["blockHash"] != log["blockHash"] || p.Receipt["blockNumber"] != log["blockNumber"] {
		return fmt.Errorf("received log not bound to mined receipt")
	}
	otherLogs, ok := p.OtherReceipt["logs"].([]any)
	if !ok || len(otherLogs) != 1 || p.OtherReceipt["status"] != "0x1" || p.OtherReceipt["transactionHash"] == p.TargetHash {
		return fmt.Errorf("missing actual negative address control")
	}
	other, ok := otherLogs[0].(map[string]any)
	if !ok || !strings.EqualFold(fmt.Sprint(other["address"]), p.OtherContract) || !reflect.DeepEqual(other["topics"], []any{p.Topic}) {
		return fmt.Errorf("negative emitter does not carry the same topic")
	}
	return nil
}

func websocketWitnessFixture() websocketWitness {
	contract, other, topic, txhash := "0x"+strings.Repeat("aa", 20), "0x"+strings.Repeat("bb", 20), "0x"+strings.Repeat("11", 32), "0x"+strings.Repeat("cc", 32)
	p := websocketWitness{HeadID: "0xhead", LogID: "0xlogs", Contract: contract, OtherContract: other, Topic: topic, TargetHash: txhash, PeriodNanos: 1000000000, SubmittedUnixNano: 1000000000, StartBlock: map[string]any{"number": "0x18", "hash": fmt.Sprintf("0x%064x", 24)}, EndBlock: map[string]any{"number": "0x1c", "hash": fmt.Sprintf("0x%064x", 28)}, Filter: map[string]any{"address": contract, "topics": []any{topic}}}
	for n := 25; n <= 28; n++ {
		b := map[string]any{"number": fmt.Sprintf("0x%x", n), "hash": fmt.Sprintf("0x%064x", n), "parentHash": fmt.Sprintf("0x%064x", n-1)}
		h := map[string]any{}
		for k, v := range b {
			h[k] = v
		}
		p.Headers = append(p.Headers, wsObservedPayload{ReceivedUnixNano: int64(n) * 1000000000, Payload: h})
		p.CanonicalBlocks = append(p.CanonicalBlocks, b)
	}
	log := map[string]any{"address": contract, "topics": []any{topic}, "data": "0x", "blockNumber": "0x19", "blockHash": fmt.Sprintf("0x%064x", 25), "transactionHash": txhash, "transactionIndex": "0x0", "logIndex": "0x0", "removed": false}
	otherLog := map[string]any{}
	for k, v := range log {
		otherLog[k] = v
	}
	otherLog["address"] = other
	otherLog["transactionHash"] = "0x" + strings.Repeat("dd", 32)
	p.Logs = []wsObservedPayload{{ReceivedUnixNano: 1250000000, Payload: log}}
	p.Receipt = map[string]any{"transactionHash": txhash, "blockNumber": "0x19", "blockHash": fmt.Sprintf("0x%064x", 25), "status": "0x1", "logs": []any{log}}
	p.OtherReceipt = map[string]any{"transactionHash": otherLog["transactionHash"], "blockNumber": "0x19", "blockHash": fmt.Sprintf("0x%064x", 25), "status": "0x1", "logs": []any{otherLog}}
	return p
}

func TestWebsocketWitnessRejectsCountsWithoutPayloadOrigin(t *testing.T) {
	if err := verifyWebsocketWitness(websocketWitnessFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*websocketWitness)
	}{
		{"missing acknowledgement", func(p *websocketWitness) { p.HeadID = "" }},
		{"missing logs acknowledgement", func(p *websocketWitness) { p.LogID = "" }},
		{"missing generated head", func(p *websocketWitness) { p.Headers = p.Headers[:3] }},
		{"extra head", func(p *websocketWitness) { p.Headers = append(p.Headers, p.Headers[3]) }},
		{"duplicate head", func(p *websocketWitness) { p.Headers[1] = p.Headers[0] }},
		{"wrong number", func(p *websocketWitness) { p.Headers[1].Payload["number"] = "0xff" }},
		{"wrong hash", func(p *websocketWitness) { p.Headers[1].Payload["hash"] = "other" }},
		{"wrong parent", func(p *websocketWitness) { p.Headers[1].Payload["parentHash"] = "other" }},
		{"wrong canonical source", func(p *websocketWitness) { p.CanonicalBlocks[1]["hash"] = "other" }},
		{"missing topic filter", func(p *websocketWitness) { delete(p.Filter, "topics") }},
		{"wrong address", func(p *websocketWitness) { p.Logs[0].Payload["address"] = p.OtherContract }},
		{"wrong topic", func(p *websocketWitness) { p.Logs[0].Payload["topics"] = []any{"other"} }},
		{"wrong transaction", func(p *websocketWitness) { p.Logs[0].Payload["transactionHash"] = "other" }},
		{"one block delay", func(p *websocketWitness) { p.Logs[0].ReceivedUnixNano = p.SubmittedUnixNano + p.PeriodNanos }},
		{"before submission", func(p *websocketWitness) { p.Logs[0].ReceivedUnixNano = p.SubmittedUnixNano - 1 }},
		{"unknown block period", func(p *websocketWitness) { p.PeriodNanos = 0 }},
		{"receipt not bound", func(p *websocketWitness) { p.Receipt["transactionHash"] = "other" }},
		{"receipt failed", func(p *websocketWitness) { p.Receipt["status"] = "0x0" }},
		{"negative filter leaked", func(p *websocketWitness) {
			p.Logs = append(p.Logs, wsObservedPayload{Payload: p.OtherReceipt["logs"].([]any)[0].(map[string]any)})
		}},
		{"removed log", func(p *websocketWitness) { p.Logs[0].Payload["removed"] = true }},
		{"missing log", func(p *websocketWitness) { p.Logs = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := websocketWitnessFixture()
			tc.change(&p)
			if verifyWebsocketWitness(p) == nil {
				t.Fatal("counts hid an unbound or late WS event")
			}
		})
	}
}

func verifyWSDeployment(row map[string]any, expected string) error {
	receipt, rok := row["receipt"].(map[string]any)
	block, bok := row["block"].(map[string]any)
	tag, tok := lifecycleQuantity(row["tag"])
	if !rok || !bok || !tok || !tag.IsUint64() || tag.Sign() <= 0 || row["tag"] != receipt["blockNumber"] || block["number"] != receipt["blockNumber"] || !epochBoundaryBytes(block["hash"], 32) || block["hash"] != receipt["blockHash"] || receipt["status"] != "0x1" || !epochBoundaryBytes(row["address"], 20) || !strings.EqualFold(fmt.Sprint(row["address"]), fmt.Sprint(receipt["contractAddress"])) || expected == "0x" || row["code"] != expected {
		return fmt.Errorf("runtime code not bound to canonical deployment receipt")
	}
	return nil
}

func TestWSDeploymentRequiresReceiptPinnedCode(t *testing.T) {
	fixture := func() map[string]any {
		return map[string]any{"address": "0x" + strings.Repeat("aa", 20), "tag": "0x19", "code": "0x6000", "receipt": map[string]any{"contractAddress": "0x" + strings.Repeat("aa", 20), "blockNumber": "0x19", "blockHash": "0x" + strings.Repeat("11", 32), "status": "0x1"}, "block": map[string]any{"number": "0x19", "hash": "0x" + strings.Repeat("11", 32)}}
	}
	if err := verifyWSDeployment(fixture(), "0x6000"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"latest tag", func(p map[string]any) { p["tag"] = "latest" }},
		{"different canonical block", func(p map[string]any) { p["block"].(map[string]any)["hash"] = "other" }},
		{"different deployment address", func(p map[string]any) { p["address"] = "other" }},
		{"failed receipt", func(p map[string]any) { p["receipt"].(map[string]any)["status"] = "0x0" }},
		{"empty code", func(p map[string]any) { p["code"] = "0x" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := fixture()
			tc.change(p)
			if verifyWSDeployment(p, "0x6000") == nil {
				t.Fatal("unbound deployment code observation accepted")
			}
		})
	}
}
