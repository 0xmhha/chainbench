package testhelper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// hostControl is a node control that runs commands on a node's machine
// (interp.HostCommander) and answers with a fixed output.
type hostControl struct {
	fakeNodeControl
	out     string
	err     error
	command string
	on      node.Node
}

func (c *hostControl) RunOnHost(_ context.Context, n node.Node, command string) (string, error) {
	c.command, c.on = command, n
	return c.out, c.err
}

// unreachableMetricsNode is a node whose recorded scrape URL does not answer:
// a docker or remote node whose metrics port is not published to the harness.
func unreachableMetricsNode() node.Node {
	n := node.Node{Index: 5, Host: "172.30.0.15", MetricsURL: "http://127.0.0.1:1/debug/metrics/prometheus"}
	n.Ports.Metrics = 6060
	return n
}

// TestMetricAssertion_ReadsOnTheNodesMachineWhenTheHarnessCannot: when the
// recorded URL does not answer and the run can reach the node's machine, the
// sample is read there, from the node's own address.
func TestMetricAssertion_ReadsOnTheNodesMachineWhenTheHarnessCannot(t *testing.T) {
	ctrl := &hostControl{out: "HTTP/1.0 200 OK\r\nContent-Type: text/plain\r\n\r\neth_downloader_headers_in 7\n"}
	d := interp.Deps{Nodes: ctrl}
	n := unreachableMetricsNode()
	r, err := metricAssertion{}.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{n}, Spec: map[string]any{
		"assert": assertMetric, "name": "eth_downloader_headers_in", "expected": 1,
	}})
	if err != nil || !r.Pass {
		t.Fatalf("pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
	}
	if r.Actual != 7.0 {
		t.Errorf("actual = %v, want 7", r.Actual)
	}
	if ctrl.on.Index != 5 || !strings.Contains(ctrl.command, "172.30.0.15") || !strings.Contains(ctrl.command, "6060") {
		t.Errorf("read on node%d with %q, want node5 at its own address and metrics port", ctrl.on.Index, ctrl.command)
	}
}

// TestMetricAssertion_WithoutAMachineTheScrapeErrorStands: a run that cannot
// reach the machine (attach mode, a local composition) reports the scrape
// failure as before rather than inventing a second path.
func TestMetricAssertion_WithoutAMachineTheScrapeErrorStands(t *testing.T) {
	d := interp.Deps{Nodes: &fakeNodeControl{}}
	r, err := metricAssertion{}.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{unreachableMetricsNode()}, Spec: map[string]any{
		"assert": assertMetric, "name": "x", "expected": 1,
	}})
	if err == nil || r.Pass {
		t.Fatalf("an unreachable endpoint with no machine access gave pass=%v err=%v", r.Pass, err)
	}
}

// TestMetricAssertion_BothPathsFailingNamesBoth: when the machine cannot be
// read either, the error says what each path answered.
func TestMetricAssertion_BothPathsFailingNamesBoth(t *testing.T) {
	ctrl := &hostControl{err: errors.New("ssh: connection refused")}
	d := interp.Deps{Nodes: ctrl}
	_, err := metricAssertion{}.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{unreachableMetricsNode()}, Spec: map[string]any{
		"assert": assertMetric, "name": "x", "expected": 1,
	}})
	if err == nil || !strings.Contains(err.Error(), "ssh: connection refused") || !strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Fatalf("err = %v, want both the scrape and the on-machine failure", err)
	}
}
