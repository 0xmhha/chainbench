package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	chainsetupmod "github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testengine"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// The workflow is what MCP exists to reach (architecture-v2 §2): one call
// that takes DSL inputs, sets the chain up through chainsetup, runs the specs
// through the test engine, collects the session, and reports — the sequence
// an operator would otherwise drive verb by verb.

// RunSuiteIn is one whole workflow request.
type RunSuiteIn struct {
	// SpecPaths are the DSL files to run, each read and env-resolved the one
	// way every surface does (testspec.ReadFiles).
	SpecPaths []string
	// DataDir is the composition workspace; the network is set up here.
	DataDir string
	// Chain, Binary, Validators, and Server shape the setup, exactly as the
	// net verbs take them.
	Chain      string
	Binary     string
	Validators int
	Server     ServerRef
	// Docker treats the servers as local docker containers (the option is the
	// power switch, as everywhere).
	Docker bool
	// KeysDir is the key set the network composes from (default keys/preset).
	KeysDir string
	// ArtifactRoot is where the test session writes; empty uses the engine
	// default.
	ArtifactRoot string
	// Caps are extra capabilities the operator asserts the network provides.
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
}

// RunSuite runs the whole flow: parse the DSL, set up the chain, run the
// tests, collect, and stop the network unless asked to keep it. Setup failure
// aborts before any test runs; a test-phase failure still tears down.
func RunSuite(ctx context.Context, d Deps, in RunSuiteIn) (RunSuiteOut, error) {
	if len(in.SpecPaths) == 0 {
		return RunSuiteOut{}, fmt.Errorf("app: run suite: no specs given")
	}
	specs, err := testspec.ReadFiles(in.SpecPaths)
	if err != nil {
		return RunSuiteOut{}, err
	}
	if in.ArtifactRoot == "" {
		// The session belongs with the workspace it tested.
		in.ArtifactRoot = filepath.Join(in.DataDir, "sessions")
	}

	up, err := NetUp(ctx, d, NetUpIn{
		DataDir: in.DataDir, Chain: in.Chain, Binary: in.Binary,
		Validators: in.Validators, Server: in.Server, Docker: in.Docker,
		KeysDir: in.KeysDir, Stage: UpStart,
	})
	if err != nil {
		return RunSuiteOut{}, fmt.Errorf("app: run suite: setup: %w", err)
	}
	out := RunSuiteOut{SetupSteps: up.Steps}

	endpoints, err := chainsetupmod.NetEndpoints(ctx, d.chainsetupDeps(), chainsetupmod.NetEndpointsIn{DataDir: in.DataDir})
	if err == nil {
		out.Endpoints = endpoints
	}
	runErr := func() error {
		if err != nil {
			return fmt.Errorf("app: run suite: endpoints: %w", err)
		}
		if in.WaitBlocks > 0 {
			if err := waitForHead(ctx, endpoints[0], in.WaitBlocks); err != nil {
				return fmt.Errorf("app: run suite: %w", err)
			}
		}
		eng, err := testengine.NewAttachEngine(testengine.AttachConfig{
			Chain: in.Chain, RPCURLs: endpoints,
			ArtifactRoot: in.ArtifactRoot, Caps: in.Caps, Clock: d.Clock,
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
		if _, err := NetStop(ctx, d, NetStopIn{DataDir: in.DataDir}); err == nil {
			out.Stopped = true
		} else if runErr == nil {
			runErr = fmt.Errorf("app: run suite: teardown: %w", err)
		}
	}
	return out, runErr
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
