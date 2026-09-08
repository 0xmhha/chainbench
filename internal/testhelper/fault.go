package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/dsl/interp"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Fault and node-lifecycle action names (design §3.2). These are what a
// destructive test uses: stop a validator to probe quorum, restart it to check
// sync recovery, or split the network to induce a fork.
const (
	actionStopNode      = "stopNode"
	actionStartNode     = "startNode"
	actionRestartNode   = "restartNode"
	actionSwapNode      = "swapNode"
	actionPartition     = "partition"
	actionHealPartition = "healPartition"
	actionReadNodeLog   = "readNodeLog"
)

// Bounds for the node-lifecycle vocabulary. A step that expects a node NOT to
// come up has to wait long enough to be sure, and a log read has to stop
// somewhere: a node that has been running for an hour has a log no assertion
// wants in full.
const (
	// nodeDownProbeTimeout is how long a step that expects a failed launch
	// keeps probing before it accepts that the node is down. A node that boots
	// normally answers JSON-RPC well inside this.
	nodeDownProbeTimeout = 20 * time.Second
	// nodeDownProbeInterval is the gap between those probes.
	nodeDownProbeInterval = time.Second
	// nodeLogDefaultMaxBytes is how much of a log's tail readNodeLog binds when
	// the step does not say. Enough for a startup failure's message and stack,
	// small enough to keep an artifact readable.
	nodeLogDefaultMaxBytes = 64 * 1024
)

// seedFaultBuiltins registers the node-lifecycle and partition actions.
func seedFaultBuiltins(r interp.Registry) {
	r.RegisterAction(actionStopNode, stopNodeAction{})
	r.RegisterAction(actionStartNode, startNodeAction{})
	r.RegisterAction(actionRestartNode, restartNodeAction{})
	r.RegisterAction(actionSwapNode, swapNodeAction{})
	r.RegisterAction(actionPartition, partitionAction{})
	r.RegisterAction(actionHealPartition, healPartitionAction{})
	r.RegisterAction(actionReadNodeLog, readNodeLogAction{})
}

// expectsNodeDown reports whether a node-lifecycle step declares that the node
// must NOT come up: expect:"fail" or expectFail:true. It mirrors sendTx's
// expect:"reject" so one grammar covers both kinds of expected failure — a
// transaction the node refuses, and a launch the node refuses.
func expectsNodeDown(args map[string]any) bool {
	if b, ok := args["expectFail"].(bool); ok {
		return b
	}
	if s, ok := args["expect"].(string); ok {
		return strings.EqualFold(s, "fail")
	}
	return false
}

// confirmNodeDown enforces expect:"fail" on a launch: the node must not answer
// JSON-RPC within nodeDownProbeTimeout. A launcher error is not enough on its
// own, because a node can start and then exit — the process manager sees a
// clean launch and the failure is only in the log — so this probes the endpoint
// instead of trusting the return value.
//
// The evidence it gathers (the launcher's error, plus the node's log tail when
// the control can read one) is bound under "save" and matched against an
// optional "reason" (case-insensitive substring), so a spec can require the
// specific rejection it expects rather than any failure at all.
func confirmNodeDown(ctx context.Context, ac *interp.ActionCtx, n node.Node,
	ctrl interp.NodeControl, launchErr error, action string) error {
	evidence := nodeFailureEvidence(ctx, ctrl, n, launchErr)
	deadline := time.Now().Add(nodeDownProbeTimeout)
	for nodeAnswers(ctx, ac, n) {
		if time.Now().After(deadline) {
			return fmt.Errorf("dsl: %s node%d expected the node not to come up, but it answers JSON-RPC after %s",
				action, n.Index, nodeDownProbeTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(nodeDownProbeInterval):
		}
	}
	if reason, ok := ac.Args["reason"].(string); ok && reason != "" {
		if !strings.Contains(strings.ToLower(evidence), strings.ToLower(reason)) {
			return fmt.Errorf("dsl: %s node%d did not come up, but not for %q. evidence: %s",
				action, n.Index, reason, evidence)
		}
	}
	ac.Value = evidence
	return nil
}

// nodeFailureEvidence collects what can be said about why a node is not up: the
// launcher's error and, when the control can read it, the tail of the node's
// log. Reading the log is best effort — the reason match falls back to the
// launcher error when there is no log to read.
func nodeFailureEvidence(ctx context.Context, ctrl interp.NodeControl, n node.Node, launchErr error) string {
	var parts []string
	if launchErr != nil {
		parts = append(parts, launchErr.Error())
	}
	if reader, ok := ctrl.(interp.NodeLogReader); ok {
		if out, err := reader.Log(ctx, n, nodeLogDefaultMaxBytes); err == nil && out != "" {
			parts = append(parts, out)
		}
	}
	if len(parts) == 0 {
		return "(no launcher error and no readable log)"
	}
	return strings.Join(parts, "\n")
}

// nodeAnswers reports whether n serves JSON-RPC right now.
func nodeAnswers(ctx context.Context, ac *interp.ActionCtx, n node.Node) bool {
	if n.RPCURL == "" {
		return false
	}
	c, err := clientFor(ac.Deps, n.RPCURL)
	if err != nil {
		return false
	}
	_, err = c.BlockNumber(ctx)
	return err == nil
}

// readNodeLogAction binds the tail of one node's captured stdout/stderr under
// "save", so an assertion can match what the node said. Args: on (selector,
// required), maxBytes (optional, defaults to nodeLogDefaultMaxBytes).
//
// It needs a control that owns the node processes; attach mode says so rather
// than binding an empty string.
type readNodeLogAction struct{}

func (readNodeLogAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionReadNodeLog)
	if err != nil {
		return err
	}
	reader, ok := ctrl.(interp.NodeLogReader)
	if !ok {
		return fmt.Errorf("dsl: readNodeLog node%d: this run's node control cannot read logs", n.Index)
	}
	maxBytes := nodeLogDefaultMaxBytes
	if v, ok := ac.Args["maxBytes"]; ok {
		b, err := positiveIntArg(v, "maxBytes")
		if err != nil {
			return fmt.Errorf("dsl: readNodeLog: %w", err)
		}
		maxBytes = b
	}
	out, err := reader.Log(ctx, n, maxBytes)
	if err != nil {
		return fmt.Errorf("dsl: readNodeLog node%d: %w", n.Index, err)
	}
	ac.Value = out
	return nil
}

// positiveIntArg reads a DSL numeric argument that must be above zero. JSON
// numbers arrive as float64, and a spec may also write one as a string.
func positiveIntArg(v any, name string) (int, error) {
	var n int
	switch t := v.(type) {
	case float64:
		n = int(t)
	case int:
		n = t
	case string:
		parsed, err := strconv.Atoi(t)
		if err != nil {
			return 0, fmt.Errorf("%s %q is not a number: %w", name, t, err)
		}
		n = parsed
	default:
		return 0, fmt.Errorf("%s must be a number", name)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero (got %d)", name, n)
	}
	return n, nil
}

// stopNodeAction stops one node. Args: on (selector, required).
type stopNodeAction struct{}

func (stopNodeAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionStopNode)
	if err != nil {
		return err
	}
	stopped, err := ctrl.Stop(ctx, n)
	if err != nil {
		return fmt.Errorf("dsl: stopNode node%d: %w", n.Index, err)
	}
	ac.Env.UpdateNode(stopped)
	return nil
}

// startNodeAction starts one previously stopped node. Args: on (required).
type startNodeAction struct{}

func (startNodeAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionStartNode)
	if err != nil {
		return err
	}
	started, err := ctrl.Start(ctx, n)
	if expectsNodeDown(ac.Args) {
		return confirmNodeDown(ctx, ac, n, ctrl, err, actionStartNode)
	}
	if err != nil {
		return fmt.Errorf("dsl: startNode node%d: %w", n.Index, err)
	}
	ac.Env.UpdateNode(started)
	return nil
}

// restartNodeAction stops then starts one node — the sync-recovery scenario.
// Args: on (required).
type restartNodeAction struct{}

func (restartNodeAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionRestartNode)
	if err != nil {
		return err
	}
	stopped, err := ctrl.Stop(ctx, n)
	if err != nil {
		return fmt.Errorf("dsl: restartNode node%d: stop: %w", n.Index, err)
	}
	ac.Env.UpdateNode(stopped)
	started, err := ctrl.Start(ctx, stopped)
	if err != nil {
		return fmt.Errorf("dsl: restartNode node%d: start: %w", n.Index, err)
	}
	ac.Env.UpdateNode(started)
	return nil
}

// swapNodeAction swaps one node onto a different binary mid-test, so a network
// runs mixed binaries (E7). The datadir and genesis are unchanged; the pre-swap
// pid/command are kept as a ledger revision. Args: on (selector, required),
// binary (path, required). The node control must own the node processes; plain
// attach cannot swap.
type swapNodeAction struct{}

func (swapNodeAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionSwapNode)
	if err != nil {
		return err
	}
	binary, _ := ac.Args["binary"].(string)
	config := configOverrides(ac.Args["config"])
	overlay, err := genesisOverlayArg(ac.Args["genesisOverlay"])
	if err != nil {
		return fmt.Errorf("dsl: swapNode node%d: %w", n.Index, err)
	}
	if binary == "" && len(config) == 0 && len(overlay) == 0 {
		return fmt.Errorf("dsl: swapNode requires a \"binary\", \"config\", or \"genesisOverlay\"")
	}
	purpose, _ := ac.Args["purpose"].(string)
	sw, ok := ctrl.(interp.NodeSwapper)
	if !ok {
		return fmt.Errorf("dsl: swapNode node%d: this run's node control cannot swap", n.Index)
	}
	swapped, err := sw.Swap(ctx, n, interp.NodeChange{
		Binary: binary, Config: config, GenesisOverlay: overlay, Purpose: purpose})
	if expectsNodeDown(ac.Args) {
		return confirmNodeDown(ctx, ac, n, ctrl, err, actionSwapNode)
	}
	if err != nil {
		return fmt.Errorf("dsl: swapNode node%d: %w", n.Index, err)
	}
	ac.Env.UpdateNode(swapped)
	return nil
}

// genesisOverlayArg encodes a swapNode "genesisOverlay" argument — a JSON object
// the spec writes inline — back into the bytes the genesis merge takes. A
// missing argument is not an error: an ordinary swap has none.
func genesisOverlayArg(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("genesisOverlay must be a JSON object")
	}
	if len(obj) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("encode genesisOverlay: %w", err)
	}
	return b, nil
}

// configOverrides normalizes a swapNode "config" arg to key=value strings: a
// JSON object {"k":"v"} (sorted so the applied order is deterministic) or a list
// ["k=v"].
func configOverrides(v any) []string {
	switch c := v.(type) {
	case map[string]any:
		out := make([]string, 0, len(c))
		for k, val := range c {
			out = append(out, fmt.Sprintf("%s=%v", k, val))
		}
		sort.Strings(out)
		return out
	case []any:
		var out []string
		for _, e := range c {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// faultTarget resolves the action's "on" selector and the injected node control,
// naming whichever is missing.
func faultTarget(ac *interp.ActionCtx, action string) (node.Node, interp.NodeControl, error) {
	if ac.Deps == nil || ac.Deps.Nodes == nil {
		return node.Node{}, nil, fmt.Errorf(
			"dsl: %s needs node control, which this run has none of (attach mode does not own the node processes)", action)
	}
	if ac.Env == nil {
		return node.Node{}, nil, fmt.Errorf("dsl: %s: no environment", action)
	}
	sel, _ := ac.Args["on"].(string)
	if sel == "" {
		return node.Node{}, nil, fmt.Errorf("dsl: %s requires an \"on\" selector", action)
	}
	n, err := ac.Env.Resolve(sel)
	if err != nil {
		return node.Node{}, nil, fmt.Errorf("dsl: %s: %w", action, err)
	}
	return n, ac.Deps.Nodes, nil
}

// partitionAction splits the network by dropping every peer link that crosses a
// group boundary, so the groups can only see themselves. This is how a spec
// induces a fork to check the cross-node divergence assertions (F8 AC-2).
//
// Args: groups — two or more lists of node selectors, e.g.
//
//	{"partition": {"groups": [["bp1","bp2"], ["bp3","bp4"]]}}
//
// Links are severed from BOTH sides, because admin_removePeer only drops the
// connection the node it is called on holds. Nodes not named in any group are
// left alone.
//
// Note: this severs current connections. A network with peer discovery enabled
// may re-establish them; chainbench networks peer through static-nodes, so the
// split holds until healPartition (or a node restart) restores it.
type partitionAction struct{}

func (partitionAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	groups, err := partitionGroups(ac)
	if err != nil {
		return err
	}
	enodes, err := enodesFor(ctx, ac, flatten(groups))
	if err != nil {
		return err
	}
	for i, a := range groups {
		for j, b := range groups {
			if i == j {
				continue
			}
			for _, from := range a {
				for _, to := range b {
					c, err := clientFor(ac.Deps, from.RPCURL)
					if err != nil {
						return err
					}
					if err := c.RemovePeer(ctx, enodes[to.Index]); err != nil {
						return fmt.Errorf("dsl: partition: node%d drop node%d: %w", from.Index, to.Index, err)
					}
				}
			}
		}
	}
	return nil
}

// healPartitionAction restores full connectivity by re-adding every pair. With
// no "groups" it heals across the whole environment, which is what a post-action
// wants after a fault test.
type healPartitionAction struct{}

func (healPartitionAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	if ac.Env == nil {
		return fmt.Errorf("dsl: healPartition: no environment")
	}
	nodes := ac.Env.Nodes()
	if raw, ok := ac.Args["groups"]; ok {
		groups, err := resolveGroups(ac, raw)
		if err != nil {
			return err
		}
		nodes = flatten(groups)
	}
	if len(nodes) < 2 {
		return fmt.Errorf("dsl: healPartition needs at least 2 nodes, got %d", len(nodes))
	}
	enodes, err := enodesFor(ctx, ac, nodes)
	if err != nil {
		return err
	}
	for _, from := range nodes {
		c, err := clientFor(ac.Deps, from.RPCURL)
		if err != nil {
			return err
		}
		for _, to := range nodes {
			if from.Index == to.Index {
				continue
			}
			if err := c.AddPeer(ctx, enodes[to.Index]); err != nil {
				return fmt.Errorf("dsl: healPartition: node%d add node%d: %w", from.Index, to.Index, err)
			}
		}
	}
	return nil
}

// partitionGroups resolves and validates the action's "groups" argument.
func partitionGroups(ac *interp.ActionCtx) ([][]node.Node, error) {
	raw, ok := ac.Args["groups"]
	if !ok {
		return nil, fmt.Errorf("dsl: partition requires \"groups\" (two or more lists of node selectors)")
	}
	groups, err := resolveGroups(ac, raw)
	if err != nil {
		return nil, err
	}
	if len(groups) < 2 {
		return nil, fmt.Errorf("dsl: partition needs at least 2 groups, got %d", len(groups))
	}
	for i, g := range groups {
		if len(g) == 0 {
			return nil, fmt.Errorf("dsl: partition: group %d is empty", i)
		}
	}
	return groups, nil
}

// resolveGroups turns the DSL's [[selector...]...] into resolved node groups.
func resolveGroups(ac *interp.ActionCtx, raw any) ([][]node.Node, error) {
	if ac.Env == nil {
		return nil, fmt.Errorf("dsl: partition: no environment")
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("dsl: partition: \"groups\" must be a list of node-selector lists")
	}
	out := make([][]node.Node, 0, len(list))
	for gi, g := range list {
		sels, ok := g.([]any)
		if !ok {
			return nil, fmt.Errorf("dsl: partition: group %d must be a list of node selectors", gi)
		}
		nodes := make([]node.Node, 0, len(sels))
		for _, s := range sels {
			sel, ok := s.(string)
			if !ok {
				return nil, fmt.Errorf("dsl: partition: group %d has a non-string selector", gi)
			}
			n, err := ac.Env.Resolve(sel)
			if err != nil {
				return nil, fmt.Errorf("dsl: partition: %w", err)
			}
			nodes = append(nodes, n)
		}
		out = append(out, nodes)
	}
	return out, nil
}

// enodesFor asks each node for its own enode (admin_nodeInfo), keyed by index.
// Peers are named by enode, and only the node itself knows its own.
func enodesFor(ctx context.Context, ac *interp.ActionCtx, nodes []node.Node) (map[int]string, error) {
	out := make(map[int]string, len(nodes))
	for _, n := range nodes {
		if _, done := out[n.Index]; done {
			continue
		}
		c, err := clientFor(ac.Deps, n.RPCURL)
		if err != nil {
			return nil, err
		}
		enode, err := c.Enode(ctx)
		if err != nil {
			return nil, fmt.Errorf("dsl: enode of node%d: %w", n.Index, err)
		}
		out[n.Index] = enode
	}
	return out, nil
}

// flatten concatenates node groups, preserving order.
func flatten(groups [][]node.Node) []node.Node {
	var out []node.Node
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// compile-time assertion that the RPC client satisfies what the actions need.
var _ = func(c *rpc.Client) { _ = c.AddPeer }
