package testspec

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// txRPC serves the send/receipt pair and records the transaction arguments it
// was given, so an action's assembled transaction can be inspected.
type txRPC struct {
	mu       sync.Mutex
	sent     []map[string]any
	receipt  map[string]any
	coinbase string
}

func (s *txRPC) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch req.Method {
		case "eth_coinbase":
			result = s.coinbase
		case "eth_sendTransaction":
			s.mu.Lock()
			if len(req.Params) > 0 {
				if m, ok := req.Params[0].(map[string]any); ok {
					s.sent = append(s.sent, m)
				}
			}
			s.mu.Unlock()
			result = "0xtxhash"
		case "eth_getTransactionReceipt":
			result = s.receipt
		default:
			http.Error(w, "unknown method "+req.Method, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFaucetAction_FundsFromTheNodeCoinbaseByDefault(t *testing.T) {
	rpcSrv := &txRPC{
		coinbase: "0xcoinbase",
		receipt:  map[string]any{"status": "0x1"},
	}
	srv := rpcSrv.server(t)
	d := deps()
	env := envWithNode(t, srv.URL)

	act, ok := d.Actions.Action(actionFaucet)
	if !ok {
		t.Fatal("faucet not registered")
	}
	err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{
		"to":     "0xrecipient",
		"amount": "1000",
	}})
	if err != nil {
		t.Fatalf("faucet: %v", err)
	}
	rpcSrv.mu.Lock()
	defer rpcSrv.mu.Unlock()
	if len(rpcSrv.sent) != 1 {
		t.Fatalf("sent %d transactions, want 1", len(rpcSrv.sent))
	}
	tx := rpcSrv.sent[0]
	if tx["from"] != "0xcoinbase" {
		t.Fatalf("from = %v, want the node coinbase", tx["from"])
	}
	if tx["to"] != "0xrecipient" {
		t.Fatalf("to = %v", tx["to"])
	}
	if tx["value"] != "0x3e8" { // 1000
		t.Fatalf("value = %v, want 0x3e8", tx["value"])
	}
}

func TestFaucetAction_RequiresRecipientAndAmount(t *testing.T) {
	d := deps()
	env := envWithNode(t, "http://unused")
	act, _ := d.Actions.Action(actionFaucet)
	if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d,
		Args: map[string]any{"amount": "1"}}); err == nil {
		t.Fatal("expected an error without a recipient")
	}
	if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d,
		Args: map[string]any{"to": "0xa"}}); err == nil {
		t.Fatal("expected an error without an amount")
	}
}

func TestDeployContractAction_BindsTheDeployedAddress(t *testing.T) {
	rpcSrv := &txRPC{
		coinbase: "0xcoinbase",
		receipt:  map[string]any{"status": "0x1", "contractAddress": "0xdeployed"},
	}
	srv := rpcSrv.server(t)
	d := deps()
	env := envWithNode(t, srv.URL)

	act, ok := d.Actions.Action(actionDeployContract)
	if !ok {
		t.Fatal("deployContract not registered")
	}
	ac := &ActionCtx{Env: env, Deps: &d, Args: map[string]any{
		"bytecode": "0x6080604052",
		"save":     "addr",
	}}
	if err := act.Do(context.Background(), ac); err != nil {
		t.Fatalf("deployContract: %v", err)
	}
	if ac.Value != "0xdeployed" {
		t.Fatalf("Value = %#v, want the contract address", ac.Value)
	}
	rpcSrv.mu.Lock()
	defer rpcSrv.mu.Unlock()
	tx := rpcSrv.sent[0]
	if _, hasTo := tx["to"]; hasTo {
		t.Fatalf("a deployment must not carry \"to\": %v", tx)
	}
	if tx["data"] != "0x6080604052" {
		t.Fatalf("data = %v", tx["data"])
	}
}

func TestDeployContractAction_FailsWhenTheReceiptHasNoAddress(t *testing.T) {
	rpcSrv := &txRPC{coinbase: "0xcoinbase", receipt: map[string]any{"status": "0x1"}}
	srv := rpcSrv.server(t)
	d := deps()
	act, _ := d.Actions.Action(actionDeployContract)
	err := act.Do(context.Background(), &ActionCtx{Env: envWithNode(t, srv.URL), Deps: &d,
		Args: map[string]any{"bytecode": "0x60"}})
	if err == nil {
		t.Fatal("expected an error when the receipt carries no contract address")
	}
	if !strings.Contains(err.Error(), "contract address") {
		t.Fatalf("error should say what was missing, got: %v", err)
	}
}

func TestRegisterContractAction_SendsToTheDeployedAddress(t *testing.T) {
	rpcSrv := &txRPC{coinbase: "0xcoinbase", receipt: map[string]any{"status": "0x1"}}
	srv := rpcSrv.server(t)
	d := deps()
	act, ok := d.Actions.Action(actionRegisterContract)
	if !ok {
		t.Fatal("registerContract not registered")
	}
	err := act.Do(context.Background(), &ActionCtx{Env: envWithNode(t, srv.URL), Deps: &d,
		Args: map[string]any{"to": "0xdeployed", "data": "0xabcdef"}})
	if err != nil {
		t.Fatalf("registerContract: %v", err)
	}
	rpcSrv.mu.Lock()
	defer rpcSrv.mu.Unlock()
	tx := rpcSrv.sent[0]
	if tx["to"] != "0xdeployed" || tx["data"] != "0xabcdef" {
		t.Fatalf("tx = %v", tx)
	}
}

func TestRegisterContractAction_RequiresATarget(t *testing.T) {
	d := deps()
	act, _ := d.Actions.Action(actionRegisterContract)
	err := act.Do(context.Background(), &ActionCtx{Env: envWithNode(t, "http://unused"), Deps: &d,
		Args: map[string]any{"data": "0xab"}})
	if err == nil {
		t.Fatal("expected an error without a target contract")
	}
}
