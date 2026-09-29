package testhelper

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/assert"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
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

func (metricAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertMetric, Provenance: ac.Spec, Pass: true}
	name, _ := ac.Spec["name"].(string)
	if name == "" {
		err := fmt.Errorf("dsl: metric: \"name\" is required")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	op := "GreaterOrEqual"
	if o, ok := ac.Spec["compare"].(string); ok && o != "" {
		op = o
	}
	fn, ok := assert.Lookup(op)
	if !ok {
		return res, fmt.Errorf("dsl: unknown comparator %q", op)
	}
	spec, rerr := resolveNamedArgs(ac.Deps, ac.Spec)
	if rerr != nil {
		res.Pass, res.Actual = false, rerr.Error()
		return res, rerr
	}
	expected := spec["expected"]
	res.Expected = expected

	targets := metricTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: metric: no target node")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	for _, t := range targets {
		// The composition records the endpoint this tool should dial, already
		// translated for the target (a docker node publishes on loopback).
		// Host+Ports is the node's OWN address, which is what peers use; building
		// the URL from it is how this assertion used to dial a container-internal
		// address and time out. Fall back to that pair only when nothing recorded
		// an endpoint — an attached node, or a plain local run.
		url := t.node.MetricsURL
		if url == "" {
			if t.node.Ports.Metrics == 0 {
				err := fmt.Errorf("dsl: metric: %s has no metrics port — was it launched with --metrics?", t.name)
				res.Pass, res.Actual = false, err.Error()
				return res, err
			}
			url = collector.MetricsURL(t.node.Host, t.node.Ports.Metrics)
		}
		samples, err := collector.ScrapeMetrics(ctx, url)
		if err != nil {
			samples, err = scrapeOnMachine(ctx, ac.Deps, t.node, err)
		}
		if err != nil {
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		v, ok := samples[name]
		if !ok {
			err := fmt.Errorf("dsl: metric: %s does not expose %q", t.name, name)
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

// onMachineScrapeTimeoutSeconds bounds the request a scrape on the node's
// machine makes, so a metrics endpoint that accepts and never answers cannot
// hold the step.
const onMachineScrapeTimeoutSeconds = 10

// scrapeOnMachine reads a node's metrics on the machine it runs on, for when
// the harness cannot reach the endpoint itself: a docker or remote node whose
// metrics port is not published. The request goes to the node's own address,
// which is reachable from its machine, through curl when the machine has it
// and bash's /dev/tcp otherwise (a minimal image carries neither curl nor
// wget). A run that cannot reach the machine keeps the scrape error.
func scrapeOnMachine(ctx context.Context, d *interp.Deps, n node.Node, scrapeErr error) (map[string]float64, error) {
	if d == nil || n.Ports.Metrics == 0 {
		return nil, scrapeErr
	}
	cmd, ok := d.Nodes.(interp.HostCommander)
	if !ok {
		return nil, scrapeErr
	}
	const path = "/debug/metrics/prometheus"
	url := collector.MetricsURL(n.Host, n.Ports.Metrics)
	command := fmt.Sprintf(
		"if command -v curl >/dev/null 2>&1; then timeout %d curl -s %s; "+
			"else timeout %d bash -c 'exec 3<>/dev/tcp/%s/%d && printf \"GET %s HTTP/1.0\\r\\nHost: %s\\r\\n\\r\\n\" >&3 && cat <&3'; fi",
		onMachineScrapeTimeoutSeconds, url,
		onMachineScrapeTimeoutSeconds, n.Host, n.Ports.Metrics, path, n.Host)
	out, err := cmd.RunOnHost(ctx, n, command)
	if err != nil {
		return nil, fmt.Errorf("%w; on node%d's machine: %v", scrapeErr, n.Index, err)
	}
	samples, err := collector.ParseMetrics(strings.NewReader(httpBody(out)))
	if err != nil {
		return nil, fmt.Errorf("%w; on node%d's machine: %v", scrapeErr, n.Index, err)
	}
	return samples, nil
}

// httpBody drops a raw HTTP response's status line and headers, leaving the
// body; output that is not a response (curl prints only the body) is returned
// as it is.
func httpBody(out string) string {
	if !strings.HasPrefix(out, "HTTP/") {
		return out
	}
	for _, sep := range []string{"\r\n\r\n", "\n\n"} {
		if i := strings.Index(out, sep); i >= 0 {
			return out[i+len(sep):]
		}
	}
	return ""
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
func metricTargets(ac *interp.AssertCtx) []metricTarget {
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
