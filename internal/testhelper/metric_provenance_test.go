package testhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestMetricMachineFailureCannotPassUsingAnUnrelatedLoopback(t *testing.T) {
	stranger, done := metricNode(t, "eth_downloader_headers_in 99\n")
	defer done()
	n := unreachableMetricsNode()
	n.MetricsURL = stranger.MetricsURL
	if n.MetricsURL == "" {
		n.MetricsURL = collector.MetricsURL(stranger.Host, stranger.Ports.Metrics)
	}
	ctrl := &hostControl{err: errors.New("machine unavailable")}
	result, err := metricAssertion{}.Check(t.Context(), &interp.AssertCtx{Deps: &interp.Deps{Nodes: ctrl}, On: []node.Node{n}, Spec: map[string]any{"name": "eth_downloader_headers_in", "expected": 1}})
	if err == nil || result.Pass {
		t.Fatal("another process's counter passed as the remote node", result, err)
	}
}

func TestMetricOnMachineHTTPFailureCannotPassAsCounterData(t *testing.T) {
	ctrl := &hostControl{out: "HTTP/1.0 500 Internal Server Error\r\n\r\neth_downloader_headers_in 99\n"}
	result, err := metricAssertion{}.Check(context.Background(), &interp.AssertCtx{Deps: &interp.Deps{Nodes: ctrl}, On: []node.Node{unreachableMetricsNode()}, Spec: map[string]any{"name": "eth_downloader_headers_in", "expected": 1}})
	if err == nil || result.Pass {
		t.Fatal("HTTP failure body counted as node telemetry", result, err)
	}
}

func TestMetricNonfiniteValuesCannotSatisfyAnAssertion(t *testing.T) {
	n, done := metricNode(t, "counter +Inf\n")
	defer done()
	result, err := metricAssertion{}.Check(t.Context(), &interp.AssertCtx{On: []node.Node{n}, Spec: map[string]any{"name": "counter", "expected": 1}})
	if err == nil || result.Pass {
		t.Fatal("nonfinite counter was accepted", result, err)
	}
}

func TestMetricRecordsNodeAndTransportWithoutChangingTheDeclaration(t *testing.T) {
	n, done := metricNode(t, "eth_downloader_headers_in 7\n")
	defer done()
	n.PID = 123
	spec := map[string]any{"name": "eth_downloader_headers_in", "expected": 1}
	result, err := metricAssertion{}.Check(t.Context(), &interp.AssertCtx{On: []node.Node{n}, Spec: spec})
	if err != nil || !result.Pass {
		t.Fatal(err, result)
	}
	samples, ok := result.Provenance["metricSamples"].([]map[string]any)
	if !ok || len(samples) != 1 || samples[0]["node"] != 1 || samples[0]["pid"] != 123 || samples[0]["source"] != "derived-endpoint" || samples[0]["value"] != float64(7) {
		t.Fatal("metric origin is absent", result.Provenance)
	}
	if _, changed := spec["metricSamples"]; changed {
		t.Fatal("observation mutated the input declaration")
	}
}

func TestMetricAlternativeReceivePathsRequireFiniteRecordedSamples(t *testing.T) {
	for _, test := range []struct {
		name, body string
		pass       bool
	}{
		{"broadcast", "broadcast 2\nannounce 0\n", true},
		{"hash-announcement", "broadcast 0\nannounce 3\n", true},
		{"neither", "broadcast 0\nannounce 0\n", false},
		{"missing-alternative", "broadcast 2\n", false},
		{"nonfinite-alternative", "broadcast 2\nannounce +Inf\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			n, done := metricNode(t, test.body)
			defer done()
			result, err := metricAssertion{}.Check(t.Context(), &interp.AssertCtx{On: []node.Node{n}, Spec: map[string]any{"name": "broadcast", "alternatives": []any{"announce"}, "expected": 1}})
			if result.Pass != test.pass || (test.pass && err != nil) {
				t.Fatalf("alternative receive path: %+v %v", result, err)
			}
			if test.pass {
				samples, ok := result.Provenance["metricSamples"].([]map[string]any)
				if !ok || len(samples) != 2 || samples[0]["name"] != "broadcast" || samples[1]["name"] != "announce" {
					t.Fatal("both paths must retain their own observations", result.Provenance)
				}
			}
		})
	}
}

func TestMetricAlternativeDeclarationsAreConstrained(t *testing.T) {
	n, done := metricNode(t, "counter 2\nother 2\n")
	defer done()
	for _, value := range []any{[]any{}, []any{""}, []any{3}, []any{"counter"}, []any{"other", "other"}, "other"} {
		result, err := metricAssertion{}.Check(t.Context(), &interp.AssertCtx{On: []node.Node{n}, Spec: map[string]any{"name": "counter", "alternatives": value, "expected": 1}})
		if err == nil || result.Pass {
			t.Fatal("invalid alternative declaration accepted", value, result, err)
		}
	}
}
