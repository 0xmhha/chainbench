package testengine_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type regularEGPWitness struct {
	Sender        string         `json:"sender"`
	Authorization string         `json:"authorization"`
	Transaction   map[string]any `json:"transaction"`
	BPReceipt     map[string]any `json:"bpReceipt"`
	SnapReceipt   map[string]any `json:"snapReceipt"`
	Block         map[string]any `json:"block"`
	Extra         map[string]any `json:"extra"`
	SnapBlock     map[string]any `json:"snapBlock"`
	ReceiptMetric uint64         `json:"receiptMetric"`
}

func regularEGPQuantity(m map[string]any, key string) (*big.Int, error) {
	s, ok := m[key].(string)
	if !ok {
		return nil, fmt.Errorf("missing quantity %s", key)
	}
	base := 16
	digits := strings.TrimPrefix(s, "0x")
	if !strings.HasPrefix(s, "0x") {
		if key != "gasTip" {
			return nil, fmt.Errorf("missing quantity %s", key)
		}
		base = 10
		digits = s
	}
	if digits == "" {
		return nil, fmt.Errorf("empty quantity %s", key)
	}
	for _, r := range digits {
		decimalDigit := r >= '0' && r <= '9'
		hexLetter := (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		if !decimalDigit && (base != 16 || !hexLetter) {
			return nil, fmt.Errorf("invalid quantity %s", key)
		}
	}
	n, ok := new(big.Int).SetString(digits, base)
	if !ok || n.Sign() < 0 {
		return nil, fmt.Errorf("invalid quantity %s", key)
	}
	return n, nil
}

func verifyRegularEGPWitness(w regularEGPWitness) (map[string]string, error) {
	if w.Authorization != "0x"+strings.Repeat("0", 64) || len(w.Sender) != 42 || !strings.EqualFold(fmt.Sprint(w.Transaction["from"]), w.Sender) {
		return nil, fmt.Errorf("regular sender not established")
	}
	hash, ok := w.Transaction["hash"].(string)
	if !ok || len(hash) != 66 || !strings.HasPrefix(hash, "0x") {
		return nil, fmt.Errorf("missing transaction identity")
	}
	for _, receipt := range []map[string]any{w.BPReceipt, w.SnapReceipt} {
		if receipt["transactionHash"] != hash || receipt["blockHash"] != w.Transaction["blockHash"] || receipt["blockNumber"] != w.Transaction["blockNumber"] || receipt["status"] != "0x1" || receipt["type"] != "0x2" {
			return nil, fmt.Errorf("unbound receipt")
		}
	}
	for _, block := range []map[string]any{w.Block, w.SnapBlock} {
		if block["hash"] != w.Transaction["blockHash"] || block["number"] != w.Transaction["blockNumber"] {
			return nil, fmt.Errorf("wrong inclusion block")
		}
	}
	if w.Transaction["type"] != "0x2" || w.ReceiptMetric < 1 {
		return nil, fmt.Errorf("dynamic type or snap path missing")
	}
	for _, key := range []string{"effectiveGasPrice", "gasUsed"} {
		if w.BPReceipt[key] != w.SnapReceipt[key] {
			return nil, fmt.Errorf("snap receipt differs: %s", key)
		}
	}
	base, e := regularEGPQuantity(w.Block, "baseFeePerGas")
	if e != nil {
		return nil, e
	}
	tip, e := regularEGPQuantity(w.Extra, "gasTip")
	if e != nil {
		return nil, e
	}
	txTip, e := regularEGPQuantity(w.Transaction, "maxPriorityFeePerGas")
	if e != nil {
		return nil, e
	}
	cap, e := regularEGPQuantity(w.Transaction, "maxFeePerGas")
	if e != nil {
		return nil, e
	}
	price, e := regularEGPQuantity(w.BPReceipt, "effectiveGasPrice")
	if e != nil {
		return nil, e
	}
	forced := new(big.Int).Add(base, tip)
	original := new(big.Int).Add(base, txTip)
	if txTip.Cmp(tip) <= 0 || cap.Cmp(original) <= 0 {
		return nil, fmt.Errorf("input does not distinguish original and current formulas")
	}
	if price.Cmp(forced) != 0 || price.Cmp(original) == 0 {
		return nil, fmt.Errorf("header replacement conflict not established")
	}
	return map[string]string{"forced": forced.String(), "original": original.String(), "observed": price.String(), "baseFee": base.String(), "headerTip": tip.String(), "txTip": txTip.String(), "feeCap": cap.String()}, nil
}

func TestRegularEGPWitnessRejectsAmbiguousOrUnboundEvidence(t *testing.T) {
	hash := "0x" + strings.Repeat("a", 64)
	block := "0x" + strings.Repeat("b", 64)
	valid := func() regularEGPWitness {
		return regularEGPWitness{Sender: "0x" + strings.Repeat("1", 40), Authorization: "0x" + strings.Repeat("0", 64), Transaction: map[string]any{"hash": hash, "from": "0x" + strings.Repeat("1", 40), "blockHash": block, "blockNumber": "0x8", "type": "0x2", "maxPriorityFeePerGas": "0x32", "maxFeePerGas": "0x3e8"}, BPReceipt: map[string]any{"transactionHash": hash, "blockHash": block, "blockNumber": "0x8", "type": "0x2", "status": "0x1", "effectiveGasPrice": "0x6e", "gasUsed": "0x5208"}, SnapReceipt: map[string]any{"transactionHash": hash, "blockHash": block, "blockNumber": "0x8", "type": "0x2", "status": "0x1", "effectiveGasPrice": "0x6e", "gasUsed": "0x5208"}, Block: map[string]any{"hash": block, "number": "0x8", "baseFeePerGas": "0x64"}, Extra: map[string]any{"gasTip": "0xa"}, SnapBlock: map[string]any{"hash": block, "number": "0x8"}, ReceiptMetric: 1}
	}
	if _, e := verifyRegularEGPWitness(valid()); e != nil {
		t.Fatal("valid distinguishing evidence rejected", e)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*regularEGPWitness)
	}{
		{"authorized", func(w *regularEGPWitness) { w.Authorization = "0x" + strings.Repeat("0", 63) + "1" }},
		{"missing authorization", func(w *regularEGPWitness) { w.Authorization = "" }},
		{"wrong sender", func(w *regularEGPWitness) { w.Sender = "0x" + strings.Repeat("2", 40) }},
		{"wrong hash", func(w *regularEGPWitness) { w.Transaction["hash"] = "0x" + strings.Repeat("c", 64) }},
		{"wrong inclusion", func(w *regularEGPWitness) { w.Block["hash"] = "0x" + strings.Repeat("c", 64) }},
		{"different snap transaction", func(w *regularEGPWitness) { w.SnapReceipt["transactionHash"] = "0x" + strings.Repeat("c", 64) }},
		{"different snap block", func(w *regularEGPWitness) { w.SnapBlock["hash"] = "0x" + strings.Repeat("c", 64) }},
		{"different snap price", func(w *regularEGPWitness) { w.SnapReceipt["effectiveGasPrice"] = "0x96" }},
		{"different snap gas", func(w *regularEGPWitness) { w.SnapReceipt["gasUsed"] = "0x5209" }},
		{"failed receipt", func(w *regularEGPWitness) { w.BPReceipt["status"] = "0x0" }},
		{"no actual receipt download", func(w *regularEGPWitness) { w.ReceiptMetric = 0 }},
		{"equal tip ambiguous", func(w *regularEGPWitness) { w.Transaction["maxPriorityFeePerGas"] = "0xa" }},
		{"cap erases distinction", func(w *regularEGPWitness) { w.Transaction["maxFeePerGas"] = "0x6e" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := valid()
			tc.mutate(&w)
			if _, e := verifyRegularEGPWitness(w); e == nil {
				t.Fatal("ambiguous or unbound evidence accepted")
			}
		})
	}
}

func TestRegularEGPFixtureFitsFundedSender(t *testing.T) {
	b, e := os.ReadFile(filepath.Join(repoRoot(t), "tests/tc/common/fee/CT-FEE-009-snap-receipt-gas-price.json"))
	if e != nil {
		t.Fatal(e)
	}
	var doc struct{ Steps []map[string]any }
	if e = json.Unmarshal(b, &doc); e != nil {
		t.Fatal(e)
	}
	var funding, gas, value *big.Int
	number := func(v any) *big.Int {
		n, ok := new(big.Int).SetString(fmt.Sprint(v), 10)
		if !ok {
			t.Fatal("bad fixture input", v)
		}
		return n
	}
	for _, s := range doc.Steps {
		if s["do"] == "sendTx" && s["to"] == "$sender" {
			funding = number(s["value"])
		}
		if s["do"] == "sendTx" && s["save"] == "h" {
			gas = number(s["gas"])
			value = number(s["value"])
		}
	}
	if funding == nil || gas == nil || value == nil {
		t.Fatal("missing funded subject inputs")
	}
	b, e = os.ReadFile(filepath.Join(repoRoot(t), "internal/chains/stablenet/genesis.json"))
	if e != nil {
		t.Fatal(e)
	}
	var genesis struct {
		BaseFeePerGas string
		Config        struct {
			Anzeon struct {
				SystemContracts struct {
					GovValidator struct{ Params struct{ GasTip string } }
				}
			}
		}
	}
	b = []byte(strings.NewReplacer("__CHAIN_ID__", "0", "__ALLOC_JSON__", "{}").Replace(string(b)))
	if e = json.Unmarshal(b, &genesis); e != nil {
		t.Fatal(e)
	}
	header := number(genesis.Config.Anzeon.SystemContracts.GovValidator.Params.GasTip)
	base, ok := new(big.Int).SetString(strings.TrimPrefix(genesis.BaseFeePerGas, "0x"), 16)
	if !ok {
		t.Fatal("bad default base fee")
	}
	requested := new(big.Int).Add(new(big.Int).Mul(header, big.NewInt(5)), big.NewInt(1))
	minimum := new(big.Int).Add(requested, base)
	if number(regularEGPFeeCap).Cmp(minimum) <= 0 {
		t.Fatalf("cap %s does not exceed distinguishing priority plus default base fee %s", regularEGPFeeCap, minimum)
	}

	cost := new(big.Int).Add(value, new(big.Int).Mul(gas, number(regularEGPFeeCap)))
	if cost.Cmp(funding) > 0 {
		t.Fatalf("maximum upfront cost %s exceeds fixture funding %s", cost, funding)
	}
}

func TestRegularEGPHeaderTipWireShape(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		pass  bool
	}{
		{"real decimal RPC field", "27600000000000", true}, {"hex control", "0xa", true},
		{"negative", "-1", false}, {"fractional", "1.2", false}, {"scientific", "1e3", false}, {"float", float64(10), false}, {"object", map[string]any{"value": "10"}, false}, {"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, e := regularEGPQuantity(map[string]any{"gasTip": tc.value}, "gasTip")
			if (e == nil) != tc.pass {
				t.Fatalf("header tip shape %v: %v %v", tc.value, n, e)
			}
		})
	}
	if _, e := regularEGPQuantity(map[string]any{"baseFeePerGas": "20000000000000"}, "baseFeePerGas"); e == nil {
		t.Fatal("ordinary ETH quantity accepted non-hex wire data")
	}
}
