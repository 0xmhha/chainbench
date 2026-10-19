package testhelper

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// A successful finalization receipt and refundable amount must not conceal a
// retained burn ledger, early native payout or incorrectly attributed event.
func TestPublicBurnRefundFinalizationKeepsDepositAndExactEvent(t *testing.T) {
	const amount = "1000000000000000000"
	const minter = "0x0000000000000000000000000000000000001003"
	const event = "0x116044c8ec23de4088f4a50cf4f8a504ec887c439452af3be4f57f6c77bf2062"
	word := func(s string) string {
		return "0x" + strings.Repeat("0", 64-len(strings.TrimPrefix(s, "0x"))) + strings.TrimPrefix(s, "0x")
	}
	for _, c := range []struct {
		file, caller, final string
		status              int
	}{{"03-burn-cancel-refundable.json", "node2", "cancelHash", 4}, {"04-burn-reject-refundable.json", "node3", "dis3Hash", 7}, {"05-burn-expire-refundable.json", "node3", "expHash", 5}} {
		for _, mutation := range []string{"valid", "burn balance retained", "native returned before claim", "wrong emitter", "wrong proposal", "wrong requester", "wrong amount"} {
			t.Run(c.file+"/"+mutation, func(t *testing.T) {
				d, ring := depsWithRing(t)
				d.RPC = deps().RPC
				d.Actions = testhelperRegistry()
				d.Contracts = contractsOf("stablenet")
				for _, label := range []keyring.Label{"node3", "node4"} {
					if _, e := ring.Add(context.Background(), label, keyring.RandomSource{}, derive.AccountOnly); e != nil {
						t.Fatal(e)
					}
				}
				entry, ok := ring.Get(keyring.Label(c.caller))
				if !ok {
					t.Fatal("missing generated caller")
				}
				address := strings.TrimPrefix(entry.Address, "0x")
				addressNumber, ok := new(big.Int).SetString(address, 16)
				if !ok {
					t.Fatal("invalid generated address")
				}
				log := map[string]any{"address": minter, "topics": []any{event, word("0x1"), word(entry.Address)}, "data": word("0xde0b6b3a7640000"), "blockNumber": "0x10", "transactionHash": "0x" + strings.Repeat("a", 64)}
				switch mutation {
				case "wrong emitter":
					log["address"] = "0x0000000000000000000000000000000000001002"
				case "wrong proposal":
					log["topics"].([]any)[1] = word("0x2")
				case "wrong requester":
					log["topics"].([]any)[2] = word("0x0")
				case "wrong amount":
					log["data"] = word("0x1")
				}
				receiptLogs := []any{log}
				if c.final == "dis3Hash" {
					rejector, _ := ring.Get("node4")
					receiptLogs = append(receiptLogs, map[string]any{"address": minter, "topics": []any{"0x6fa60fd92b2557298bb7b39e7b538b3f00ebf09b9f056bdeac7f0e5d57e8fcbb", word("0x1"), word(rejector.Address)}, "data": word("0x1") + strings.TrimPrefix(word("0x3"), "0x")})
				}
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
					var value any
					switch q.Method {
					case "eth_getTransactionReceipt":
						value = map[string]any{"status": "0x1", "blockNumber": "0x10", "logs": receiptLogs}
					case "eth_getBalance":
						value = "0xde0b6b3a7640000"
						if mutation == "native returned before claim" {
							value = "0x0"
						}
					case "eth_getLogs":
						value = receiptLogs
					case "eth_call":
						tx, _ := q.Params[0].(map[string]any)
						data, _ := tx["data"].(string)
						switch {
						case strings.HasPrefix(data, "0xb03d36cd"):
							value = word("0xde0b6b3a7640000")
						case strings.HasPrefix(data, "0x98179c41"):
							value = word("0x0")
							if mutation == "burn balance retained" {
								value = word("0x1")
							}
						case strings.HasPrefix(data, "0x013cf08b"):
							value = "0x" + strings.Repeat("0", 9*64) + strings.TrimPrefix(word("0x"+strconv.FormatInt(int64(c.status), 16)), "0x")
						default:
							t.Errorf("unexpected checkpoint call %s", data)
							http.Error(w, "unexpected", 400)
							return
						}
					default:
						t.Errorf("unexpected checkpoint method %s", q.Method)
						http.Error(w, "unexpected", 400)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": value})
				}))
				defer srv.Close()
				raws, e := dsl.ReadFiles([]string{filepath.Join("../../tests/tc/go-stablenet/post-v1.0.0-change/common-all", c.file)})
				if e != nil {
					t.Fatal(e)
				}
				spec, e := dsl.ParseV2(raws[0])
				if e != nil {
					t.Fatal(e)
				}
				checkpoint := []dsl.Statement{}
				found := false
				for _, s := range spec.Sequence {
					if !found {
						if s.Do == "sendTx" && s.Args["save"] == c.final {
							found = true
						}
						continue
					}
					if s.Do == "sendTx" {
						break
					}
					checkpoint = append(checkpoint, s)
				}
				if !found || len(checkpoint) == 0 {
					t.Fatal("public finalization checkpoint missing")
				}
				spec.Sequence = checkpoint
				values := map[string]any{"refBefore": "0", "govBefore": "0", "govAfterBurn": amount, "requesterNumber": addressNumber.String(), "refBalCall": "0xb03d36cd" + strings.TrimPrefix(word(entry.Address), "0x"), "burnBalanceCall": "0x98179c41" + strings.TrimPrefix(word(entry.Address), "0x"), "burnPid": word("0x1"), "cancelPid": word("0x1")}
				for _, key := range []string{"burnHash", "cancelHash", "dis1Hash", "dis2Hash", "dis3Hash", "expHash"} {
					values[key] = "0x" + strings.Repeat("a", 64)
				}
				spec.EnvValues = values
				rec := &recordStub{}
				status, e := interp.NewInterpreter(*d).Run(context.Background(), spec, envOf(t, srv.URL, srv.URL, srv.URL, srv.URL), rec)
				if mutation == "valid" {
					if e != nil || status != session.StatusPass {
						t.Fatalf("valid checkpoint rejected: %s %v %+v", status, e, rec.asserts)
					}
				} else if e == nil && status == session.StatusPass {
					t.Fatalf("public checkpoint hid %s: %+v", mutation, rec.asserts)
				}
			})
		}
	}
}
