package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type durationForkCrosser struct {
	fakeNodeControl
	calls atomic.Int32
}

func (c *durationForkCrosser) CrossFork(context.Context, time.Duration) ([]node.Node, error) {
	c.calls.Add(1)
	return nil, nil
}

func TestDurationControlsRejectInvalidExplicitValuesBeforeBuiltinIO(t *testing.T) {
	for _, c := range []struct{ kind, name, field string }{
		{"action", "crossFork", "timeout"},
		{"action", "sendTx", "timeout"}, {"action", "sendTx", "pollInterval"},
		{"action", "waitBlock", "timeout"}, {"action", "waitBlock", "pollInterval"},
		{"action", "waitFor", "timeout"}, {"action", "waitFor", "pollInterval"},
		{"action", "load", "timeout"}, {"action", "load", "pollInterval"},
		{"action", "faucet", "timeout"}, {"action", "faucet", "pollInterval"},
		{"action", "deployContract", "timeout"}, {"action", "deployContract", "pollInterval"},
		{"action", "registerContract", "timeout"}, {"action", "registerContract", "pollInterval"},
		{"assertion", "blockHalt", "within"},
		{"assertion", "blockStalled", "timeout"}, {"assertion", "blockStalled", "pollInterval"},
		{"assertion", "blockLag", "timeout"}, {"assertion", "blockLag", "pollInterval"},
		{"assertion", "blockAdvance", "timeout"}, {"assertion", "blockAdvance", "pollInterval"},
		{"assertion", "wsSubscribe", "timeout"}, {"assertion", "wsCollected", "timeout"},
	} {
		for index, bad := range []any{"0s", "-1ms", "", "invalid", true, nil, 1.5, "9223372036854775808ns"} {
			t.Run(fmt.Sprintf("%s/%s/%d", c.name, c.field, index), func(t *testing.T) {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("invalid duration panicked instead of returning a field error: %v", p)
					}
				}()
				var requests, opens atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					var req struct {
						ID     any
						Method string
					}
					_ = json.NewDecoder(r.Body).Decode(&req)
					var result any = "0x3"
					switch req.Method {
					case "eth_sendTransaction":
						result = "0xhash"
					case "eth_getTransactionReceipt":
						result = map[string]any{"status": "0x1", "contractAddress": testTo}
					case "eth_getBlockByNumber":
						result = map[string]any{"number": "0x3", "gasLimit": "0xf4240", "baseFeePerGas": "0x1"}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
				}))
				defer server.Close()
				d := deps()
				control := &durationForkCrosser{}
				d.Nodes = control
				env := envWithNode(t, server.URL)
				args := map[string]any{"on": "node1", "from": testTo, "to": testTo, "value": "0", "amount": "1", "gas": "21000", "bytecode": "0x00", "data": "0x", "source": "blockNumber", "expected": "3", "target": 3, "blocks": 1, "timeout": "5ms", "pollInterval": "1ms", "within": "1ms"}
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()
				if c.name == "wsSubscribe" || c.name == "wsCollected" {
					base := wsHeadServerOf(t, []string{"0x1"})
					ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { opens.Add(1); base.Config.Handler.ServeHTTP(w, r) }))
					defer ws.Close()
					host, port := hostPort(t, ws.URL)
					env = envWithWS(t, host, port)
					if c.name == "wsCollected" {
						sub, err := rpc.Subscribe(ctx, strings.Replace(ws.URL, "http://", "ws://", 1), "newHeads")
						if err != nil {
							t.Fatal(err)
						}
						defer func() { _ = sub.Close() }()
						args["sub"] = sub
						opens.Store(0)
					}
				}
				args[c.field] = bad
				var err error
				if c.kind == "action" {
					a, _ := d.Actions.Action(c.name)
					err = a.Do(ctx, &interp.ActionCtx{Env: env, Deps: &d, Args: args})
				} else {
					a, _ := d.Actions.Assertion(c.name)
					_, err = a.Check(ctx, &interp.AssertCtx{Env: env, Deps: &d, On: env.Nodes(), Spec: args})
				}
				if err == nil || !strings.Contains(err.Error(), "bad "+c.field) {
					t.Errorf("invalid explicit duration hidden or unidentified: %v", err)
				}
				if requests.Load() != 0 || opens.Load() != 0 || control.calls.Load() != 0 {
					t.Errorf("invalid duration reached IO: RPC=%d WS=%d fork=%d", requests.Load(), opens.Load(), control.calls.Load())
				}
			})
		}
	}
}

func TestDurationControlsPreserveOmissionAndGoDurationLiterals(t *testing.T) {
	args := map[string]any{}
	if err := validateDurationControl("waitBlock", "timeout", args); err != nil || durationArg(args, "timeout", defaultWaitBlockTimeout) != defaultWaitBlockTimeout {
		t.Fatal("omission changed the existing default", err)
	}
	for _, text := range []string{"1ns", "1.5ms", "250µs", "250us", "+2s", "1m0.5s", "9223372036854775807ns"} {
		args["timeout"] = text
		want, err := time.ParseDuration(text)
		if err != nil || validateDurationControl("waitBlock", "timeout", args) != nil || durationArg(args, "timeout", time.Second) != want {
			t.Fatal("valid Go duration changed", text, err)
		}
	}
	if got := editorDefaults("blockHalt")["within"]; got != defaultBlockHaltWindow.String() {
		t.Fatalf("editor omitted the runtime watch-window default: %v", got)
	}
}

func TestInvalidCollectedDurationClosesTheExistingSubscription(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := wsHeadServerOf(t, nil)
	sub, err := rpc.Subscribe(ctx, strings.Replace(server.URL, "http://", "ws://", 1), "newHeads")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sub.Close() }()
	result, err := (wsCollectedAssertion{}).Check(ctx, &interp.AssertCtx{Spec: map[string]any{"sub": sub, "timeout": "0s"}})
	if err == nil || !strings.Contains(err.Error(), "bad timeout") {
		t.Fatal(err)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatal("live subscription leaked into evidence", err)
	}
	select {
	case _, open := <-sub.Notifications():
		if open {
			t.Fatal("invalid timeout left notifications open")
		}
	case <-time.After(time.Second):
		t.Fatal("invalid timeout did not release its subscription")
	}
}
