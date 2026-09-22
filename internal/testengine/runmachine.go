package testengine

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// A run's own states.
//
// The six stages are not invented here: design-v3/state-machine-05-test-failures
// counted 47 failure sites in a run and found they fall into six places, each
// pointing somewhere different to go and fix it. Those six are these states, so
// a run's position and the classification of its failures are one thing instead
// of a walk and a table that have to agree.
const (
	nameRun                statemachine.StateName = "Run"
	namePending            statemachine.StateName = "Pending"
	nameReadingDeclaration statemachine.StateName = "ReadingDeclaration"
	nameOpeningSession     statemachine.StateName = "OpeningSession"
	nameReachingNetwork    statemachine.StateName = "ReachingNetwork"
	nameComposingNetwork   statemachine.StateName = "ComposingNetwork"
	nameAttachingToNetwork statemachine.StateName = "AttachingToNetwork"
	namePreparing          statemachine.StateName = "Preparing"
	nameRunningCases       statemachine.StateName = "RunningCases"
	nameCollecting         statemachine.StateName = "Collecting"
	nameDone               statemachine.StateName = "Done"
	nameRunFailed          statemachine.StateName = "Failed"
)

// runner walks one suite.
//
// It holds what the states hand each other. A run is a sequence of things that
// each need what came before — the specs, the session, the network — and those
// used to be local variables in one long function, which is why the function
// was one long function.
type runner struct {
	m   *statemachine.Machine
	sd  chainsetup.Deps
	in  RunSuiteIn
	out RunSuiteOut

	// What the stages produce, in the order they produce it.
	raw   [][]byte
	specs []dsl.Spec
	comp  composition
	plan  ComposePlan
	sess  session.Session
	// keysDir is where an attached network's key set lives, read from the
	// workspace it was composed by.
	keysDir string
	net     composed

	// failure is why the run stopped, written just before the move to failed.
	failure error
	// failedIn is the path of the stage it stopped in. It is taken when the
	// stage reports, not at the end: by then the machine has moved on to
	// collecting, which is where every run ends and says nothing about why.
	failedIn string
	// networkUp says whether there is a network to take down. A run that failed
	// before one came up has nothing to collect from and nothing to stop.
	networkUp bool
	// attach says this run did not build its network and does not own it: it
	// found one an existing workspace had already brought up.
	attach bool

	pending   *pendingState
	reading   *readingDeclaration
	opening   *openingSession
	reaching  *reachingNetwork
	preparing *preparingRun
	running   *runningCases
	collect   *collecting
	done      *doneState
	failed    *runFailedState
}

// newRunner builds a run machine.
//
// The Add calls are the tree, written as the tree.
func newRunner(sd chainsetup.Deps, in RunSuiteIn) *runner {
	r := &runner{m: statemachine.New("run", nil), sd: sd, in: in}
	root := &runState{r: r}
	r.pending = &pendingState{r: r}
	r.reading = &readingDeclaration{r: r}
	r.opening = &openingSession{r: r}
	r.reaching = newReachingNetwork(r)
	r.preparing = &preparingRun{r: r}
	r.running = &runningCases{r: r}
	r.collect = &collecting{r: r}
	r.done = &doneState{r: r}
	r.failed = &runFailedState{r: r}

	r.m.Add(root, nil)
	r.m.Add(r.pending, root)
	r.m.Add(r.reading, root)
	r.m.Add(r.opening, root)
	r.m.Add(r.reaching, root)
	for _, leaf := range r.reaching.leafStates() {
		r.m.Add(leaf, r.reaching)
	}
	r.m.Add(r.preparing, root)
	r.m.Add(r.running, root)
	r.m.Add(r.collect, root)
	r.m.Add(r.done, root)
	r.m.Add(r.failed, root)
	return r
}

// Run walks the suite and reports what it did.
func (r *runner) Run(ctx context.Context) (RunSuiteOut, error) {
	if err := r.m.Start(ctx, r.pending); err != nil {
		return r.out, err
	}
	if err := r.m.Send(ctx, startRun{}); err != nil {
		return r.out, err
	}
	if r.failure != nil {
		r.out.FailedAt = r.failedIn
	}
	return r.out, r.failure
}

// fail records why a stage could not finish and moves on to collecting what is
// left, which is the same thing a finished run does.
//
// Failing does not skip the collecting. Setting a network up is not
// all-or-nothing: the nodes launch and then a gate refuses what came up, and
// everything that would say why is on those nodes. Gathering first and stopping
// second is the order that keeps it.
func (r *runner) fail(m *statemachine.Machine, in statemachine.State, err error) {
	r.failure = err
	// The state being entered, not the current one: current changes when the
	// whole move is done, so during an Enter it still names the state being
	// left.
	r.failedIn = r.m.Path(in)
	m.SendSelf(stageStopped{})
}

// chain is the chain this run's cases address nodes and accounts against.
//
// A composing run reads it from the declaration it composed from; an attaching
// one from the workspace whose network it joined, because it never parsed a
// declaration at all.
func (r *runner) chain() string {
	if len(r.specs) > 0 {
		return r.specs[0].Chain.Name
	}
	return r.in.Chain
}

// readAttachTarget reads what an attaching run needs from the workspace whose
// network it is joining: the chain, and where that network's keys live.
func (r *runner) readAttachTarget() error {
	if r.in.DataDir == "" {
		return lifecycle.Mark(errUnreadable, fmt.Errorf("engine: attach workspace: a workspace directory is required"))
	}
	if ws, err := chainsetup.Open(r.in.DataDir, r.sd.Clock); err == nil {
		st := ws.State()
		r.keysDir = st.KeysDir
		if r.in.Chain == "" {
			r.in.Chain = st.Chain
		}
	}
	if r.in.Chain == "" {
		return lifecycle.Mark(errIncomplete, fmt.Errorf("engine: attach workspace: a chain is required to attach"))
	}
	// The documents as given: an attaching run does not parse them, because
	// what they declare about a network is not this run's to act on.
	r.raw = r.in.SpecContent
	return nil
}

// runState is the root. It turns a stage's failure into the move to collecting.
type runState struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (runState) Name() statemachine.StateName { return nameRun }

// Process takes a stage's stop from anywhere below.
func (s *runState) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	if _, ok := msg.(stageStopped); !ok {
		return false, nil
	}
	m.TransitionTo(s.r.collect)
	return true, nil
}

// pendingState is a run that has not begun.
type pendingState struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (pendingState) Name() statemachine.StateName { return namePending }

// Process begins the run.
func (s *pendingState) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	if _, ok := msg.(startRun); !ok {
		return false, nil
	}
	m.TransitionTo(s.r.reading)
	return true, nil
}

// readingDeclaration turns the documents into a request and a plan.
type readingDeclaration struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (readingDeclaration) Name() statemachine.StateName { return nameReadingDeclaration }

// Enter reads the specs and settles what network they ask for.
//
// A run that attaches settles less: the network is already there, so what it
// declares about one is not this run's to act on. What it still needs is the
// chain, because a spec addresses nodes by role and resolves account labels
// against it.
func (s *readingDeclaration) Enter(ctx context.Context, m *statemachine.Machine) error {
	r := s.r
	if r.attach {
		if err := r.readAttachTarget(); err != nil {
			r.fail(m, s, err)
			return nil
		}
		m.SendSelf(declarationRead{})
		return nil
	}
	if len(r.in.SpecPaths) == 0 && len(r.in.SpecContent) == 0 {
		r.fail(m, s, fmt.Errorf("engine: run suite: no specs given"))
		return nil
	}
	raw, specs, comp, err := resolveComposition(ctx, r.in)
	if err != nil {
		r.fail(m, s, err)
		return nil
	}
	r.raw, r.specs, r.comp = raw, specs, comp
	// Announced before the first byte is written: the merge that produced this
	// happened across three layers and none of them is the file the operator
	// just named, so this is the only place the network can be seen whole while
	// it is still cheap to stop.
	r.plan = planOf(comp, specs[0].Chain.Name)
	if r.in.OnPlan != nil {
		r.in.OnPlan(r.plan)
	}
	// And kept, so the question survives the run that answered it.
	if perr := WritePlan(r.in.DataDir, r.plan); perr != nil {
		r.out.SetupSteps = append(r.out.SetupSteps, "plan: "+perr.Error())
	}
	m.SendSelf(declarationRead{})
	return nil
}

// Process moves on to opening the session.
func (s *readingDeclaration) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	if _, ok := msg.(declarationRead); !ok {
		return false, nil
	}
	m.TransitionTo(s.r.opening)
	return true, nil
}

// openingSession gives the run somewhere to write itself down.
type openingSession struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (openingSession) Name() statemachine.StateName { return nameOpeningSession }

// Enter opens the session.
//
// Before the network, not after. Everything from here on — including a network
// that never comes up — is this run, and a run that has nowhere to write is a
// run nobody can debug afterwards.
func (s *openingSession) Enter(_ context.Context, m *statemachine.Machine) error {
	r := s.r
	root, err := artifactRoot(r.in.ArtifactRoot, r.in.WorkspaceConfigPath, r.in.DataDir)
	if err != nil {
		r.fail(m, s, lifecycle.Mark(errNoRoot, err))
		return nil
	}
	r.in.ArtifactRoot = root
	if r.attach {
		// A run that attaches records itself in the session the engine makes,
		// because it has no setup of its own to write down before the cases
		// start -- the network was up before this run existed.
		m.SendSelf(sessionOpened{})
		return nil
	}
	sess, err := session.New(root, engineCommand, r.sd.Now())
	if err != nil {
		r.fail(m, s, fmt.Errorf("engine: run suite: %w", err))
		return nil
	}
	r.sess = sess
	r.out.SessionRoot = sess.Root()
	m.SendSelf(sessionOpened{})
	return nil
}

// Process moves on to reaching the network.
func (s *openingSession) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	if _, ok := msg.(sessionOpened); !ok {
		return false, nil
	}
	m.TransitionTo(s.r.reaching)
	return true, nil
}

// reachingNetwork gets the network the run needs, however it gets one.
//
// Composing one and finding one already up are the same stage done two ways:
// both end with a network held to the same readiness gate, and a run that
// attached has reached a network exactly as much as one that built it. Which
// way it went is what the two leaves say.
type reachingNetwork struct {
	statemachine.Base
	r *runner

	composing *composingNetwork
	attaching *attachingToNetwork
}

// newReachingNetwork builds the stage and the two ways it can go.
func newReachingNetwork(r *runner) *reachingNetwork {
	s := &reachingNetwork{r: r}
	s.composing = &composingNetwork{r: r}
	s.attaching = &attachingToNetwork{r: r}
	return s
}

// Name says what this state is called.
func (reachingNetwork) Name() statemachine.StateName { return nameReachingNetwork }

// leafStates is the two ways to reach a network, in the order the tree shows.
func (s *reachingNetwork) leafStates() []statemachine.State {
	return []statemachine.State{s.composing, s.attaching}
}

// Enter chooses which way this run reaches its network.
func (s *reachingNetwork) Enter(_ context.Context, m *statemachine.Machine) error {
	m.SendSelf(networkWayChosen{})
	return nil
}

// Process goes the way the request implies.
func (s *reachingNetwork) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	switch msg.(type) {
	case networkWayChosen:
		if s.r.attach {
			m.TransitionTo(s.attaching)
			return true, nil
		}
		m.TransitionTo(s.composing)
		return true, nil
	case networkReached:
		// A run that attached prepares nothing: the fork, the height and the
		// accounts belong to whoever brought the network up.
		if s.r.attach {
			m.TransitionTo(s.r.running)
			return true, nil
		}
		m.TransitionTo(s.r.preparing)
		return true, nil
	}
	return false, nil
}

// composingNetwork builds the network the declaration asked for.
type composingNetwork struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (composingNetwork) Name() statemachine.StateName { return nameComposingNetwork }

// Enter composes the network and checks it is the one the plan described.
func (s *composingNetwork) Enter(ctx context.Context, m *statemachine.Machine) error {
	r := s.r
	net, err := composeWorkspace(ctx, r.sd, *r.comp.up, &r.out, r.in.NodeMonitorTimeout)
	r.net = net
	r.networkUp = true
	if verr := verifyAgainstPlan(r.plan, r.comp.up.DataDir, &r.out); err == nil && verr != nil {
		r.fail(m, s, verr)
		return nil
	}
	if err != nil {
		// The chain area refused. Which of its stages did is its own state's,
		// recorded by the compose itself, and this one does not repeat it.
		r.fail(m, s, err)
		return nil
	}
	r.out.Endpoints = net.endpoints
	m.SendSelf(networkReached{})
	return nil
}

// attachingToNetwork finds the network an existing workspace brought up.
type attachingToNetwork struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (attachingToNetwork) Name() statemachine.StateName { return nameAttachingToNetwork }

// Enter reads the network the workspace already has, and holds it to the same
// readiness gate a composed one is held to.
func (s *attachingToNetwork) Enter(ctx context.Context, m *statemachine.Machine) error {
	r := s.r
	net, err := readWorkspaceComposed(ctx, r.sd, r.in.DataDir, r.keysDir, &r.out.SetupSteps, r.in.NodeMonitorTimeout)
	if err != nil {
		r.fail(m, s, err)
		return nil
	}
	r.net = net
	r.networkUp = true
	r.out.Endpoints = net.endpoints
	m.SendSelf(networkReached{})
	return nil
}

// preparingRun makes the chain ready for the cases to run against.
type preparingRun struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (preparingRun) Name() statemachine.StateName { return namePreparing }

// Enter crosses the declared fork, waits for the height and funds the accounts.
func (s *preparingRun) Enter(ctx context.Context, m *statemachine.Machine) error {
	r := s.r
	if err := prepareChain(ctx, r.sd, r.in, r.comp, r.specs, r.net, &r.out); err != nil {
		r.fail(m, s, err)
		return nil
	}
	m.SendSelf(chainPrepared{})
	return nil
}

// Process moves on to running the cases.
func (s *preparingRun) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	if _, ok := msg.(chainPrepared); !ok {
		return false, nil
	}
	m.TransitionTo(s.r.running)
	return true, nil
}

// runningCases runs the declared cases against the network.
type runningCases struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (runningCases) Name() statemachine.StateName { return nameRunningCases }

// Enter runs every case.
//
// A case reporting a failure is a verdict and not a failure of the run; what
// fails here is the running itself.
func (s *runningCases) Enter(ctx context.Context, m *statemachine.Machine) error {
	r := s.r
	if err := runCases(ctx, r.sd, r.in, r.chain(), r.raw, r.net, r.sess, &r.out); err != nil {
		r.fail(m, s, err)
		return nil
	}
	m.SendSelf(casesRun{})
	return nil
}

// Process moves on to collecting.
func (s *runningCases) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	if _, ok := msg.(casesRun); !ok {
		return false, nil
	}
	m.TransitionTo(s.r.collect)
	return true, nil
}

// collecting gathers what the run is judged on and takes the network down.
//
// Every run passes through here, the ones that failed included. Gathering is
// what a failure is read from, and it has to happen while the nodes still
// answer — taking them down is exactly what stops them answering.
type collecting struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (collecting) Name() statemachine.StateName { return nameCollecting }

// Enter gathers and tears down.
func (s *collecting) Enter(ctx context.Context, m *statemachine.Machine) error {
	r := s.r
	if r.failure != nil && r.networkUp && r.sess != nil {
		recordBlockedBySetup(ctx, r.sd, r.comp.up.DataDir, r.net, &r.out, r.failure,
			blockedRun{sess: r.sess, raw: r.raw, specs: r.specs})
	}
	if !r.in.KeepUp && r.net.teardown != nil {
		if err := r.net.teardown(ctx); err == nil {
			r.out.Stopped = true
		} else if r.failure == nil {
			r.failure = fmt.Errorf("engine: run suite: teardown: %w", err)
		} else {
			r.failure = fmt.Errorf("%w (and the network could not be taken down: %v)", r.failure, err)
		}
	}
	m.SendSelf(collected{})
	return nil
}

// Process ends the run, either way.
func (s *collecting) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	if _, ok := msg.(collected); !ok {
		return false, nil
	}
	if s.r.failure != nil {
		m.TransitionTo(s.r.failed)
		return true, nil
	}
	m.TransitionTo(s.r.done)
	return true, nil
}

// doneState is a run that finished, whatever its cases reported.
type doneState struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (doneState) Name() statemachine.StateName { return nameDone }

// runFailedState is a run that could not go on.
//
// Which stage it could not go on from is the path, not a separate
// classification: the six states ARE the six places a run's failures fall into.
type runFailedState struct {
	statemachine.Base
	r *runner
}

// Name says what this state is called.
func (runFailedState) Name() statemachine.StateName { return nameRunFailed }
