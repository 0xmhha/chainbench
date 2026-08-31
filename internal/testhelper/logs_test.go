package testhelper

import (
	"context"
	"encoding/json"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// logsRPC serves a canned eth_getLogs result and records the filter it received.
type logsRPC struct {
	filter map[string]any
	result []any
}

func (s *logsRPC) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		if req.Method != "eth_getLogs" {
			http.Error(w, "unknown method "+req.Method, http.StatusBadRequest)
			return
		}
		if len(req.Params) > 0 {
			s.filter, _ = req.Params[0].(map[string]any)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": s.result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func transferLog() map[string]any {
	return map[string]any{
		"address": "0x0000000000000000000000000000000000001001",
		"topics": []any{
			"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x0000000000000000000000000000000000000000000000000000000000000002",
		},
		"data":        "0x00000000000000000000000000000000000000000000000000000000000003e8",
		"blockNumber": "0x10",
		"txHash":      "0xabc",
	}
}

func TestLogsAssertion_CountsMatchingLogs(t *testing.T) {
	srv := (&logsRPC{result: []any{transferLog(), transferLog()}}).server(t)
	d := deps()
	as, ok := d.Actions.Assertion(assertLogs)
	if !ok {
		t.Fatal("logs assertion not registered")
	}
	res, err := as.Check(context.Background(), &interp.AssertCtx{
		Env:  envWithNode(t, srv.URL),
		Deps: &d,
		Spec: map[string]any{"assert": assertLogs, "expected": "2"},
	})
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if !res.Pass {
		t.Fatalf("expected pass, got %+v", res)
	}
}

func TestLogsAssertion_PassesTheFilterThrough(t *testing.T) {
	rec := &logsRPC{result: []any{transferLog()}}
	srv := rec.server(t)
	d := deps()
	as, _ := d.Actions.Assertion(assertLogs)
	_, err := as.Check(context.Background(), &interp.AssertCtx{
		Env:  envWithNode(t, srv.URL),
		Deps: &d,
		Spec: map[string]any{
			"assert":    assertLogs,
			"address":   "0x0000000000000000000000000000000000001001",
			"topics":    []any{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"},
			"fromBlock": "0x5",
			"toBlock":   "latest",
			"expected":  "1",
		},
	})
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if rec.filter["address"] != "0x0000000000000000000000000000000000001001" {
		t.Fatalf("address filter = %v", rec.filter["address"])
	}
	if rec.filter["fromBlock"] != "0x5" || rec.filter["toBlock"] != "latest" {
		t.Fatalf("block range = %v..%v", rec.filter["fromBlock"], rec.filter["toBlock"])
	}
	topics, _ := rec.filter["topics"].([]any)
	if len(topics) != 1 {
		t.Fatalf("topics = %v", rec.filter["topics"])
	}
}

func TestLogsAssertion_SelectsAFieldOfOneLog(t *testing.T) {
	srv := (&logsRPC{result: []any{transferLog()}}).server(t)
	d := deps()
	as, _ := d.Actions.Assertion(assertLogs)

	cases := []struct {
		sel  string
		want any
	}{
		{"data", "0x00000000000000000000000000000000000000000000000000000000000003e8"},
		{"topic1", "0x0000000000000000000000000000000000000000000000000000000000000001"},
		{"topic2", "0x0000000000000000000000000000000000000000000000000000000000000002"},
		{"address", "0x0000000000000000000000000000000000001001"},
		{"blockNumber", "16"},
	}
	for _, tc := range cases {
		t.Run(tc.sel, func(t *testing.T) {
			res, err := as.Check(context.Background(), &interp.AssertCtx{
				Env:  envWithNode(t, srv.URL),
				Deps: &d,
				Spec: map[string]any{"assert": assertLogs, "select": tc.sel, "expected": tc.want},
			})
			if err != nil {
				t.Fatalf("logs: %v", err)
			}
			if !res.Pass {
				t.Fatalf("select %s: actual %#v, expected %#v", tc.sel, res.Actual, tc.want)
			}
		})
	}
}

func TestLogsAssertion_SelectOnAnEmptyResultIsAnError(t *testing.T) {
	srv := (&logsRPC{result: []any{}}).server(t)
	d := deps()
	as, _ := d.Actions.Assertion(assertLogs)
	res, err := as.Check(context.Background(), &interp.AssertCtx{
		Env:  envWithNode(t, srv.URL),
		Deps: &d,
		Spec: map[string]any{"assert": assertLogs, "select": "data", "expected": "0x"},
	})
	if err == nil {
		t.Fatalf("expected an error selecting a field with no matching logs, got %+v", res)
	}
}

func TestLogsAssertion_CountOnAnEmptyResultIsZeroNotAnError(t *testing.T) {
	srv := (&logsRPC{result: []any{}}).server(t)
	d := deps()
	as, _ := d.Actions.Assertion(assertLogs)
	res, err := as.Check(context.Background(), &interp.AssertCtx{
		Env:  envWithNode(t, srv.URL),
		Deps: &d,
		Spec: map[string]any{"assert": assertLogs, "expected": "0"},
	})
	if err != nil {
		t.Fatalf("counting zero logs should not error: %v", err)
	}
	if !res.Pass {
		t.Fatalf("expected pass on a zero count, got %+v", res)
	}
}

// A governance flow: send a transaction, then assert the event it emitted, with
// the log filtered to the block the receipt reports.
func TestRun_EventAssertionAfterATransaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch req.Method {
		case "eth_sendTransaction":
			result = "0xsent"
		case "eth_getTransactionReceipt":
			result = map[string]any{"status": "0x1", "blockNumber": "0x10"}
		case "eth_getLogs":
			result = []any{transferLog()}
		default:
			http.Error(w, "unknown method "+req.Method, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)

	d := deps()
	spec := dsl.Spec{
		Steps: []map[string]any{
			{actionSendTx: map[string]any{"from": "0xa", "to": "0xb", "save": "hash"}},
		},
		Assertions: []map[string]any{
			{"assert": assertLogs, "select": "count", "compare": "GreaterOrEqual", "expected": "1"},
			{"assert": assertTxStatus, "hash": "$hash", "expected": "0x1"},
		},
	}
	st, err := interp.NewInterpreter(d).Run(context.Background(), spec, envWithNode(t, srv.URL), &recordStub{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st != "pass" {
		t.Fatalf("status = %s, want pass", st)
	}
}
