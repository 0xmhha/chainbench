package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type controlLogReader struct {
	fakeNodeControl
	reads atomic.Int32
	limit int
}

func (c *controlLogReader) Log(_ context.Context, _ node.Node, limit int) (string, error) {
	c.reads.Add(1)
	c.limit = limit
	return "public log", nil
}

func TestUnsignedBoundPatternsPreserveExactDecimalAndHexValues(t *testing.T) {
	for _, maximum := range []uint64{100, 9007199254740991, uint64(math.MaxInt), math.MaxUint64} {
		for _, minimum := range []uint64{0, 1} {
			re := regexp.MustCompile("^" + boundedUnsignedPattern(minimum, maximum) + "$")
			values := []uint64{0, 1, maximum, maximum - 1, math.MaxUint64}
			if maximum < math.MaxUint64 {
				values = append(values, maximum+1)
			}
			for n := uint64(0); n < 220; n++ {
				values = append(values, n)
			}
			for _, base := range []uint64{10, 16} {
				for n := uint64(1); n <= math.MaxUint64/base; n *= base {
					values = append(values, n, n-1, n+1)
				}
			}
			for _, value := range values {
				want := value >= minimum && value <= maximum
				for _, literal := range []string{strconv.FormatUint(value, 10), "000" + strconv.FormatUint(value, 10), "0x" + strconv.FormatUint(value, 16), "0X000" + strings.ToUpper(strconv.FormatUint(value, 16))} {
					if re.MatchString(literal) != want {
						t.Fatalf("range %d..%d incorrectly matches %s", minimum, maximum, literal)
					}
				}
			}
			for _, bad := range []string{"", "-1", "1.5", "0x", "18446744073709551616", "0x10000000000000000"} {
				if re.MatchString(bad) {
					t.Fatalf("range accepted malformed/overflow value: %s", bad)
				}
			}
		}
	}
}

func TestUnsignedZeroOmissionAndFullWidthNonceRemainDistinct(t *testing.T) {
	var nonceReads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonceReads.Add(1)
		var req struct{ ID any }
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x2"})
	}))
	defer server.Close()
	reader, _ := Registry().Reader("createAddress")
	zero, err := reader(context.Background(), nil, rpc.Dial(server.URL), map[string]any{"deployer": testTo, "nonce": "0X00"})
	if err != nil || nonceReads.Load() != 0 {
		t.Fatal("explicit zero became a nonce query", err)
	}
	omitted, err := reader(context.Background(), nil, rpc.Dial(server.URL), map[string]any{"deployer": testTo})
	if err != nil || nonceReads.Load() != 1 || zero == omitted {
		t.Fatal("omission did not query the current nonce", err)
	}
	large, err := reader(context.Background(), nil, rpc.Dial(server.URL), map[string]any{"deployer": testTo, "nonce": "18446744073709551615"})
	if err != nil || nonceReads.Load() != 1 || large == zero {
		t.Fatal("full-width nonce changed or queried RPC", err)
	}
	word, _ := Registry().Reader("derive")
	blob := "0x" + strings.Repeat("1", 64) + strings.Repeat("2", 64)
	for _, v := range []any{1, "1", "0X01"} {
		out, err := word(context.Background(), nil, nil, map[string]any{"op": "word", "of": []any{blob}, "index": v})
		if err != nil || out != "0x"+strings.Repeat("2", 64) {
			t.Fatalf("index was not preserved: %v %v %v", v, out, err)
		}
	}
	control := &controlLogReader{}
	d := deps()
	d.Nodes = control
	act, _ := Registry().Action("readNodeLog")
	if err := act.Do(context.Background(), &interp.ActionCtx{Env: envWithNode(t, server.URL), Deps: &d, Args: map[string]any{"on": "node1", "maxBytes": "0X10000"}}); err != nil || control.limit != 65536 {
		t.Fatal("log bound not preserved", err, control.limit)
	}
}

func TestUnsignedControlsRejectExplicitInvalidValuesBeforeBuiltinIO(t *testing.T) {
	for _, c := range []struct {
		kind, name, field string
		positive          bool
	}{
		{"action", "waitBlock", "target", false},
		{"action", "sendTx", "blocks", true},
		{"action", "load", "blocks", true},
		{"action", "load", "fillPercent", true},
		{"action", "readNodeLog", "maxBytes", true},
		{"assertion", "blockLag", "maxLag", false},
		{"assertion", "blockHalt", "maxAdvance", false},
		{"assertion", "blockInterval", "blocks", true},
		{"assertion", "blockInterval", "maxSeconds", false},
		{"assertion", "blockInterval", "minSeconds", false},
		{"assertion", "blockInterval", "maxMillis", false},
		{"assertion", "blockInterval", "minMillis", false},
		{"assertion", "wsSubscribe", "count", true},
		{"assertion", "wsCollected", "count", true},
		{"reader", "derive", "index", false},
		{"reader", "logs", "index", false},
		{"reader", "receiptLog", "index", false},
		{"reader", "receiptLog", "topic", false},
		{"reader", "createAddress", "nonce", false},
	} {
		bad := []any{1.5, -1, true, nil, "not-a-number", float64(9007199254740992), "18446744073709551616"}
		if c.positive {
			bad = append(bad, 0)
		}
		if c.field == "index" || c.field == "topic" || c.field == "count" || c.field == "maxBytes" {
			bad = append(bad, "18446744073709551615")
		}
		for i, value := range bad {
			t.Run(fmt.Sprintf("%s/%s/%s/%d", c.kind, c.name, c.field, i), func(t *testing.T) {
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
					case "eth_getLogs":
						result = []any{map[string]any{"data": "0x01", "address": testTo, "topics": []any{"0x11", "0x22"}}}
					case "eth_getTransactionReceipt":
						result = map[string]any{"status": "0x1", "logs": []any{map[string]any{"data": "0x01", "address": testTo, "topics": []any{"0x11", "0x22"}}}}
					case "eth_getBlockByNumber":
						result = map[string]any{"number": "0x3", "timestamp": "0x3", "gasLimit": "0xf4240", "baseFeePerGas": "0x1"}
					case "eth_sendTransaction":
						result = "0xh"
					case "eth_getTransactionCount":
						result = "0x0"
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
				}))
				defer server.Close()
				d := deps()
				control := &controlLogReader{}
				d.Nodes = control
				env := envWithNode(t, server.URL)
				args := map[string]any{"on": "node1", "from": testTo, "to": testTo, "value": "0", "gas": "21000", "wait": false, "expect": "keptOut", "within": "1ms", "timeout": "5ms", "pollInterval": "1ms", "op": "word", "of": []any{"0x" + strings.Repeat("0", 128)}, "select": "data", "hash": "0xh", "deployer": testTo}
				var sub *rpc.Subscription
				if c.name == "wsSubscribe" || c.name == "wsCollected" {
					args["select"] = ""
					base := wsHeadServerOf(t, []string{"0x1"})
					ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { opens.Add(1); base.Config.Handler.ServeHTTP(w, r) }))
					defer ws.Close()
					host, port := hostPort(t, ws.URL)
					env = envWithWS(t, host, port)
					if c.name == "wsCollected" {
						var err error
						subCtx, subCancel := context.WithCancel(context.Background())
						defer subCancel()
						sub, err = rpc.Subscribe(subCtx, strings.Replace(ws.URL, "http://", "ws://", 1), "newHeads")
						if err != nil {
							t.Fatal(err)
						}
						defer func() { _ = sub.Close() }()
						args["sub"] = sub
						opens.Store(0)
					}
				}
				if c.field == "topic" {
					delete(args, "select")
				}
				args[c.field] = value
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()
				var err error
				switch c.kind {
				case "action":
					action, _ := d.Actions.Action(c.name)
					err = action.Do(ctx, &interp.ActionCtx{Env: env, Deps: &d, Args: args})
				case "assertion":
					assertion, _ := d.Actions.Assertion(c.name)
					_, err = assertion.Check(ctx, &interp.AssertCtx{Env: env, Deps: &d, On: env.Nodes(), Spec: args})
				case "reader":
					reader, _ := d.Actions.Reader(c.name)
					_, err = reader(ctx, &d, rpc.Dial(server.URL), args)
				}
				if err == nil || !strings.Contains(err.Error(), c.field) {
					t.Errorf("explicit invalid control hidden or not identified: %v", err)
				}
				if requests.Load() != 0 || opens.Load() != 0 || control.reads.Load() != 0 {
					t.Errorf("invalid control reached IO: RPC=%d WS=%d logs=%d", requests.Load(), opens.Load(), control.reads.Load())
				}
			})
		}
	}
}

func TestUnsignedIndexOverflowCannotReachSliceSelection(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("unsigned index caused panic: %v", p)
		}
	}()
	for _, name := range []string{"derive", "logs", "receiptLog"} {
		reader, _ := Registry().Reader(name)
		_, err := reader(context.Background(), nil, nil, map[string]any{"op": "word", "of": []any{"0x" + strings.Repeat("0", 64)}, "index": uint64(math.MaxUint64), "hash": "0xh", "select": "data"})
		if err == nil || !strings.Contains(err.Error(), "index") {
			t.Errorf("overflow not rejected before dependency use: %s: %v", name, err)
		}
	}
}

func TestKeptOutFullWidthBlockCountCannotOverflowIntoSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var heads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any
			Method string
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var result any
		if req.Method == "eth_blockNumber" {
			result = "0x3"
			heads.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		if heads.Load() >= 3 {
			cancel()
		}
	}))
	defer server.Close()
	err := checkKeptOut(ctx, rpc.Dial(server.URL), "0xh", nil, &interp.ActionCtx{Args: map[string]any{"blocks": "18446744073709551615", "timeout": "1s", "pollInterval": "1ms"}})
	if err == nil || heads.Load() < 2 {
		t.Fatalf("unelapsed full-width window became success: %v heads=%d", err, heads.Load())
	}
}

func TestInvalidCollectedCountClosesTheExistingSubscription(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := wsHeadServerOf(t, nil)
	sub, err := rpc.Subscribe(ctx, strings.Replace(server.URL, "http://", "ws://", 1), "newHeads")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sub.Close() }()
	_, err = (wsCollectedAssertion{}).Check(context.Background(), &interp.AssertCtx{Spec: map[string]any{"sub": sub, "count": 1.5}})
	if err == nil || !strings.Contains(err.Error(), "count") {
		t.Fatal(err)
	}
	select {
	case _, open := <-sub.Notifications():
		if open {
			t.Fatal("invalid count left notifications open")
		}
	case <-time.After(time.Second):
		t.Fatal("invalid count did not release its subscription")
	}
}
