package testengine

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The suite is the engine's outermost flow (R4, consolidation plan §0-2): one
// call takes DSL files, composes the chain they declare through chainsetup —
// the one composition owner — and runs the specs against it. The four parts
// the architecture names are ① compose (chainsetup, here), then per spec
// ② pre-test hooks, ③ the test, ④ post-test hooks — the interpreter runs
// ②–④ from the spec's own hooks and steps.

// RunSuiteIn is one whole suite request. The network is declared by the
// specs' env; the fields here are the operator's overrides and the places
// only the operator knows.
type RunSuiteIn struct {
	// SpecPaths are the DSL files to run, each read and env-resolved the one
	// way every surface does (dsl.ReadFiles).
	SpecPaths []string
	// SpecContent is inline, self-contained spec bytes to run instead of reading
	// files — the form the MCP surface passes so an agent composes from inline
	// specs without writing them to disk. When set, SpecPaths is ignored.
	SpecContent [][]byte
	// DataDir is the composition workspace; the network is set up here.
	DataDir string
	// Chain, when set, must agree with what the specs declare.
	Chain string
	// Binary overrides the declared binary path for a single-binary network.
	Binary string
	// Validators overrides the declared validator count.
	Validators int
	// Server selects where the nodes run, from the operator's server set.
	Server resource.ServerRef
	// Docker treats the servers as local docker containers (the option is the
	// power switch, as everywhere).
	Docker bool
	// KeysDir overrides the declared key set (default keys/preset).
	KeysDir string
	// KeysSource overrides where node identities come from ("preset" or
	// "generate"); empty follows the declaration.
	KeysSource string
	// ChainID overrides the manifest chain id in the built genesis.
	ChainID int64
	// NetworkID pins the devp2p network id on every node's command line.
	NetworkID int64
	// LaunchOpts are high-precedence launch knobs (key=value, bare key for a
	// boolean) applied on top of the declaration's launch block.
	LaunchOpts []string
	// ArtifactRoot is where the test session writes; empty uses the
	// workspace's sessions directory.
	ArtifactRoot string
	// Caps are extra capabilities the operator asserts the network provides,
	// beyond what the composition advertises.
	Caps []string
	// KeepUp leaves the network running after the tests (default: stop it).
	KeepUp bool
	// WaitBlocks, when positive, waits until the chain's head reaches this
	// height before running any test — a spec asserting on sealed blocks is
	// meaningless against a chain that has not sealed one. Bounded by
	// waitBlocksTimeout.
	WaitBlocks uint64
}

// waitBlocksTimeout bounds the wait for the chain to reach WaitBlocks.
const waitBlocksTimeout = 60 * time.Second

// waitForHead polls the first endpoint until the head reaches target.
func waitForHead(ctx context.Context, url string, target uint64) error {
	ctx, cancel := context.WithTimeout(ctx, waitBlocksTimeout)
	defer cancel()
	c := rpc.Dial(url)
	for {
		if n, err := c.BlockNumber(ctx); err == nil && n >= target {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("chain did not reach block %d within %s", target, waitBlocksTimeout)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// RunSuiteOut is the suite's report.
type RunSuiteOut struct {
	// SetupSteps are the composition steps' recorded details, in order.
	SetupSteps []string
	// Endpoints are the RPC URLs the tests ran against.
	Endpoints []string
	// SessionRoot holds the run's artifacts.
	SessionRoot string
	// Summary is the collected result.
	Summary Summary
	// Stopped reports whether the network was torn down afterwards.
	Stopped bool
	// Preflight is the reuse decision the run started from: reuse, which
	// nodes were rebuilt, or why everything was. A handoff is always composed.
	Preflight string
}

// composed is a network the suite brought up: where the tests reach it, what
// it advertises, and how to take it down.
type composed struct {
	endpoints []string
	caps      []string
	teardown  func(context.Context) error
	// nodes is the full node table when the workspace knows it; nil for a
	// handoff, whose engine builds the table from the endpoints.
	nodes *node.NodeSet
	// control acts on the node processes for fault steps; nil when the run
	// does not own them.
	control interp.NodeControl
	// keysDir is the key set the network was composed from, so a spec can name
	// an account by label instead of by address.
	keysDir string
}

// workspaceNodes adapts the workspace's node verbs to the interpreter's
// NodeControl, so fault steps (stopNode/startNode/restartNode) act on a
// suite-composed network through the same record every other verb uses.
type workspaceNodes struct {
	sd      chainsetup.Deps
	dataDir string
}

// Stop stops one node through the workspace and returns it with its pid
// cleared, which the interpreter writes back to the environment's node table.
func (w workspaceNodes) Stop(ctx context.Context, n node.Node) (node.Node, error) {
	if err := chainsetup.NodeStop(ctx, w.sd, chainsetup.NodeStopIn{DataDir: w.dataDir, Index: n.Index}); err != nil {
		return n, err
	}
	n.PID = 0
	return n, nil
}

// Start relaunches one previously stopped node through the workspace and
// returns it with its new pid.
func (w workspaceNodes) Start(ctx context.Context, n node.Node) (node.Node, error) {
	out, err := chainsetup.NodeStart(ctx, w.sd, chainsetup.NodeStartIn{DataDir: w.dataDir, Index: n.Index})
	if err != nil {
		return n, err
	}
	return out.Node, nil
}

// Swap relaunches one node with a different binary and/or config through the
// workspace, satisfying interp.NodeSwapper so the swapNode action reaches it.
func (w workspaceNodes) Swap(ctx context.Context, n node.Node, change interp.NodeChange) (node.Node, error) {
	out, err := chainsetup.NodeSwap(ctx, w.sd, chainsetup.NodeSwapIn{
		DataDir: w.dataDir, Index: n.Index,
		Binary: change.Binary, Config: change.Config, Purpose: change.Purpose,
	})
	if err != nil {
		return n, err
	}
	return out.Node, nil
}

// RunSuite runs the whole flow: read the DSL, compose the chain it declares
// through chainsetup, run the tests, collect, and stop the network unless
// asked to keep it. Setup failure aborts before any test runs; a test-phase
// failure still tears down.
func RunSuite(ctx context.Context, sd chainsetup.Deps, in RunSuiteIn) (RunSuiteOut, error) {
	if len(in.SpecPaths) == 0 && len(in.SpecContent) == 0 {
		return RunSuiteOut{}, fmt.Errorf("engine: run suite: no specs given")
	}
	if in.DataDir == "" {
		return RunSuiteOut{}, fmt.Errorf("engine: run suite: a workspace directory is required")
	}
	specs := in.SpecContent
	if len(specs) == 0 {
		var err error
		if specs, err = dsl.ReadFiles(in.SpecPaths); err != nil {
			return RunSuiteOut{}, err
		}
	}
	parsed := make([]dsl.Spec, 0, len(specs))
	for i, raw := range specs {
		s, err := dsl.Parse(raw)
		if err != nil {
			return RunSuiteOut{}, fmt.Errorf("engine: run suite: spec %d: %w", i+1, err)
		}
		parsed = append(parsed, s)
	}
	if err := sameChain(parsed); err != nil {
		return RunSuiteOut{}, fmt.Errorf("engine: run suite: %w", err)
	}
	// Pre-flight before anything is allocated or written: a spec that names an
	// action/assertion/reader/reference that does not resolve, or a malformed
	// node selector, fails here rather than after a network is composed.
	if err := Precheck(parsed); err != nil {
		return RunSuiteOut{}, fmt.Errorf("engine: run suite: %w", err)
	}
	comp, err := compositionOf(ctx, parsed[0], in)
	if err != nil {
		return RunSuiteOut{}, fmt.Errorf("engine: run suite: %w", err)
	}
	if in.ArtifactRoot == "" {
		// The session belongs with the workspace it tested.
		in.ArtifactRoot = filepath.Join(in.DataDir, "sessions")
	}
	chain := parsed[0].Chain.Name

	out := RunSuiteOut{}
	var net composed
	if comp.handoff != nil {
		ns, steps, teardown, err := handoffUp(ctx, *comp.handoff)
		out.SetupSteps = steps
		if err != nil {
			return out, fmt.Errorf("engine: run suite: setup: %w", err)
		}
		out.Preflight = preflight.Compose.String()
		net = composed{endpoints: handoffEndpoints(ns), caps: chainCaps(chain), teardown: teardown}
	} else {
		net, err = composeWorkspace(ctx, sd, *comp.up, &out)
		if err != nil {
			return out, err
		}
	}
	out.Endpoints = net.endpoints

	runErr := func() error {
		if in.WaitBlocks > 0 {
			if err := waitForHead(ctx, net.endpoints[0], in.WaitBlocks); err != nil {
				return fmt.Errorf("engine: run suite: %w", err)
			}
		}
		eng, err := NewAttachEngine(AttachConfig{
			Chain: chain, RPCURLs: net.endpoints,
			ArtifactRoot: in.ArtifactRoot, Caps: append(append([]string(nil), net.caps...), in.Caps...), Clock: sd.Clock,
			NodeSet: net.nodes, Control: net.control, KeysDir: net.keysDir,
			// A bus turns on chainstate sampling: chainstate.jsonl is written per
			// environment (E8) even on the headless suite path, which has no
			// dashboard subscriber (events are simply dropped).
			Bus: collector.NewBus(),
			// A remote target reads its node logs over SSH (and reconnects a
			// dropped session, E8); a local target leaves this nil and the
			// collector reads the local filesystem.
			LogReader: remoteLogReader(sd, in.DataDir),
			// Gate the network before each test: a node a prior fault test left
			// down is restarted or waited on within limits before the next test
			// runs (E6). A handoff or bare-URL attach (no workspace) passes no
			// nodes, so the gate is a no-op there.
			PreSpec: func(ctx context.Context, _ session.Environment) error {
				return gateReady(ctx, sd, in.DataDir, net.nodes, &out.SetupSteps)
			},
			// Gather a failed test's evidence (node logs, process, RPC/block) into
			// its observations/ before the run moves on (E8).
			OnFail: func(ctx context.Context, _ session.Environment, rec session.TestRecord) error {
				collectFailureData(ctx, sd, in.DataDir, net.nodes, rec)
				return nil
			},
		})
		if err != nil {
			return fmt.Errorf("engine: run suite: engine: %w", err)
		}
		root, err := eng.Run(ctx, specs)
		if root != "" {
			out.SessionRoot = root
			if sum, serr := ReadSessionSummary(root); serr == nil {
				out.Summary = sum
			}
		}
		if err != nil {
			return fmt.Errorf("engine: run suite: %w", err)
		}
		return nil
	}()

	if !in.KeepUp {
		if err := net.teardown(ctx); err == nil {
			out.Stopped = true
		} else if runErr == nil {
			runErr = fmt.Errorf("engine: run suite: teardown: %w", err)
		}
	}
	return out, runErr
}

// composeWorkspace composes a single-binary network through the workspace
// steps, reusing what is already composed when preflight says it can. It
// records the steps and the preflight decision on out.
func composeWorkspace(ctx context.Context, sd chainsetup.Deps, up chainsetup.NetUpIn, out *RunSuiteOut) (composed, error) {
	// What is composed here already may be what this suite wants: ask before
	// rebuilding. The decision is recorded beside the setup steps so a run
	// that reused a network says so, and one that rebuilt says why.
	decision := preflightDecision(ctx, sd, up.DataDir, chainsetup.WantOf(up))
	out.Preflight = decision.String()
	switch decision.Verdict {
	case preflight.Reuse:
		out.SetupSteps = []string{"preflight: reuse — " + decision.String()}
	case preflight.RebuildNodes:
		for _, idx := range decision.Nodes {
			st, err := chainsetup.NetRestart(ctx, sd, chainsetup.NetRestartIn{DataDir: up.DataDir, Node: idx})
			if err != nil {
				return composed{}, fmt.Errorf("engine: run suite: preflight restart node%d: %w", idx, err)
			}
			out.SetupSteps = append(out.SetupSteps, "restart: "+st.Detail)
		}
	default:
		res, err := chainsetup.NetUp(ctx, sd, up)
		out.SetupSteps = res.Steps
		if err != nil {
			return composed{}, fmt.Errorf("engine: run suite: setup: %w", err)
		}
	}

	endpoints, err := chainsetup.NetEndpoints(ctx, sd, chainsetup.NetEndpointsIn{DataDir: up.DataDir})
	if err != nil {
		return composed{}, fmt.Errorf("engine: run suite: endpoints: %w", err)
	}
	var caps []string
	if ws, err := chainsetup.Open(up.DataDir, sd.Clock); err == nil {
		caps = ws.State().Capabilities
	}
	// The workspace knows the whole node table — indices, hosts, every
	// endpoint — so the engine attaches to that, not to bare URLs, and fault
	// steps get a control over the recorded processes. The recorded RPC host is
	// the node's own (a container's internal address under --docker, an SSH
	// host under a remote target); the dial address a test must actually use is
	// the localmap-translated endpoint, so overwrite each node's RPCURL with the
	// matching entry from NetEndpoints (same node order). Without this a
	// docker/remote run dials the untranslated host and times out.
	var nodes *node.NodeSet
	if st, err := chainsetup.NetworkStatus(ctx, sd, chainsetup.NetworkStatusIn{DataDir: up.DataDir}); err == nil && len(st.Nodes.Nodes) > 0 {
		ns := st.Nodes
		if len(ns.Nodes) == len(endpoints) {
			for i := range ns.Nodes {
				ns.Nodes[i].RPCURL = endpoints[i]
			}
		}
		nodes = &ns
	}
	// The network is composed (or reused); gate it before any test runs on it —
	// wait on nodes still coming up, restart dead ones within limits, terminate
	// on a state that would need a destructive remedy (E6).
	if err := gateReady(ctx, sd, up.DataDir, nodes, &out.SetupSteps); err != nil {
		return composed{}, fmt.Errorf("engine: run suite: %w", err)
	}
	return composed{
		endpoints: endpoints,
		caps:      caps,
		teardown: func(ctx context.Context) error {
			_, err := chainsetup.NetStop(ctx, sd, chainsetup.NetStopIn{DataDir: up.DataDir})
			return err
		},
		nodes:   nodes,
		control: workspaceNodes{sd: sd, dataDir: up.DataDir},
		keysDir: up.KeysDir,
	}, nil
}

// remoteLogReader returns an SSH-backed log reader for a remote target's node
// logs, which the collector wraps so a dropped session reconnects (E8); a local
// target (or a lookup error — collection is best-effort) returns nil, and the
// collector reads the local filesystem.
func remoteLogReader(sd chainsetup.Deps, dataDir string) collector.LogReader {
	runner, err := chainsetup.NetRunner(sd, dataDir)
	if err != nil || runner == nil {
		return nil
	}
	return process.NewRemoteLogReader(runner)
}

// chainCaps is what a chain advertises by its manifest: what a handoff
// network, which has no workspace record, tells capability-gated specs.
func chainCaps(chain string) []string {
	p, err := external.ResolveChain(chain, "", "")
	if err != nil {
		return nil
	}
	return p.Manifest().Capabilities
}

// preflightDecision asks the workspace, when there is one, how much of what it
// holds the request can reuse. No workspace, or one that cannot be read, is
// simply "compose".
func preflightDecision(ctx context.Context, sd chainsetup.Deps, dir string, want preflight.Want) preflight.Decision {
	ws, err := chainsetup.Open(dir, sd.Clock)
	if err != nil || len(ws.State().Nodes) == 0 {
		return preflight.Decision{Verdict: preflight.Compose, Reasons: []string{"nothing is composed on the target"}}
	}
	return ws.Compare(ctx, want)
}
