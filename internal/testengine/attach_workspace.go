package testengine

import (
	"github.com/0xmhha/chainbench/internal/core/lifecycle"

	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"os"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/home"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Attaching to a network a workspace already composed.
//
// The record is what makes this possible: it holds the node table, the
// endpoints and the capabilities the network advertised, so a later run can
// test that network without re-composing it — and without guessing what it is.

func artifactRoot(explicit, configPath, _ string) (string, error) {
	// The command names no layer and wins over every document.
	if explicit != "" {
		return explicit, nil
	}
	if configPath != "" {
		wc, err := resource.LoadWorkspaceConfig(configPath)
		if err != nil {
			return "", fmt.Errorf("engine: run suite: %w", err)
		}
		root, err := wc.ArtifactRoot()
		if err != nil {
			return "", fmt.Errorf("engine: run suite: %w", err)
		}
		if root != "" {
			if err := os.MkdirAll(root, 0o755); err != nil {
				return "", fmt.Errorf("engine: run suite: workspace-config artifactRoot %q: %w", root, err)
			}
			return root, nil
		}
	}
	return home.Sessions()
}

// attachWiring is the run-side wiring the compose path and the workspace-attach
// path share: which workspace, where the session goes, extra capabilities, the
// readiness-gate budget, and where to record the gate's steps.
type attachWiring struct {
	Chain              string
	DataDir            string
	ArtifactRoot       string
	Caps               []string
	NodeMonitorTimeout time.Duration
	SetupSteps         *[]string
	// Session, when non-nil, is the session the caller already opened. The
	// compose path opens one before it composes so a setup failure has a test
	// folder to be recorded in; the attach path has nothing to record before
	// the engine runs and leaves this nil.
	Session session.Session
}

// wiredAttachEngine builds an attach engine over a composed network with the
// full wiring: the composition manifest (WA11), chainstate sampling, a
// remote-aware log reader, a readiness gate before each test (E6), and
// failure-evidence collection (E8). Both RunSuite (after composing) and
// AttachWorkspaceRun (attaching to an existing workspace) build it the same
// way, so the attach path no longer loses the gate and the evidence (WA10).
func wiredAttachEngine(sd chainsetup.Deps, net composed, w attachWiring) (Engine, error) {
	return NewAttachEngine(AttachConfig{
		Chain: w.Chain, RPCURLs: net.endpoints,
		ArtifactRoot: w.ArtifactRoot, Caps: append(append([]string(nil), net.caps...), w.Caps...), Clock: sd.Clock,
		Session: w.Session,
		NodeSet: net.nodes, Control: net.control, KeysDir: net.keysDir,
		Artifacts: composedArtifacts(net),
		Bus:       collector.NewBus(),
		LogReader: remoteLogReader(sd, w.DataDir),
		PreSpec: func(ctx context.Context, _ session.Environment) error {
			return gateReady(ctx, sd, w.DataDir, net.nodes, w.SetupSteps, w.NodeMonitorTimeout, net.fork)
		},
		OnFail: func(ctx context.Context, _ session.Environment, rec session.TestRecord) error {
			collectFailureData(ctx, sd, w.DataDir, net.nodes, rec)
			return nil
		},
	})
}

// AttachWorkspaceIn attaches to the network a workspace already composed.
type AttachWorkspaceIn struct {
	// DataDir is the workspace whose network is up.
	DataDir string
	// Chain is the chain family; empty reads it from the workspace.
	Chain string
	// ArtifactRoot is where the session is written; empty defaults to the
	// workspace's sessions directory.
	ArtifactRoot string
	// Caps are extra capabilities the operator asserts, beyond what the
	// workspace advertised.
	Caps []string
	// Specs are the DSL blobs to run (already env-resolved).
	Specs [][]byte
	// NodeMonitorTimeout budgets the readiness gate; zero takes the default.
	NodeMonitorTimeout time.Duration
}

// AttachWorkspaceRun attaches to the network the workspace at DataDir composed
// and runs the specs against it, with the same readiness gate (E6), failure-
// evidence collection (E8), fault control, remote log reading, and composition
// manifest the compose path wires. It composes nothing — the network is already
// up — but reads the workspace's node table, capabilities, and key set so a
// spec addresses nodes by role and resolves account labels (WA10).
func AttachWorkspaceRun(ctx context.Context, sd chainsetup.Deps, in AttachWorkspaceIn) (string, error) {
	r := newRunner(sd, RunSuiteIn{
		DataDir: in.DataDir, Chain: in.Chain, SpecContent: in.Specs,
		ArtifactRoot: in.ArtifactRoot, Caps: in.Caps,
		NodeMonitorTimeout: in.NodeMonitorTimeout,
		// The network is somebody else's. Attaching to one and then taking it
		// down is not attaching.
		KeepUp: true,
	})
	r.attach = true
	out, err := r.Run(ctx)
	return out.SessionRoot, err
}

// composedArtifacts is the manifest of composition inputs a workspace-owned
// network was brought up against, recorded into each test's artifacts.json so a
// verdict is traceable to what it ran on (WA11). It names the genesis by its
// env-relative path — the one input every node shares and the anchor of the
// run's provenance. A network the suite did not compose (a handoff, or a
// bare-URL attach with no node table) owns no single genesis and gets none;
// the config, command, and deployment refs are the next fill of this seam.
func composedArtifacts(net composed) []session.ArtifactRef {
	if net.nodes == nil {
		return nil
	}
	return []session.ArtifactRef{{Kind: "genesis", Ref: "genesis.json"}}
}

// composeWorkspace composes a single-binary network through the workspace
// steps, reusing what is already composed when preflight says it can. It
// records the steps and the preflight decision on out.
func composeWorkspace(ctx context.Context, sd chainsetup.Deps, up chainsetup.ChainUpIn, out *RunSuiteOut, gateBudget time.Duration) (composed, error) {
	// What is composed here already may be what this suite wants: ask before
	// rebuilding. The asking, and the four things the answer leads to, are the
	// comparison block of the chain's own lifecycle — this used to be a switch
	// over four verdicts written here, which could say what was done but not
	// where the run was when it did it.
	res, err := verb.ChainUpComparing(ctx, sd, verb.CompareIn{Up: up})
	out.Preflight = res.Decision
	out.SetupSteps = append(out.SetupSteps, res.Steps...)
	if err != nil {
		// The chain's own state rides on the result, not in the sentence. The
		// surface prints one suffix for both areas.
		out.ComposeFailedAt = res.At
		return composed{}, fmt.Errorf("engine: run suite: setup: %w", err)
	}

	return readWorkspaceComposed(ctx, sd, up.DataDir, up.KeysDir, &out.SetupSteps, gateBudget)
}

// readWorkspaceComposed reads the network a workspace has composed — endpoints,
// advertised capabilities, the full node table (with dial addresses translated
// to the reachable endpoints), and a control over the recorded processes — then
// gates it ready (E6). It is the shared tail of both composing a network and
// attaching to one an existing workspace already brought up (WA10).
func readWorkspaceComposed(ctx context.Context, sd chainsetup.Deps, dataDir, keysDir string, setupSteps *[]string, gateBudget time.Duration) (composed, error) {
	endpoints, err := verb.ChainEndpoints(ctx, sd, verb.ChainEndpointsIn{DataDir: dataDir})
	if err != nil {
		return composed{}, lifecycle.Mark(errUnreachable, fmt.Errorf("engine: run suite: endpoints: %w", err))
	}
	var caps []string
	if ws, err := chainsetup.Open(dataDir, sd.Clock); err == nil {
		caps = ws.State().Capabilities
	}
	// The workspace knows the whole node table — indices, hosts, every
	// endpoint — so the engine attaches to that, not to bare URLs, and fault
	// steps get a control over the recorded processes. NodeSet's RPCURL is the
	// reachable, dial-time-translated address (the same ChainEndpoints returns), so
	// a docker/remote run attaches to the right endpoint with no re-translation
	// here.
	var nodes *node.NodeSet
	if st, err := verb.NetworkStatus(ctx, sd, verb.NetworkStatusIn{DataDir: dataDir}); err == nil && len(st.Nodes.Nodes) > 0 {
		ns := st.Nodes
		nodes = &ns
	}
	// The network is composed (or reused); gate it before any test runs on it —
	// wait on nodes still coming up, restart dead ones within limits, terminate
	// on a state that would need a destructive remedy (E6).
	out := composed{
		endpoints: endpoints,
		caps:      caps,
		teardown: func(ctx context.Context) error {
			_, err := verb.ChainStop(ctx, sd, verb.ChainStopIn{DataDir: dataDir})
			return err
		},
		nodes:   nodes,
		control: workspaceNodes{sd: sd, dataDir: dataDir},
		keysDir: keysDir,
	}
	// Where this network's declared fork is and who hands over at it. Read from
	// the record, so attaching to a composed network knows it too.
	if f, ferr := verb.ChainFork(ctx, sd, verb.ChainForkIn{DataDir: dataDir}); ferr == nil {
		switch {
		case f.Fork != nil:
			out.fork = forkGate{at: f.Fork.At, restart: f.Fork.Restart, preFork: make(map[int]bool, len(f.PreFork))}
			for _, i := range f.PreFork {
				out.fork.preFork[i] = true
			}
		case f.HaltsAt > 0:
			// A network whose genesis stops it stands at the same place a
			// handover does — every node one block short, nothing advancing —
			// and nobody hands over. Same gate, no pre-fork side.
			out.fork = forkGate{at: f.HaltsAt}
		}
	}
	// The gate can fail on a network that is already up — nodes launched, one
	// of them not answering yet — so the way to take it down is returned WITH
	// the error rather than dropped with it. Returning the zero value here left
	// four nodes holding their ports after every readiness failure, and the
	// next run on those ports could not compose at all.
	if err := gateReady(ctx, sd, dataDir, nodes, setupSteps, gateBudget, out.fork); err != nil {
		return out, fmt.Errorf("engine: run suite: %w", err)
	}
	return out, nil
}

// remoteLogReader returns an SSH-backed log reader for a remote target's node
// logs, which the collector wraps so a dropped session reconnects (E8); a local
// target (or a lookup error — collection is best-effort) returns nil, and the
// collector reads the local filesystem.
func remoteLogReader(sd chainsetup.Deps, dataDir string) collector.LogReader {
	runner, err := verb.NetworkRunner(sd, dataDir)
	if err != nil || runner == nil {
		return nil
	}
	return process.NewRemoteLogReader(runner)
}

// casesCrossFork reports whether any case names the step that crosses the
// hardfork, which is how a case says the moment is its own.
//
// Read from the runtime statement list rather than the document, so a v1 spec
// and a v2 case are read the same way and a step's spelling is resolved once.
// A case that puts the step in its pre-actions is not counted and the
// composition crosses first; the step is idempotent, so that case still runs —
// it simply does not get to act before the fork, which is the thing it asked
// for by putting it there.
func casesCrossFork(specs []dsl.Spec) bool {
	for _, sp := range specs {
		for _, st := range sp.Sequence {
			if st.Do == dsl.ActionCrossFork {
				return true
			}
		}
	}
	return false
}
