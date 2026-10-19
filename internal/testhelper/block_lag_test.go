package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type lagRoundTripper func(*http.Request) (*http.Response, error)

func (f lagRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func lagRPCResponse(t *testing.T, r *http.Request, value string) (*http.Response, error) {
	t.Helper()
	var q struct {
		ID     any
		Method string
	}
	if e := json.NewDecoder(r.Body).Decode(&q); e != nil {
		return nil, e
	}
	if q.Method != "eth_blockNumber" {
		return nil, fmt.Errorf("unexpected RPC %s", q.Method)
	}
	b, e := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": value})
	if e != nil {
		return nil, e
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(b)))}, nil
}

func TestPublicFaultProgressUsesOriginalDeadline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		budget time.Duration
	}{{"CT-FAULT-001-producer-crash-and-restart.json", 60 * time.Second}, {"CT-FAULT-002-node-recover-and-sync.json", 90 * time.Second}} {
		t.Run(tc.name, func(t *testing.T) {
			raw, e := dsl.ReadFiles([]string{"../../tests/tc/common/fault/" + tc.name})
			if e != nil {
				t.Fatal(e)
			}
			spec, e := dsl.ParseV2(raw[0])
			if e != nil {
				t.Fatal(e)
			}
			var args map[string]any
			for _, s := range spec.Sequence {
				if s.Do == "waitBlock" && s.Args["on"] == "node1" {
					args = map[string]any{}
					for k, v := range s.Args {
						args[k] = v
					}
					args["target"] = uint64(10)
				}
			}
			if args == nil {
				t.Fatal("missing public progress wait")
			}
			called := 0
			d := deps()
			d.RPC = func(u string) *rpc.Client {
				return rpc.DialWithClient(u, &http.Client{Transport: lagRoundTripper(func(r *http.Request) (*http.Response, error) {
					called++
					deadline, ok := r.Context().Deadline()
					remaining := time.Until(deadline)
					if !ok || remaining > tc.budget+100*time.Millisecond || remaining < tc.budget-time.Second {
						t.Errorf("public progress deadline %s, want %s", remaining, tc.budget)
					}
					return lagRPCResponse(t, r, "0x64")
				})})
			}
			env := envWithNode(t, "http://primary")
			e = (waitBlockAction{}).Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: args})
			if e != nil || called != 1 {
				t.Fatal("deadline probe did not reach runtime RPC", called, e)
			}
		})
	}
}

func TestBlockLagFreshHeadsAndDirectionalReference(t *testing.T) {
	assertion, ok := Registry().Assertion("blockLag")
	if !ok {
		t.Fatal("missing canonical blockLag assertion")
	}
	for _, tc := range []struct {
		name         string
		heads        [][]uint64
		reference    string
		target       []string
		pass         bool
		minimumReads int
		missing      int
	}{
		{name: "all nodes converge", heads: [][]uint64{{100, 105}, {99, 104}, {10, 104}}, pass: true, minimumReads: 2},
		{name: "moving reference escapes stale goal", heads: [][]uint64{{100, 110, 120, 130}, {90, 98, 100, 101}}, reference: "node1", target: []string{"node2"}, pass: false, minimumReads: 2},
		{name: "reference selector alias", heads: [][]uint64{{100}, {99}}, reference: "bp1", target: []string{"node2"}, pass: true, minimumReads: 1},
		{name: "directional recovered ahead allowed", heads: [][]uint64{{10}, {30}}, reference: "node1", target: []string{"node2"}, pass: true, minimumReads: 1},
		{name: "explicit second reference", heads: [][]uint64{{1}, {100}, {99}}, reference: "node2", target: []string{"node3"}, pass: true, minimumReads: 1},
		{name: "zero is not recovered", heads: [][]uint64{{0}, {0}}, pass: false, minimumReads: 2},
		{name: "missing endpoint cannot be ignored", heads: [][]uint64{{100}, {100}, {100}}, pass: false, missing: 3},
		{name: "reference must exist", heads: [][]uint64{{100}, {100}}, reference: "node9", target: []string{"node2"}, pass: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			reads := make([]int, len(tc.heads))
			urls := make([]string, len(reads))
			for i := range urls {
				urls[i] = fmt.Sprintf("http://n%d", i+1)
				if tc.missing == i+1 {
					urls[i] = ""
				}
			}
			env := envOf(t, urls...)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			d := deps()
			d.RPC = func(u string) *rpc.Client {
				return rpc.DialWithClient(u, &http.Client{Transport: lagRoundTripper(func(r *http.Request) (*http.Response, error) {
					var index int
					if _, e := fmt.Sscanf(r.URL.Host, "n%d", &index); e != nil || index < 1 || index > len(reads) {
						return nil, fmt.Errorf("bad test endpoint %s", r.URL.Host)
					}
					mu.Lock()
					i := index - 1
					n := reads[i]
					reads[i]++
					if n >= len(tc.heads[i]) {
						n = len(tc.heads[i]) - 1
					}
					height := tc.heads[i][n]
					stop := !tc.pass && reads[i] >= 5
					mu.Unlock()
					if stop {
						cancel()
					}
					return lagRPCResponse(t, r, fmt.Sprintf("0x%x", height))
				})})
			}
			args := map[string]any{"maxLag": "2", "timeout": "1s", "pollInterval": "1ms"}
			if tc.reference != "" {
				args["reference"] = tc.reference
			}
			ac := &interp.AssertCtx{Deps: &d, Env: env, Spec: args}
			if len(tc.target) > 0 {
				var e error
				ac.On, e = env.ResolveEach(tc.target)
				if e != nil {
					t.Fatal(e)
				}
			}
			result, e := assertion.Check(ctx, ac)
			if tc.pass {
				if e != nil || !result.Pass {
					t.Fatalf("valid recovery rejected: %+v %v", result, e)
				}
			} else if result.Pass {
				t.Fatalf("invalid recovery accepted: %+v", result)
			}
			if tc.minimumReads > 0 {
				mu.Lock()
				n := reads[0]
				if tc.reference == "node2" {
					n = reads[1]
				}
				mu.Unlock()
				if n < tc.minimumReads {
					t.Errorf("fresh reference not sampled: %v", reads)
				}
			}
		})
	}
}

func TestBlockLagRejectsInvalidControlsBeforeIO(t *testing.T) {
	assertion, _ := Registry().Assertion("blockLag")
	for _, tc := range []struct {
		field string
		value any
	}{
		{"timeout", "0s"}, {"pollInterval", "-1s"}, {"maxLag", -1}, {"maxLag", true}, {"maxLag", nil}, {"maxLag", 1.5}, {"maxLag", "18446744073709551616"}, {"reference", ""}, {"reference", true},
	} {
		t.Run(fmt.Sprintf("%s=%v", tc.field, tc.value), func(t *testing.T) {
			d := deps()
			calls := 0
			d.RPC = func(u string) *rpc.Client { calls++; return rpc.Dial(u) }
			args := map[string]any{"maxLag": "2", "timeout": "1s", "pollInterval": "1ms"}
			args[tc.field] = tc.value
			result, e := assertion.Check(context.Background(), &interp.AssertCtx{Env: envOf(t, "http://n1", "http://n2"), Deps: &d, Spec: args})
			if e == nil || result.Pass || calls != 0 || result.Actual != e.Error() {
				t.Fatalf("invalid control reached IO or passed: %+v %v calls=%d", result, e, calls)
			}
		})
	}
}

func TestBlockLagBoundsObservationHistory(t *testing.T) {
	assertion, _ := Registry().Assertion("blockLag")
	d := deps()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	d.RPC = func(u string) *rpc.Client {
		return rpc.DialWithClient(u, &http.Client{Transport: lagRoundTripper(func(r *http.Request) (*http.Response, error) {
			if r.URL.Host == "n1" {
				calls++
				if calls == 140 {
					cancel()
				}
			}
			value := "0x64"
			if r.URL.Host == "n2" {
				value = "0x1"
			}
			return lagRPCResponse(t, r, value)
		})})
	}
	result, e := assertion.Check(ctx, &interp.AssertCtx{Env: envOf(t, "http://n1", "http://n2"), Deps: &d, Spec: map[string]any{"timeout": "2s", "pollInterval": "1ns"}})
	if result.Pass || e == nil {
		t.Fatal("divergence passed", result, e)
	}
	actual := result.Actual.(map[string]any)
	samples := actual["samples"].([]lagSample)
	if actual["attempts"] != 140 || len(samples) != 128 || actual["droppedSamples"] != 12 {
		t.Fatal("unbounded or hidden truncated diagnostics", actual)
	}
}

func TestPublicFaultRecoveryUsesOriginalDeadline(t *testing.T) {
	for _, tc := range []struct {
		name   string
		budget time.Duration
		calls  int
	}{
		{"CT-FAULT-001-producer-crash-and-restart.json", 90 * time.Second, 4},
		{"CT-FAULT-002-node-recover-and-sync.json", 120 * time.Second, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, e := dsl.ReadFiles([]string{"../../tests/tc/common/fault/" + tc.name})
			if e != nil {
				t.Fatal(e)
			}
			spec, e := dsl.ParseV2(raw[0])
			if e != nil {
				t.Fatal(e)
			}
			var args map[string]any
			for _, s := range spec.Sequence {
				if s.Expect == "blockLag" {
					if args != nil {
						t.Fatal("duplicate recovery wait")
					}
					args = s.Args
				}
			}
			if args == nil {
				t.Fatal("missing live recovery wait")
			}
			env := envOf(t, "http://n1", "http://n2", "http://n3", "http://n4")
			ac := &interp.AssertCtx{Env: env, Spec: args}
			if tc.calls == 2 {
				if args["reference"] != "node1" || args["on"] != "node3" {
					t.Fatal("wrong primary/recovered roles", args)
				}
				ac.On, e = env.ResolveEach([]string{"node3"})
			} else {
				ac.On, e = env.ResolveEach([]string{"node1", "node2", "node3", "node4"})
			}
			if e != nil {
				t.Fatal(e)
			}
			called := 0
			d := deps()
			d.RPC = func(u string) *rpc.Client {
				return rpc.DialWithClient(u, &http.Client{Transport: lagRoundTripper(func(r *http.Request) (*http.Response, error) {
					called++
					deadline, ok := r.Context().Deadline()
					remaining := time.Until(deadline)
					if !ok || remaining > tc.budget+100*time.Millisecond || remaining < tc.budget-time.Second {
						t.Errorf("public recovery deadline %s, want %s", remaining, tc.budget)
					}
					return lagRPCResponse(t, r, "0x64")
				})})
			}
			ac.Deps = &d
			assertion, _ := d.Actions.Assertion("blockLag")
			result, e := assertion.Check(context.Background(), ac)
			if e != nil || !result.Pass || called != tc.calls {
				t.Fatal("recovery deadline did not reach selected runtime targets", called, result, e)
			}
		})
	}
}
