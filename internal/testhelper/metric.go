package testhelper

import (
	"context"
	"fmt"
	"maps"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

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
// Spec: name (the Prometheus sample name, required), alternatives (optional
// additional sample names; at least one must satisfy the comparison), expected, compare
// (default GreaterOrEqual — counters and heights grow, so a floor is the
// common check), on/onEach for targeting. The target node must have been
// launched with a metrics port; a node without one is an explicit failure,
// not a skip, because "metrics silently off" is the defect class the
// launchopt Metrics module also guards against.
type metricAssertion struct{}

func (metricAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertMetric, Provenance: maps.Clone(ac.Spec), Pass: true}
	if res.Provenance == nil {
		res.Provenance = map[string]any{}
	}
	observed := []map[string]any{}
	res.Provenance["metricSamples"] = observed
	name, _ := ac.Spec["name"].(string)
	if name == "" {
		err := fmt.Errorf("dsl: metric: \"name\" is required")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	names := []string{name}
	if value, exists := ac.Spec["alternatives"]; exists {
		values, err := metricAlternatives(value, name)
		if err != nil {
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		names = append(names, values...)
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
		source := "recorded-endpoint"
		if _, ok := machineReader(ac.Deps, t.node); ok {
			source = "node-machine"
		} else if t.node.MetricsURL == "" {
			source = "derived-endpoint"
		}
		observations := make([]map[string]any, len(names))
		for i, sampleName := range names {
			observations[i] = map[string]any{"node": t.node.Index, "role": string(t.node.Role), "pid": t.node.PID, "source": source, "name": sampleName, "observedAt": time.Now().UTC().Format(time.RFC3339Nano), "status": "unavailable"}
			observed = append(observed, observations[i])
		}
		res.Provenance["metricSamples"] = observed
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
		samples, err := scrapeNode(ctx, ac.Deps, t.node, url)
		if err != nil {
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		matched := false
		var detail string
		for i, sampleName := range names {
			v, ok := samples[sampleName]
			if !ok {
				err := fmt.Errorf("dsl: metric: %s does not expose %q", t.name, sampleName)
				res.Pass, res.Actual = false, err.Error()
				return res, err
			}
			if math.IsInf(v, 0) || math.IsNaN(v) {
				err := fmt.Errorf("dsl: metric: %s has a nonfinite sample for %q", t.name, sampleName)
				res.Pass, res.Actual = false, err.Error()
				return res, err
			}
			observations[i]["value"], observations[i]["status"] = v, "observed"
			pass, sampleDetail := fn(v, expected)
			if pass && !matched {
				res.Actual = v
			}
			matched = matched || pass
			detail = sampleDetail
		}
		if !matched {
			res.Pass = false
			res.Actual = fmt.Sprintf("%s: %s", t.name, detail)
			return res, nil
		}
	}
	return res, nil
}

// metricAlternatives names additional receive paths whose same comparator is
// satisfied by at least one sample. All declared samples must be exposed and
// finite; an unavailable alternative never turns a telemetry error into a pass.
func metricAlternatives(value any, primary string) ([]string, error) {
	var values []any
	switch list := value.(type) {
	case []any:
		values = list
	case []string:
		for _, name := range list {
			values = append(values, name)
		}
	}
	if len(values) == 0 || len(values) > 31 {
		return nil, fmt.Errorf("dsl: metric: alternatives must contain 1 to 31 distinct names")
	}
	seen := map[string]bool{primary: true}
	names := make([]string, 0, len(values))
	for _, value := range values {
		name, ok := value.(string)
		if !ok || strings.TrimSpace(name) == "" || seen[name] {
			return nil, fmt.Errorf("dsl: metric: alternatives must contain nonempty distinct names, excluding name")
		}
		seen[name] = true
		names = append(names, name)
	}
	return names, nil
}

// onMachineScrapeTimeoutSeconds bounds the request a scrape on the node's
// machine makes, so a metrics endpoint that accepts and never answers cannot
// hold the step.
const onMachineScrapeTimeoutSeconds = 10

// scrapeNode reads a node's metrics, on its machine first when the node runs
// elsewhere and the run can reach that machine, else at url.
//
// The order matters. A docker node's recorded URL is loopback with the node's
// own metrics port, which nothing publishes; whatever answers there on the
// harness is some other process, and its numbers would pass for the node's.
// The machine is asked first so that cannot happen. A local node, or a run
// that cannot reach the machine, reads url as before.
func scrapeNode(ctx context.Context, d *interp.Deps, n node.Node, url string) (map[string]float64, error) {
	cmd, ok := machineReader(d, n)
	if !ok {
		return scrapeWithRunClient(ctx, d, url)
	}
	samples, err := scrapeOnMachine(ctx, cmd, n)
	if err != nil {
		return nil, fmt.Errorf("on node%d's machine: %w", n.Index, err)
	}
	return samples, nil
}

// machineReader returns the run's reach into n's machine when n runs on
// another machine and has a metrics port.
func machineReader(d *interp.Deps, n node.Node) (interp.HostCommander, bool) {
	if d == nil || n.Ports.Metrics == 0 || isLoopbackHost(n.Host) {
		return nil, false
	}
	cmd, ok := d.Nodes.(interp.HostCommander)
	return cmd, ok
}

// isLoopbackHost reports whether host names this machine.
func isLoopbackHost(host string) bool {
	switch host {
	case "", "localhost", "127.0.0.1", "::1":
		return true
	}
	return strings.HasPrefix(host, "127.")
}

// scrapeOnMachine reads n's metrics on the machine it runs on, at the node's
// own address, which is reachable from there. The request goes through curl
// when the machine has it and bash's /dev/tcp otherwise (a minimal image
// carries neither curl nor wget).
func scrapeOnMachine(ctx context.Context, cmd interp.HostCommander, n node.Node) (map[string]float64, error) {
	const path = "/debug/metrics/prometheus"
	url := collector.MetricsURL(n.Host, n.Ports.Metrics)
	command := fmt.Sprintf(
		"if command -v curl >/dev/null 2>&1; then timeout %d curl -fsS %s; "+
			"else timeout %d bash -c 'exec 3<>/dev/tcp/%s/%d && printf \"GET %s HTTP/1.0\\r\\nHost: %s\\r\\n\\r\\n\" >&3 && cat <&3'; fi",
		onMachineScrapeTimeoutSeconds, url,
		onMachineScrapeTimeoutSeconds, n.Host, n.Ports.Metrics, path, n.Host)
	out, err := cmd.RunOnHost(ctx, n, command)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(out, "HTTP/") {
		status := strings.Fields(strings.SplitN(out, "\n", 2)[0])
		if len(status) < 2 {
			return nil, fmt.Errorf("metric: malformed on-machine HTTP response")
		}
		code, err := strconv.Atoi(status[1])
		if err != nil || code != http.StatusOK {
			return nil, fmt.Errorf("metric: unsuccessful on-machine HTTP response")
		}
		body := httpBody(out)
		if body == "" {
			return nil, fmt.Errorf("metric: empty on-machine HTTP response")
		}
		out = body
	}
	return collector.ParseMetrics(strings.NewReader(out))
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

func scrapeWithRunClient(ctx context.Context, d *interp.Deps, url string) (map[string]float64, error) {
	if d != nil && d.HTTP != nil {
		return collector.ScrapeMetricsWithClient(ctx, url, d.HTTP(url))
	}
	return collector.ScrapeMetrics(ctx, url)
}
