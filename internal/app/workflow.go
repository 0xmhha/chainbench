package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	chainsetupmod "github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// The workflow is what MCP exists to reach (architecture-v2 §2): one call
// that takes DSL inputs, composes the chain the specs declare, runs the specs
// through the test engine, collects the session, and reports — the sequence
// an operator would otherwise drive verb by verb.

// RunSuiteIn is one whole workflow request. The network is declared by the
// specs' env; the fields here are the operator's overrides and the places
// only the operator knows.
type RunSuiteIn struct {
	// SpecPaths are the DSL files to run, each read and env-resolved the one
	// way every surface does (dsl.ReadFiles).
	SpecPaths []string
	// DataDir is the composition workspace; the network is set up here.
	DataDir string
	// Chain, when set, must agree with what the specs declare.
	Chain string
	// Binary overrides the declared binary path for a single-binary network.
	Binary string
	// Validators overrides the declared validator count.
	Validators int
	// Server selects where the nodes run, from the operator's server set.
	Server ServerRef
	// Docker treats the servers as local docker containers (the option is the
	// power switch, as everywhere).
	Docker bool
	// KeysDir overrides the declared key set (default keys/preset).
	KeysDir string
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

// RunSuiteOut is the workflow's report.
type RunSuiteOut struct {
	// SetupSteps are the composition steps' recorded details, in order.
	SetupSteps []string
	// Endpoints are the RPC URLs the tests ran against.
	Endpoints []string
	// SessionRoot holds the run's artifacts.
	SessionRoot string
	// Summary is the collected result.
	Summary testengine.Summary
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
}

// RunSuite runs the whole flow: read the DSL, compose the chain it declares,
// run the tests, collect, and stop the network unless asked to keep it. Setup
// failure aborts before any test runs; a test-phase failure still tears down.
func RunSuite(ctx context.Context, d Deps, in RunSuiteIn) (RunSuiteOut, error) {
	if len(in.SpecPaths) == 0 {
		return RunSuiteOut{}, fmt.Errorf("app: run suite: no specs given")
	}
	if in.DataDir == "" {
		return RunSuiteOut{}, fmt.Errorf("app: run suite: a workspace directory is required")
	}
	specs, err := dsl.ReadFiles(in.SpecPaths)
	if err != nil {
		return RunSuiteOut{}, err
	}
	parsed := make([]dsl.Spec, 0, len(specs))
	for i, raw := range specs {
		s, err := dsl.Parse(raw)
		if err != nil {
			return RunSuiteOut{}, fmt.Errorf("app: run suite: %s: %w", in.SpecPaths[i], err)
		}
		parsed = append(parsed, s)
	}
	if err := sameChain(parsed); err != nil {
		return RunSuiteOut{}, fmt.Errorf("app: run suite: %w", err)
	}
	comp, err := compositionOf(ctx, parsed[0], in)
	if err != nil {
		return RunSuiteOut{}, fmt.Errorf("app: run suite: %w", err)
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
			return out, fmt.Errorf("app: run suite: setup: %w", err)
		}
		out.Preflight = preflight.Compose.String()
		net = composed{endpoints: handoffEndpoints(ns), caps: chainCaps(chain), teardown: teardown}
	} else {
		net, err = workspaceUp(ctx, d, *comp.up, &out)
		if err != nil {
			return out, err
		}
	}
	out.Endpoints = net.endpoints

	runErr := func() error {
		if in.WaitBlocks > 0 {
			if err := waitForHead(ctx, net.endpoints[0], in.WaitBlocks); err != nil {
				return fmt.Errorf("app: run suite: %w", err)
			}
		}
		eng, err := testengine.NewAttachEngine(testengine.AttachConfig{
			Chain: chain, RPCURLs: net.endpoints,
			ArtifactRoot: in.ArtifactRoot, Caps: append(append([]string(nil), net.caps...), in.Caps...), Clock: d.Clock,
		})
		if err != nil {
			return fmt.Errorf("app: run suite: engine: %w", err)
		}
		root, err := eng.Run(ctx, specs)
		if root != "" {
			out.SessionRoot = root
			if sum, serr := testengine.ReadSessionSummary(root); serr == nil {
				out.Summary = sum
			}
		}
		if err != nil {
			return fmt.Errorf("app: run suite: %w", err)
		}
		return nil
	}()

	if !in.KeepUp {
		if err := net.teardown(ctx); err == nil {
			out.Stopped = true
		} else if runErr == nil {
			runErr = fmt.Errorf("app: run suite: teardown: %w", err)
		}
	}
	return out, runErr
}

// workspaceUp composes a single-binary network through the workspace steps,
// reusing what is already composed when preflight says it can. It records the
// steps and the preflight decision on out.
func workspaceUp(ctx context.Context, d Deps, up NetUpIn, out *RunSuiteOut) (composed, error) {
	// What is composed here already may be what this suite wants: ask before
	// rebuilding. The decision is recorded beside the setup steps so a run
	// that reused a network says so, and one that rebuilt says why.
	decision := preflightDecision(ctx, d, up.DataDir, chainsetupmod.WantOf(up))
	out.Preflight = decision.String()
	switch decision.Verdict {
	case preflight.Reuse:
		out.SetupSteps = []string{"preflight: reuse — " + decision.String()}
	case preflight.RebuildNodes:
		for _, idx := range decision.Nodes {
			st, err := NetRestart(ctx, d, NetRestartIn{DataDir: up.DataDir, Node: idx})
			if err != nil {
				return composed{}, fmt.Errorf("app: run suite: preflight restart node%d: %w", idx, err)
			}
			out.SetupSteps = append(out.SetupSteps, "restart: "+st.Detail)
		}
	default:
		res, err := NetUp(ctx, d, up)
		out.SetupSteps = res.Steps
		if err != nil {
			return composed{}, fmt.Errorf("app: run suite: setup: %w", err)
		}
	}

	endpoints, err := chainsetupmod.NetEndpoints(ctx, d.chainsetupDeps(), chainsetupmod.NetEndpointsIn{DataDir: up.DataDir})
	if err != nil {
		return composed{}, fmt.Errorf("app: run suite: endpoints: %w", err)
	}
	var caps []string
	if ws, err := chainsetupmod.Open(up.DataDir, d.Clock); err == nil {
		caps = ws.State().Capabilities
	}
	return composed{
		endpoints: endpoints,
		caps:      caps,
		teardown: func(ctx context.Context) error {
			_, err := NetStop(ctx, d, NetStopIn{DataDir: up.DataDir})
			return err
		},
	}, nil
}

// chainCaps is what a chain advertises by its manifest: what a handoff
// network, which has no workspace record, tells capability-gated specs.
func chainCaps(chain string) []string {
	p, err := ResolveChain(chain, "", "")
	if err != nil {
		return nil
	}
	return p.Manifest().Capabilities
}

// RunSummary is a collected session result.
type RunSummary = testengine.Summary

// AttachRunIn runs specs against an already-running network.
type AttachRunIn struct {
	Chain        string
	RPCURLs      []string
	ArtifactRoot string
	Caps         []string
	// Specs are raw DSL JSON blobs (already env-resolved).
	Specs [][]byte
}

// AttachRun attaches the test engine to a running network and runs the specs,
// returning the session root.
func AttachRun(ctx context.Context, d Deps, in AttachRunIn) (string, error) {
	eng, err := testengine.NewAttachEngine(testengine.AttachConfig{
		Chain: in.Chain, RPCURLs: in.RPCURLs,
		ArtifactRoot: in.ArtifactRoot, Caps: in.Caps, Clock: d.Clock,
	})
	if err != nil {
		return "", fmt.Errorf("app: attach run: %w", err)
	}
	return eng.Run(ctx, in.Specs)
}

// SessionSummary reads a session's collected summary.
func SessionSummary(root string) (RunSummary, error) {
	return testengine.ReadSessionSummary(root)
}

// preflightDecision asks the workspace, when there is one, how much of what it
// holds the request can reuse. No workspace, or one that cannot be read, is
// simply "compose".
func preflightDecision(ctx context.Context, d Deps, dir string, want preflight.Want) preflight.Decision {
	ws, err := chainsetupmod.Open(dir, d.Clock)
	if err != nil || len(ws.State().Nodes) == 0 {
		return preflight.Decision{Verdict: preflight.Compose, Reasons: []string{"nothing is composed on the target"}}
	}
	return ws.Compare(ctx, want)
}
