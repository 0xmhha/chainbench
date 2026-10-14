package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeriveMinimumFeeCap(t *testing.T) {
	for _, tc := range []struct {
		name string
		of   []any
		want string
	}{
		{"tip limited", []any{"107", "150"}, "107"},
		{"fee cap limited", []any{"107", "103"}, "103"},
		{"zero", []any{"0x0", "100"}, "0"},
		{"arbitrary precision", []any{"100000000000000000000000000000000000001", "100000000000000000000000000000000000000"}, "100000000000000000000000000000000000000"},
		{"negative derived value", []any{"-1", "0", "-5"}, "-5"},
		{"single operand", []any{"0x64"}, "100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readDerive(context.Background(), nil, nil, map[string]any{"op": "min", "of": tc.of})
			if err != nil || got != tc.want {
				t.Fatalf("minimum=%v err=%v want=%s", got, err, tc.want)
			}
		})
	}
	for _, spec := range []map[string]any{{"op": "pow", "of": []any{"1"}}, {"op": "min", "of": []any{}}, {"op": "min", "of": []any{"invalid"}}} {
		if _, err := readDerive(context.Background(), nil, nil, spec); err == nil {
			t.Errorf("invalid arithmetic accepted: %v", spec)
		}
	}
	_, schemas, err := EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"reader-derive", "assertion-derive"} {
		properties := schemas[kind]["properties"].(map[string]any)
		choices := properties["op"].(map[string]any)["enum"].([]string)
		found := false
		for _, choice := range choices {
			if choice == "min" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s editor cannot select the supported minimum", kind)
		}
	}
}

func TestPublicFeeCasesBindActualTransactionPrices(t *testing.T) {
	for _, tc := range []struct {
		name, caseFile                   string
		txType, receiptType              string
		cap, tip, gasPrice, receiptPrice int64
		want                             session.TestStatus
	}{
		{"dynamic tip limited", "CT-TX-003-dynamic-fee-transfer.json", "0x2", "0x2", 150, 7, 107, 107, session.StatusPass},
		{"dynamic cap limited", "CT-TX-003-dynamic-fee-transfer.json", "0x2", "0x2", 103, 7, 103, 103, session.StatusPass},
		{"wrong capped price", "CT-TX-003-dynamic-fee-transfer.json", "0x2", "0x2", 103, 7, 107, 107, session.StatusFail},
		{"wrong actual cap", "CT-TX-003-dynamic-fee-transfer.json", "0x2", "0x2", 151, 7, 107, 107, session.StatusFail},
		{"wrong actual tip", "CT-TX-003-dynamic-fee-transfer.json", "0x2", "0x2", 150, 8, 107, 107, session.StatusFail},
		{"wrong receipt type", "CT-TX-003-dynamic-fee-transfer.json", "0x2", "0x0", 150, 7, 107, 107, session.StatusFail},
		{"legacy exact price", "CT-TX-002-legacy-transfer.json", "0x0", "0x0", 150, 7, 107, 107, session.StatusPass},
		{"legacy wrong receipt price", "CT-TX-002-legacy-transfer.json", "0x0", "0x0", 150, 7, 107, 105, session.StatusFail},
		{"legacy wrong signed price", "CT-TX-002-legacy-transfer.json", "0x0", "0x0", 150, 7, 108, 107, session.StatusFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const sender = "0x0000000000000000000000000000000000001234"
			const recipient = "0x0000000000000000000000000000000000005678"
			balance, _ := new(big.Int).SetString("10000000000000000000", 10)
			amount := big.NewInt(1000000000000000)
			fee := new(big.Int).Mul(big.NewInt(21000), big.NewInt(tc.receiptPrice))
			after := new(big.Int).Sub(new(big.Int).Sub(balance, amount), fee)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []any
				}
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				var result any
				switch q.Method {
				case "eth_getTransactionReceipt":
					result = map[string]any{"status": "0x1", "type": tc.receiptType, "blockNumber": "0x20", "gasUsed": "0x5208", "effectiveGasPrice": fmt.Sprintf("0x%x", tc.receiptPrice)}
				case "eth_getTransactionByHash":
					result = map[string]any{"type": tc.txType, "maxFeePerGas": fmt.Sprintf("0x%x", tc.cap), "maxPriorityFeePerGas": fmt.Sprintf("0x%x", tc.tip), "gasPrice": fmt.Sprintf("0x%x", tc.gasPrice)}
				case "eth_getBlockByNumber":
					result = map[string]any{"baseFeePerGas": "0x64"}
				case "eth_getBalance":
					if q.Params[0] == sender {
						result = "0x" + after.Text(16)
					} else {
						result = "0x" + amount.Text(16)
					}
				default:
					t.Errorf("unexpected RPC %s", q.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			s := nonceCase(t, tc.caseFile)
			start := 0
			for i, st := range s.Sequence {
				if st.Do == "sendTx" {
					start = i + 1
				}
			}
			s.Sequence = s.Sequence[start:]
			declaredCap := int64(150)
			if tc.name == "dynamic cap limited" || tc.name == "wrong capped price" {
				declaredCap = 103
			}
			s.EnvValues = map[string]any{"sender": sender, "recipient": recipient, "before": balance.String(), "h": "hash", "tip": "7", "feeCap": fmt.Sprint(declaredCap), "gp": "107"}
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if err != nil || got != tc.want {
				t.Fatalf("public fee case=%s err=%v want=%s; assertions=%+v", got, err, tc.want, rec.asserts)
			}
		})
	}
}

func TestMinimumFormatIsConsumedThroughPublicValidation(t *testing.T) {
	s := argumentCase(t, `{"do":"read","source":"derive","op":"min","of":["0x64","0x7"],"format":"hex","save":"n"},{"expect":"derive","op":"min","of":["$n","99"],"format":"dec","is":"7"}`)
	if ignored := IgnoredArguments(s); len(ignored) != 0 {
		t.Fatalf("valid minimum format rejected as unused: %v", ignored)
	}
	got, err := readDerive(context.Background(), nil, nil, map[string]any{"op": "min", "of": []any{"0x64", "0x7"}, "format": "hex"})
	if err != nil || got != "0x7" {
		t.Fatalf("minimum hex=%v err=%v", got, err)
	}
}
