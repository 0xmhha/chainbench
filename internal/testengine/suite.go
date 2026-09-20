package testengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/home"
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
	// BPCount overrides the bp node count the specs declare.
	BPCount int
	// Server selects where the nodes run, from the operator's server set.
	Server resource.ServerRef
	// Docker treats the servers as local docker containers (the option is the
	// power switch, as everywhere).
	Docker bool
	// KeysDir overrides the declared key set (default keys/preset).
	KeysDir string
	// KeysSource overrides where node identities come from ("keyPreset" or
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
	// NodeMonitorTimeout, when positive, is how long the readiness gate waits on
	// nodes still coming up before it gives up (E6). Zero takes the default; a
	// large or slow bring-up (a 15-node poa network over docker, whose late
	// endpoints sync slowly) raises it so the gate does not terminate a network
	// that is merely still forming.
	NodeMonitorTimeout time.Duration
	// Env, when set, moves every case onto that chain declaration instead of
	// the one it names. It is an env id, or a path to an env file.
	//
	// It is what lets one set of cases meet more than one chain: the steps say
	// nothing about which mainnet they are on, so the only thing that has to
	// change is which declaration they compose against. A case's own overrides
	// survive the swap, because those belong to the test.
	Env string
	// OnPlan, when set, is handed the merged composition plan after the
	// declaration and these overrides are resolved and before anything is
	// written or launched. It is the seam the CLI prints through: a library
	// that writes to a terminal cannot be used by one that does not have one.
	// Nil runs silently.
	OnPlan func(ComposePlan)
	// WorkspaceConfigPath is the environment file (--workspace-config) that owns
	// the target dataRoot and its purpose directories. When set, its dataRoot is
	// the target's data root, so the same DSL runs across targets by swapping
	// this file. Empty keeps the local default (the workspace directory).
	WorkspaceConfigPath string
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
	// fork is what the readiness gate has to know about a network composed to
	// cross a hardfork: where the fork is, and which nodes hand over at it.
	// Zero for a network that crosses none.
	fork forkGate
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
		Binary: change.Binary, Config: change.Config,
		GenesisOverlay: change.GenesisOverlay, Purpose: change.Purpose,
	})
	if err != nil {
		return n, err
	}
	return out.Node, nil
}

// CrossFork waits for the network to reach the block before its declared
// hardfork and hands production to the build that seals after it, satisfying
// interp.ForkCrosser so the crossFork action reaches it.
//
// The whole table comes back: crossing gives every successor a new role and a
// new pid, and a caller holding the old ones would stop the wrong process.
func (w workspaceNodes) CrossFork(ctx context.Context, timeout time.Duration) ([]node.Node, error) {
	out, err := chainsetup.NetCrossFork(ctx, w.sd, chainsetup.NetCrossForkIn{
		DataDir: w.dataDir, Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return out.Nodes.Nodes, nil
}

// Log returns the tail of one node's captured stdout/stderr, satisfying
// interp.NodeLogReader. It is what lets a spec say WHY a node is not up: a node
// that refuses its genesis prints the reason and exits, and the process manager
// sees only an exit.
//
// The log lives in the workspace this suite composed, under the conventional
// per-node label. A node that has never been launched has no log file, which is
// not an error — it reads as an empty log.
func (w workspaceNodes) Log(_ context.Context, n node.Node, maxBytes int) (string, error) {
	path := node.Layout{Root: w.dataDir}.LogPath(node.LabelFor(n.Index))
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("engine: open node%d log %s: %w", n.Index, path, err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("engine: stat node%d log: %w", n.Index, err)
	}
	size := info.Size()
	if maxBytes <= 0 || int64(maxBytes) > size {
		maxBytes = int(size)
	}
	if maxBytes == 0 {
		return "", nil
	}
	if _, err := f.Seek(size-int64(maxBytes), io.SeekStart); err != nil {
		return "", fmt.Errorf("engine: seek node%d log: %w", n.Index, err)
	}
	buf := make([]byte, maxBytes)
	read, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("engine: read node%d log: %w", n.Index, err)
	}
	return string(buf[:read]), nil
}

// verifyAgainstPlan holds the launched network to the plan and refuses to test
// one that is not it.
//
// Every other check in this package asks whether the declaration is coherent.
// This is the only one that asks whether the network that came up is the one
// described, which matters because the last word belongs to the command line:
// an override naming no layer beats every document, so the merge can be right
// and the nodes still run something else. A test against the wrong network
// does not fail, it answers a question nobody asked.
//
// Reading the record rather than the in-memory state is deliberate: the record
// is what a later reader sees, so a fact that never reached it is a fact the
// run cannot show afterwards either.
func verifyAgainstPlan(plan ComposePlan, dir string, out *RunSuiteOut) error {
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		// The compose error, if there is one, says more than this would.
		return nil //nolint:nilerr // absence of a record is reported by the caller
	}
	bad := VerifyLaunched(plan, ws.State())
	if len(bad) == 0 {
		out.SetupSteps = append(out.SetupSteps, "verify: the launched network matches the plan")
		return nil
	}
	lines := make([]string, 0, len(bad))
	for _, m := range bad {
		lines = append(lines, m.String())
	}
	out.SetupSteps = append(out.SetupSteps, "verify: "+strings.Join(lines, "; "))
	return fmt.Errorf("engine: run suite: the launched network is not the one planned: %s", strings.Join(lines, "; "))
}

// resolveComposition is everything RunSuite does before it writes anything:
// read the specs, parse them, refuse a set that cannot share one network, run
// the pre-flight, and merge the declaration with the command's overrides.
//
// It is separate so that planning and running resolve the same way. A planner
// that repeated these steps would eventually answer a different question than
// the runner, which is the one thing a plan must never do.
func resolveComposition(ctx context.Context, in RunSuiteIn) ([][]byte, []dsl.Spec, composition, error) {
	if in.DataDir == "" {
		return nil, nil, composition{}, fmt.Errorf("engine: run suite: a workspace directory is required")
	}
	specs := in.SpecContent
	if len(specs) > 0 && in.Env != "" {
		return nil, nil, composition{}, fmt.Errorf("engine: run suite: --env moves a case onto another declaration, and inline spec content names no file to resolve it against")
	}
	if len(specs) == 0 {
		var err error
		if specs, err = dsl.ReadFilesWithEnv(in.SpecPaths, in.Env); err != nil {
			return nil, nil, composition{}, err
		}
	}
	parsed := make([]dsl.Spec, 0, len(specs))
	for i, raw := range specs {
		s, err := dsl.Parse(raw)
		if err != nil {
			return nil, nil, composition{}, fmt.Errorf("engine: run suite: spec %d: %w", i+1, err)
		}
		parsed = append(parsed, s)
	}
	if err := sameChain(parsed); err != nil {
		return nil, nil, composition{}, fmt.Errorf("engine: run suite: %w", err)
	}
	if err := sameComposition(parsed); err != nil {
		return nil, nil, composition{}, fmt.Errorf("engine: run suite: %w", err)
	}
	// Pre-flight before anything is allocated or written: a spec that names an
	// action/assertion/reader/reference that does not resolve, or a malformed
	// node selector, fails here rather than after a network is composed.
	if err := Precheck(parsed); err != nil {
		return nil, nil, composition{}, fmt.Errorf("engine: run suite: %w", err)
	}
	comp, err := compositionOf(ctx, parsed[0], in)
	if err != nil {
		return nil, nil, composition{}, fmt.Errorf("engine: run suite: %w", err)
	}
	return specs, parsed, comp, nil
}

// RunSuite runs the whole flow: read the DSL, compose the chain it declares
// through chainsetup, run the tests, collect, and stop the network unless
// asked to keep it. Setup failure aborts before any test runs; a test-phase
// failure still tears down.
func RunSuite(ctx context.Context, sd chainsetup.Deps, in RunSuiteIn) (RunSuiteOut, error) {
	if len(in.SpecPaths) == 0 && len(in.SpecContent) == 0 {
		return RunSuiteOut{}, fmt.Errorf("engine: run suite: no specs given")
	}
	specs, parsed, comp, err := resolveComposition(ctx, in)
	if err != nil {
		return RunSuiteOut{}, err
	}
	out := RunSuiteOut{}
	// Announced before the first byte is written: the merge that produced this
	// happened across three layers and none of them is the file the operator
	// just named, so this is the only place the network can be seen whole
	// while it is still cheap to stop.
	plan := planOf(comp, parsed[0].Chain.Name)
	if in.OnPlan != nil {
		in.OnPlan(plan)
	}
	// And kept, so the question survives the run that answered it. A handoff
	// composes from its profile into the same directory, so it is saved too.
	if err := WritePlan(in.DataDir, plan); err != nil {
		out.SetupSteps = append(out.SetupSteps, "plan: "+err.Error())
	}
	root, rerr := artifactRoot(in.ArtifactRoot, in.WorkspaceConfigPath, in.DataDir)
	if rerr != nil {
		return out, rerr
	}
	in.ArtifactRoot = root
	chain := parsed[0].Chain.Name

	// Before the network, not after. Everything from here on — including a
	// network that never comes up — is this run, and a run that has nowhere to
	// write is a run nobody can debug afterwards.
	sess, serr := session.New(root, engineCommand, sd.Now())
	if serr != nil {
		return out, fmt.Errorf("engine: run suite: %w", serr)
	}
	out.SessionRoot = sess.Root()

	net, err := composeWorkspace(ctx, sd, *comp.up, &out, in.NodeMonitorTimeout)
	blocked := blockedRun{sess: sess, raw: specs, specs: parsed}
	if verr := verifyAgainstPlan(plan, comp.up.DataDir, &out); err == nil && verr != nil {
		return out, afterFailedSetup(ctx, sd, comp.up.DataDir, net, in.KeepUp, &out, verr, blocked)
	}
	if err != nil {
		return out, afterFailedSetup(ctx, sd, comp.up.DataDir, net, in.KeepUp, &out, err, blocked)
	}
	out.Endpoints = net.endpoints

	runErr := func() error {
		// The declared fork is crossed before anything else runs, because
		// everything else assumes a producing chain: funding an account is a
		// transaction, and a network sitting at the block before its fork seals
		// none. A case that has to act BEFORE the fork says so by naming the
		// crossFork step, and then this leaves the fork to it.
		if comp.up.GenesisFork != nil && !casesCrossFork(parsed) {
			res, cerr := chainsetup.NetCrossFork(ctx, sd, chainsetup.NetCrossForkIn{DataDir: comp.up.DataDir})
			if cerr != nil {
				return fmt.Errorf("engine: run suite: %w", cerr)
			}
			out.SetupSteps = append(out.SetupSteps, "cross-fork: "+res.Detail)
		}
		if in.WaitBlocks > 0 {
			if err := waitForHead(ctx, net.endpoints[0], in.WaitBlocks); err != nil {
				return fmt.Errorf("engine: run suite: %w", err)
			}
		}
		// Declared test accounts are created and funded here: after the chain
		// seals (funding is a transaction) and before any spec runs (a spec
		// referring to one must find it).
		if len(parsed[0].EnvAccounts) > 0 {
			ring, rerr := ringFor(net.keysDir)
			if rerr != nil {
				return fmt.Errorf("engine: run suite: accounts: %w", rerr)
			}
			funder, ok := ring.Get("node1")
			if !ok {
				return fmt.Errorf("engine: run suite: accounts: the key set has no node1 to fund from")
			}
			if err := prepareAccounts(ctx, ring, net.keysDir, net.endpoints[0], parsed[0].EnvAccounts, funder.Address); err != nil {
				return fmt.Errorf("engine: run suite: %w", err)
			}
		}
		eng, err := wiredAttachEngine(sd, net, attachWiring{
			Chain: chain, DataDir: in.DataDir, ArtifactRoot: in.ArtifactRoot,
			Caps: in.Caps, NodeMonitorTimeout: in.NodeMonitorTimeout, SetupSteps: &out.SetupSteps,
			Session: sess,
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

// blockedRun is the run a failed setup never got to: the session it would have
// been recorded in, and the specs that will not run.
type blockedRun struct {
	sess  session.Session
	raw   [][]byte
	specs []dsl.Spec
}

// afterFailedSetup records the failure as the tests' own, gathers what the
// network can still say, then takes it down, and returns the setup error either
// way.
//
// Setting a network up is not all-or-nothing: the nodes launch and then a
// readiness gate, or the plan check, refuses what came up. Two things used to go
// wrong at once. Nothing was collected, because evidence gathering hung on a
// test record and no test had started — so "1 node still not ready" was the
// entire account of the failure, and which node, and why, was gone. And the
// nodes were left running, holding the ports and the datadirs, so the next run
// could not compose either.
//
// Order matters: gather first. The health probe needs the nodes answering, and
// taking them down is exactly what stops them answering.
//
// --keep-up still keeps the network. A failure is when an operator wants to
// look, and the flag says they will.
func afterFailedSetup(ctx context.Context, sd chainsetup.Deps, dataDir string, net composed, keepUp bool, out *RunSuiteOut, setupErr error, blocked blockedRun) error {
	recordBlockedBySetup(ctx, sd, dataDir, net, out, setupErr, blocked)
	if keepUp || net.teardown == nil {
		return setupErr
	}
	if err := net.teardown(ctx); err != nil {
		return fmt.Errorf("%w (and the network could not be taken down: %v)", setupErr, err)
	}
	return setupErr
}

// recordBlockedBySetup writes the setup failure into the run's artifacts: every
// spec that was going to run gets its folder, a blocked verdict naming the setup
// error, and the gathered evidence under its observations/.
//
// Under the test, because a network is composed for the test that asked for it.
// Five definitions given to one command are five attempts, each composing its
// own network, and a failure filed anywhere else — a folder named after the
// chain, a shared dump beside the workspace — severs the one link a reader
// needs: which attempt this was. The evidence is copied per test for the same
// reason, and it is bulky on purpose.
func recordBlockedBySetup(ctx context.Context, sd chainsetup.Deps, dataDir string, net composed, out *RunSuiteOut, setupErr error, blocked blockedRun) {
	if blocked.sess == nil || len(blocked.specs) == 0 {
		return
	}
	ev := gatherFailureData(ctx, sd, dataDir, net.nodes)
	for i, spec := range blocked.specs {
		rec := blocked.sess.Test(i+1, spec.ID)
		if i < len(blocked.raw) {
			rec.Spec(blocked.raw[i])
		}
		rec.Status(session.StatusBlocked)
		rec.Reason(setupErr.Error())
		for _, e := range ev {
			rec.Observation(e.Name, e.Data)
		}
	}
	if err := blocked.sess.Save(); err != nil {
		out.SetupSteps = append(out.SetupSteps, "evidence: "+err.Error())
		return
	}
	out.SetupSteps = append(out.SetupSteps, "evidence: "+blocked.sess.Root())
	if sum, serr := ReadSessionSummary(blocked.sess.Root()); serr == nil {
		out.Summary = sum
	}
}

// artifactRoot decides where the session lands, in the layers this track uses
// everywhere else: the harness default, then what a declaration said, then what
// the invocation said.
//
// The workspace-config layer was written and never read. The file requires
// control.artifactRoot — an empty one is refused — and nothing called the
// resolver, so an operator was told to name a path that was then ignored. That
// is the failure this track keeps removing: a declaration that does not reach
// the run.
//
// A configured root that cannot be created is an error rather than a fall back
// to the default. Falling back would put the results somewhere the operator did
// not ask for and say nothing, and they would go looking in the path they wrote.
//
// The harness default is ~/.chainbench/sessions, not a directory inside the
// workspace. What a run produced and what it ran on have different lifetimes: a
// workspace is scratch and gets removed to reclaim the disk or to force a clean
// compose, and that used to take every verdict and every log with it. The
// workspace is still where the chain is; this is where the record of testing it
// is kept.
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
	if in.DataDir == "" {
		return "", fmt.Errorf("engine: attach workspace: a workspace directory is required")
	}
	chain := in.Chain
	keysDir := ""
	if ws, err := chainsetup.Open(in.DataDir, sd.Clock); err == nil {
		st := ws.State()
		keysDir = st.KeysDir
		if chain == "" {
			chain = st.Chain
		}
	}
	if chain == "" {
		return "", fmt.Errorf("engine: attach workspace: a chain is required to attach")
	}
	artifactRoot, err := artifactRoot(in.ArtifactRoot, "", in.DataDir)
	if err != nil {
		return "", fmt.Errorf("engine: attach workspace: %w", err)
	}
	var setupSteps []string
	net, err := readWorkspaceComposed(ctx, sd, in.DataDir, keysDir, &setupSteps, in.NodeMonitorTimeout)
	if err != nil {
		return "", err
	}
	eng, err := wiredAttachEngine(sd, net, attachWiring{
		Chain: chain, DataDir: in.DataDir, ArtifactRoot: artifactRoot,
		Caps: in.Caps, NodeMonitorTimeout: in.NodeMonitorTimeout, SetupSteps: &setupSteps,
	})
	if err != nil {
		return "", fmt.Errorf("engine: attach workspace: %w", err)
	}
	return eng.Run(ctx, in.Specs)
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
func composeWorkspace(ctx context.Context, sd chainsetup.Deps, up chainsetup.NetUpIn, out *RunSuiteOut, gateBudget time.Duration) (composed, error) {
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
		// RebuildAll means a network-wide fact differs, and the most common one
		// is the genesis — a different chain. The compose steps alone do not
		// deliver that: init and start SKIP a node that still carries a recorded
		// pid, so a second up over a workspace whose nodes are still running
		// rewrites the genesis on disk and leaves every node serving the old one.
		// Measured: the verdict read "rebuild-all: genesis differs", the
		// workspace genesis had applepieBlock 0, and all four running nodes
		// reported "Applepie: #<nil>" with one startup each.
		//
		// So the network is stopped first, which is what makes "rebuild all"
		// true. Only on RebuildAll: Compose has nothing to stop, and
		// RebuildNodes is handled above, per node.
		if decision.Verdict == preflight.RebuildAll {
			st, serr := chainsetup.NetStop(ctx, sd, chainsetup.NetStopIn{DataDir: up.DataDir})
			if serr != nil {
				return composed{}, fmt.Errorf("engine: run suite: preflight stop before rebuild: %w", serr)
			}
			out.SetupSteps = append(out.SetupSteps, "stop (rebuild-all): "+st.Detail)
		}
		res, err := chainsetup.NetUp(ctx, sd, up)
		out.SetupSteps = append(out.SetupSteps, res.Steps...)
		if err != nil {
			return composed{}, fmt.Errorf("engine: run suite: setup: %w", err)
		}
	}

	return readWorkspaceComposed(ctx, sd, up.DataDir, up.KeysDir, &out.SetupSteps, gateBudget)
}

// readWorkspaceComposed reads the network a workspace has composed — endpoints,
// advertised capabilities, the full node table (with dial addresses translated
// to the reachable endpoints), and a control over the recorded processes — then
// gates it ready (E6). It is the shared tail of both composing a network and
// attaching to one an existing workspace already brought up (WA10).
func readWorkspaceComposed(ctx context.Context, sd chainsetup.Deps, dataDir, keysDir string, setupSteps *[]string, gateBudget time.Duration) (composed, error) {
	endpoints, err := chainsetup.NetEndpoints(ctx, sd, chainsetup.NetEndpointsIn{DataDir: dataDir})
	if err != nil {
		return composed{}, fmt.Errorf("engine: run suite: endpoints: %w", err)
	}
	var caps []string
	if ws, err := chainsetup.Open(dataDir, sd.Clock); err == nil {
		caps = ws.State().Capabilities
	}
	// The workspace knows the whole node table — indices, hosts, every
	// endpoint — so the engine attaches to that, not to bare URLs, and fault
	// steps get a control over the recorded processes. NodeSet's RPCURL is the
	// reachable, dial-time-translated address (the same NetEndpoints returns), so
	// a docker/remote run attaches to the right endpoint with no re-translation
	// here.
	var nodes *node.NodeSet
	if st, err := chainsetup.NetworkStatus(ctx, sd, chainsetup.NetworkStatusIn{DataDir: dataDir}); err == nil && len(st.Nodes.Nodes) > 0 {
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
			_, err := chainsetup.NetStop(ctx, sd, chainsetup.NetStopIn{DataDir: dataDir})
			return err
		},
		nodes:   nodes,
		control: workspaceNodes{sd: sd, dataDir: dataDir},
		keysDir: keysDir,
	}
	// Where this network's declared fork is and who hands over at it. Read from
	// the record, so attaching to a composed network knows it too.
	if f, ferr := chainsetup.NetFork(ctx, sd, chainsetup.NetForkIn{DataDir: dataDir}); ferr == nil {
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
	runner, err := chainsetup.NetRunner(sd, dataDir)
	if err != nil || runner == nil {
		return nil
	}
	return process.NewRemoteLogReader(runner)
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
