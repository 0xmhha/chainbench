package testspec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// metricNode points a node's metrics endpoint at a test server.
func metricNode(t *testing.T, body string) (node.Node, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return node.Node{Index: 1, Host: u.Hostname(), Ports: node.Endpoints{Metrics: port}}, srv.Close
}

func TestMetricAssertion(t *testing.T) {
	n, done := metricNode(t, "chain_head_block 42\n")
	defer done()

	as := metricAssertion{}

	// Default comparator is GreaterOrEqual: a floor check passes.
	r, err := as.Check(context.Background(), &AssertCtx{On: []node.Node{n}, Spec: map[string]any{
		"assert": assertMetric, "name": "chain_head_block", "expected": 10,
	}})
	if err != nil || !r.Pass {
		t.Fatalf("floor check: pass=%v err=%v (%v)", r.Pass, err, r.Actual)
	}
	if r.Actual != 42.0 {
		t.Fatalf("actual = %v, want 42", r.Actual)
	}

	// A failed comparison is a recorded failure, not an error.
	r, err = as.Check(context.Background(), &AssertCtx{On: []node.Node{n}, Spec: map[string]any{
		"assert": assertMetric, "name": "chain_head_block", "expected": 100,
	}})
	if err != nil {
		t.Fatalf("comparison failure must not error: %v", err)
	}
	if r.Pass {
		t.Fatal("42 >= 100 must fail")
	}

	// An unknown sample is an explicit error.
	if r, _ := as.Check(context.Background(), &AssertCtx{On: []node.Node{n}, Spec: map[string]any{
		"assert": assertMetric, "name": "no_such_metric", "expected": 1,
	}}); r.Pass {
		t.Fatal("missing metric must fail")
	}
}

func TestMetricAssertionRequiresPortAndName(t *testing.T) {
	as := metricAssertion{}

	// No metrics port: explicit failure naming the launch condition.
	r, err := as.Check(context.Background(), &AssertCtx{
		On:   []node.Node{{Index: 1, Host: "127.0.0.1"}},
		Spec: map[string]any{"assert": assertMetric, "name": "x", "expected": 1},
	})
	if err == nil || r.Pass {
		t.Fatalf("node without a metrics port must fail: pass=%v err=%v", r.Pass, err)
	}

	// Missing name.
	if _, err := as.Check(context.Background(), &AssertCtx{
		On:   []node.Node{{Index: 1}},
		Spec: map[string]any{"assert": assertMetric},
	}); err == nil {
		t.Fatal("missing name must fail")
	}
}
