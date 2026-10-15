package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPublicAccessListRequiresDeployedSlotZero(t *testing.T) {
	for _, name := range []string{"valid", "wrong contract", "wrong slot", "empty slots", "missing list", "wrong transaction type", "wrong receipt type", "wrong stored value"} {
		t.Run(name, func(t *testing.T) {
			box := "0x" + strings.Repeat("12", 20)
			slot := "0x" + strings.Repeat("0", 64)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []any
				}
				if e := json.NewDecoder(r.Body).Decode(&q); e != nil {
					t.Error(e)
					return
				}
				var result any
				switch q.Method {
				case "eth_getTransactionByHash":
					result = map[string]any{"type": "0x1", "accessList": []any{map[string]any{"address": "0x00000000000000000000000000000000c0ffee0d", "storageKeys": []any{}}}}
					if q.Params[0] == "slotTx" {
						tuple := map[string]any{"address": box, "storageKeys": []any{slot}}
						tx := map[string]any{"type": "0x1", "to": box, "input": "0x" + strings.Repeat("0", 62) + "2a", "accessList": []any{tuple}}
						switch name {
						case "wrong contract":
							tuple["address"] = "0x" + strings.Repeat("34", 20)
						case "wrong slot":
							tuple["storageKeys"] = []any{"0x" + strings.Repeat("0", 63) + "1"}
						case "empty slots":
							tuple["storageKeys"] = []any{}
						case "missing list":
							tx["accessList"] = []any{}
						case "wrong transaction type":
							tx["type"] = "0x0"
						}
						result = tx
					}
				case "eth_getTransactionReceipt":
					typ := "0x1"
					if q.Params[0] == "slotTx" && name == "wrong receipt type" {
						typ = "0x0"
					}
					result = map[string]any{"status": "0x1", "type": typ, "contractAddress": box}
				case "eth_getStorageAt":
					if len(q.Params) != 3 || q.Params[0] != box || q.Params[1] != slot {
						t.Error("wrong storage probe", q.Params)
					}
					value := "0x" + strings.Repeat("0", 62) + "2a"
					if name == "wrong stored value" {
						value = "0x" + strings.Repeat("0", 64)
					}
					result = value
				default:
					t.Error("unexpected assertion RPC", q.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			raws, e := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/tx/CT-TX-004-access-list-tx.json"}, "../../presets/chain/stablenet-bp4.json")
			if e != nil {
				t.Fatal(e)
			}
			s, e := dsl.ParseV2(raws[0])
			if e != nil {
				t.Fatal(e)
			}
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Expect == "rpcCall" || st.Expect == "txStatus" {
					selected = append(selected, st)
				}
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"hash": "oldTx", "slotHash": "slotTx", "slotBox": box}
			rec := &recordStub{}
			got, e := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if name == "valid" {
				if e != nil || got != session.StatusPass {
					t.Fatalf("valid slot list failed: %s %v %+v", got, e, rec.asserts)
				}
			} else if e == nil && got == session.StatusPass {
				t.Fatalf("invalid %s accepted: %+v", name, rec.asserts)
			}
		})
	}
}

func TestAccessListCaseUsesDedicatedSignedDeployment(t *testing.T) {
	p := "../../tests/tc/common/tx/CT-TX-004-access-list-tx.json"
	raw, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	var doc struct{ Steps []map[string]any }
	if e = json.Unmarshal(raw, &doc); e != nil {
		t.Fatal(e)
	}
	found := false
	for _, st := range doc.Steps {
		if st["do"] == "deployContract" && st["save"] == "slotBox" {
			found = true
			if st["key"] != "$slotSenderKey" || st["bytecode"] != "0x6007600c60003960076000f360003560005500" {
				t.Fatal("wrong local creation fixture")
			}
		}
	}
	if !found {
		t.Fatal("slot deployment must use the local creation action and bind its actual address")
	}
	raws, e := dsl.ReadFilesWithChainPreset([]string{p}, "../../presets/chain/stablenet-bp4.json")
	if e != nil {
		t.Fatal(e)
	}
	spec, e := dsl.ParseV2(raws[0])
	if e != nil {
		t.Fatal(e)
	}
	if ignored := IgnoredArguments(spec); len(ignored) != 0 {
		t.Fatal("unsupported creation/call inputs", ignored)
	}
}

func TestAccessListPreparationUsesBoundedCurrentFeeInputs(t *testing.T) {
	raw, e := os.ReadFile("../../tests/tc/common/tx/CT-TX-004-access-list-tx.json")
	if e != nil {
		t.Fatal(e)
	}
	var doc struct{ Steps []map[string]any }
	if e = json.Unmarshal(raw, &doc); e != nil {
		t.Fatal(e)
	}
	funded, quotedBeforeDeploy, boundedSimulation := false, false, false
	quoteSeen := false
	for _, st := range doc.Steps {
		if st["do"] == "read" && st["source"] == "gasPrice" && st["save"] == "slotGP" {
			quoteSeen = true
		}
		if st["do"] == "sendTx" && st["to"] == "$slotSender" {
			funded = st["value"] == "$slotBudget"
		}
		if st["do"] == "deployContract" && st["save"] == "slotBox" {
			quotedBeforeDeploy = quoteSeen && st["gasPrice"] == "$slotGP"
		}
		if st["method"] == "eth_createAccessList" {
			params := st["params"].([]any)
			call := params[0].(map[string]any)
			if call["to"] == "$slotBox" {
				boundedSimulation = call["gas"] == "0x186a0" && call["gasPrice"] == "$slotGPHex"
			}
		}
	}
	if !funded || !quotedBeforeDeploy || !boundedSimulation {
		t.Fatalf("preparation relies on unbounded defaults: budget=%v deployment=%v simulation=%v", funded, quotedBeforeDeploy, boundedSimulation)
	}
}

func TestAccessListFundingBudgetScalesWithActualQuote(t *testing.T) {
	for _, price := range []int64{2000000000, 100000000000, 67600000000000} {
		t.Run(fmt.Sprint(price), func(t *testing.T) {
			srv := mockRPC(t, map[string]any{"eth_gasPrice": fmt.Sprintf("0x%x", price)})
			raws, e := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/tx/CT-TX-004-access-list-tx.json"}, "../../presets/chain/stablenet-bp4.json")
			if e != nil {
				t.Fatal(e)
			}
			s, e := dsl.ParseV2(raws[0])
			if e != nil {
				t.Fatal(e)
			}
			sequence := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Do == "read" && (st.Args["source"] == "gasPrice" || st.Args["source"] == "derive") {
					sequence = append(sequence, st)
				}
			}
			expected := new(big.Int).Add(new(big.Int).Mul(big.NewInt(price), big.NewInt(500000)), big.NewInt(1000000000000000000))
			sequence = append(sequence, dsl.Statement{Expect: "derive", Args: map[string]any{"op": "sum", "of": []any{"$slotBudget"}, "compare": "Equal", "expected": expected.String()}})
			s.Sequence = sequence
			rec := &recordStub{}
			got, e := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if e != nil || got != session.StatusPass {
				t.Fatalf("quote-based budget lost: %s %v %+v", got, e, rec.asserts)
			}
		})
	}
}
