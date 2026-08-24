package testspec

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec/assert"
)

// assertMetric is the metric-source assertion name — the third verification
// source next to log and rpc (background requirement #3).
const assertMetric = "metric"

// metricAssertion scrapes a node's --metrics endpoint and compares one sample
// to the spec's expected value.
//
// Spec: name (the Prometheus sample name, required), expected, compare
// (default GreaterOrEqual — counters and heights grow, so a floor is the
// common check), on/onEach for targeting. The target node must have been
// launched with a metrics port; a node without one is an explicit failure,
// not a skip, because "metrics silently off" is the defect class the
// launchopt Metrics module also guards against.
type metricAssertion struct{}

func (metricAssertion) Check(ctx context.Context, ac *AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertMetric, Provenance: ac.Spec, Pass: true}
	name, _ := ac.Spec["name"].(string)
	if name == "" {
		err := fmt.Errorf("testspec: metric: \"name\" is required")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	op := "GreaterOrEqual"
	if o, ok := ac.Spec["compare"].(string); ok && o != "" {
		op = o
	}
	fn, ok := assert.Lookup(op)
	if !ok {
		return res, fmt.Errorf("testspec: unknown comparator %q", op)
	}
	expected := ac.Spec["expected"]
	res.Expected = expected

	targets := metricTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("testspec: metric: no target node")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	for _, t := range targets {
		if t.node.Ports.Metrics == 0 {
			err := fmt.Errorf("testspec: metric: %s has no metrics port — was it launched with --metrics?", t.name)
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		samples, err := collector.ScrapeMetrics(ctx, collector.MetricsURL(t.node.Host, t.node.Ports.Metrics))
		if err != nil {
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		v, ok := samples[name]
		if !ok {
			err := fmt.Errorf("testspec: metric: %s does not expose %q", t.name, name)
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		res.Actual = v
		if pass, detail := fn(v, expected); !pass {
			res.Pass = false
			res.Actual = fmt.Sprintf("%s: %s", t.name, detail)
			return res, nil
		}
	}
	return res, nil
}

// metricTarget pairs a target node with its display name.
type metricTarget struct {
	name string
	node node.Node
}

// metricTargets are the nodes the assertion checks: every resolved "on"/
// "onEach" node, else the environment's primary node. Unlike assertTargets it
// keeps the full node (host + ports), because the scrape URL is derived from
// the metrics port, not the RPC URL.
func metricTargets(ac *AssertCtx) []metricTarget {
	if len(ac.On) > 0 {
		out := make([]metricTarget, 0, len(ac.On))
		for _, n := range ac.On {
			out = append(out, metricTarget{name: fmt.Sprintf("node%d", n.Index), node: n})
		}
		return out
	}
	if ac.Env != nil {
		if nodes := ac.Env.Nodes(); len(nodes) > 0 {
			return []metricTarget{{name: fmt.Sprintf("node%d", nodes[0].Index), node: nodes[0]}}
		}
	}
	return nil
}
