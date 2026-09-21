package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Crossing the fork a network is composed to cross.
//
// Up to the fork block the post-fork build is an endpoint: it syncs and does
// not seal. That is not a simplification, it is what the chain requires. Before
// the fork its consensus has no validator set — the section that names one is
// not active yet — so a node asked to seal there signs a round-change every
// second, has it refused as an unauthorized address, and does it again for as
// long as the pre-fork chain runs. Multiplied by the size of the successor set
// that is load the network is made to carry for nothing.
//
// So the handover is a moment, not a setting. The pre-fork build seals up to
// At-1 and stops; the successors are stopped there, told they produce now, and
// relaunched. Nothing on disk changes: the fork's configuration is already in
// the genesis (or in their configs), and their datadirs keep everything the
// chain built before the fork.
//
// A restart crosses the same moment with a different network behind it. There
// is one build at a time, so every node stops and comes back on the post-fork
// one, and what each node DOES is unchanged — the producers go on producing.
// The fork's configuration reaches them through their config files, which are
// read on every launch; a genesis document would not be read at all, because
// their databases already hold the pre-fork chain.
//
// Whose moment it is depends on the case. A case that only wants a chain past
// the fork lets the composition cross it and never mentions it. A case that
// has to act before the fork — send a transaction, deploy a contract, and then
// ask whether it survived — names the step and crosses when it is ready. Both
// call this, so the two cannot drift into doing different things.

const (
	// crossForkPoll is how often the boundary is re-read. The chains here seal
	// about once a second, so this notices the stop within a block.
	crossForkPoll = 1 * time.Second
	// crossForkTimeout is the default wait for a network to reach the block
	// before its fork.
	crossForkTimeout = 5 * time.Minute
)

// CrossForkOpts is how long to wait for the network to reach the fork.
type CrossForkOpts struct {
	// Timeout bounds the wait for the pre-fork build to reach At-1; zero is
	// crossForkTimeout.
	Timeout time.Duration
}

// CrossFork waits for the network to reach the block before its fork and hands
// production to the build that seals after it.
//
// It is idempotent: a network whose successors already produce is reported as
// already across rather than bounced, so a case that names the step on a
// composition that crossed on its own does not restart a working chain.
// The kinds of failure crossing a fork has.
//
// Four, and they are the three moments the twenty failure sites fall into plus
// the one that says there is nothing to cross. Which moment a run stopped in is
// what says what to do next: a fork nobody declared is a document to fix, a
// head that would not arrive is a network to look at, and a chain past the fork
// block cannot be crossed by this restart at all.
var (
	errCrossForkNoFork         = errors.New("this network crosses no fork")
	errCrossForkHeadUnreadable = errors.New("the chain head could not be read")
	errCrossForkAlreadyPast    = errors.New("the chain is past the fork block")
	errCrossForkNobodyCameBack = errors.New("the restart left no node running")
)

func (w *Workspace) CrossFork(ctx context.Context, opts CrossForkOpts) (StepOut, error) {
	f := w.state.Fork
	if f == nil {
		return StepOut{}, ofKind(errCrossForkNoFork,
			fmt.Errorf("chainsetup: cross-fork: this network is composed to cross no fork — declare one under env.upgrade"))
	}
	successors := w.forkSuccessors(*f)
	if len(successors) == 0 {
		if f.Restart {
			return StepOut{}, ofKind(errCrossForkNoFork,
				fmt.Errorf("chainsetup: cross-fork: this network has no node to restart across the %q fork", f.Name))
		}
		return StepOut{}, ofKind(errCrossForkNoFork,
			fmt.Errorf("chainsetup: cross-fork: no node runs binary %q, so the %q fork has nobody to hand over to", f.Binary, f.Name))
	}
	// passed is the moments this crossing goes through, appended where each is
	// reached. A run that stops partway returns what it got through, which is
	// what says which moment to look at.
	var passed []lifecycle.Status
	if w.alreadyCrossed(*f, successors) {
		detail := fmt.Sprintf("%s already crossed: %d node(s) are past it", f.Name, len(successors))
		w.markStep("cross-fork", detail)
		return StepOut{Detail: detail, Passed: []lifecycle.Status{
			lifecycle.ChainOpCrossForkBeforeFork,
			lifecycle.ChainOpCrossForkHandingOver,
			lifecycle.ChainOpCrossForkCrossed,
		}}, nil
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = crossForkTimeout
	}
	head, err := w.forkMoment(ctx, *f, timeout)
	if err != nil {
		return StepOut{Passed: passed}, err
	}
	passed = append(passed, lifecycle.ChainOpCrossForkBeforeFork)
	if err := w.handOver(ctx, *f, successors); err != nil {
		return StepOut{Passed: passed}, err
	}
	passed = append(passed, lifecycle.ChainOpCrossForkHandingOver)
	if err := w.confirmBeforeFork(ctx, *f); err != nil {
		return StepOut{Passed: passed}, err
	}
	passed = append(passed, lifecycle.ChainOpCrossForkCrossed)
	detail := fmt.Sprintf("%s at %d: head %d, %d successor(s) now produce", f.Name, f.At, head, len(successors))
	if f.Restart {
		detail = fmt.Sprintf("%s at %d: %d node(s) relaunched on %s at head %d, before the fork", f.Name, f.At, len(successors), f.Binary, head)
	}
	w.markStep("cross-fork", detail)
	return StepOut{Detail: detail, Passed: passed}, nil
}

// forkSuccessors is the node table's positions for the build that seals after
// the fork. By binary rather than by role, which is the same question
// binaryFor, genesisFor and pluginFor each answer — before the fork these nodes
// are endpoints, and after it they produce.
func (w *Workspace) forkSuccessors(f GenesisFork) []int {
	var out []int
	for i, ns := range w.state.Nodes {
		// A restart moves the whole network, so every node is on the list. It
		// is not "the nodes running the post-fork build" because none of them
		// is running it yet — that is what the restart does.
		if f.Restart || ns.Binary == f.Binary {
			out = append(out, i)
		}
	}
	return out
}

// alreadyCrossed reports whether every successor already produces. A partially
// crossed network is not "already crossed": the remaining ones still have to be
// told, or the set that reaches quorum is smaller than the genesis says.
func (w *Workspace) alreadyCrossed(f GenesisFork, successors []int) bool {
	for _, i := range successors {
		// What crossing changed is what the node is: the build it runs on a
		// restart, the work it does on a handover. Reading the other one would
		// report a restarted network as uncrossed, because a restart leaves
		// every role where it was.
		if f.Restart {
			if w.state.Nodes[i].Binary != f.Binary {
				return false
			}
			continue
		}
		if !node.Is(node.Role(w.state.Nodes[i].Role), node.RoleBP) {
			return false
		}
	}
	return true
}

// forkMoment brings the network to the point where it can be moved, and returns
// the head it was at.
//
// The two shapes reach that point from opposite directions, because only one of
// them stops. A handover waits: the pre-fork build refuses to seal the fork
// block, the chain stands at At-1, and the successors are told to produce there.
//
// A restart cannot wait for that. An ordinary fork does not halt the chain, it
// just activates, so the nodes have to be on the post-fork build BEFORE the
// fork block arrives — which is what an operator does, and when they do it.
// Waiting for At-1 would leave one block to swap four executables in, and
// whether that worked would be a matter of luck.
//
// So a restart moves now, and a chain already past its fork is too late: those
// blocks were crossed by the old build, and relaunching afterwards says nothing
// about which build was needed.
func (w *Workspace) forkMoment(ctx context.Context, f GenesisFork, timeout time.Duration) (int64, error) {
	if !f.Restart {
		return w.awaitForkBoundary(ctx, f, timeout)
	}
	obs, ok := w.forkObserver(f)
	if !ok {
		return 0, ofKind(errCrossForkNoFork,
			fmt.Errorf("chainsetup: cross-fork: no node is running, so there is nothing to restart across the %q fork", f.Name))
	}
	url, err := w.nodeHTTPURL(obs)
	if err != nil {
		return 0, fmt.Errorf("chainsetup: cross-fork: %w", err)
	}
	head, err := rpc.Dial(url).BlockNumber(ctx)
	if err != nil {
		return 0, ofKind(errCrossForkHeadUnreadable,
			fmt.Errorf("chainsetup: cross-fork: %s reached no head: %w", obs.NodeLabel(), err))
	}
	at := int64(head) //nolint:gosec // a chain head, compared against a declared block
	if at >= f.At {
		return 0, ofKind(errCrossForkAlreadyPast,
			fmt.Errorf("chainsetup: cross-fork: the chain is at block %d and the %q fork is at %d — the nodes had to be on the post-fork build before it, and crossing it on the pre-fork one proves nothing about the build that replaces it", at, f.Name, f.At))
	}
	return at, nil
}

// awaitForkBoundary waits for the chain to reach the last block before the
// fork, and returns the head it settled on.
//
// Read from a node on the PRE-fork side, because that is the side whose head
// answers the question. The pre-fork build refuses to seal the fork block —
// "skips mining due to <fork> hard fork" — so its head rises to At-1 and stops
// there. A successor's head says the same thing while it is still syncing, and
// says something else entirely once it is not.
func (w *Workspace) awaitForkBoundary(ctx context.Context, f GenesisFork, timeout time.Duration) (int64, error) {
	obs, ok := w.forkObserver(f)
	if !ok {
		return 0, ofKind(errCrossForkNoFork,
			fmt.Errorf("chainsetup: cross-fork: no node runs the pre-fork build, so there is nothing to read the fork boundary from"))
	}
	url, err := w.nodeHTTPURL(obs)
	if err != nil {
		return 0, fmt.Errorf("chainsetup: cross-fork: %w", err)
	}
	want := f.At - 1
	deadline := w.now().Add(timeout)
	client := rpc.Dial(url)
	var last int64 = -1
	var lastErr error
	for {
		head, err := client.BlockNumber(ctx)
		if err != nil {
			lastErr = err
		} else {
			lastErr = nil
			last = int64(head)
			if last >= want {
				return last, nil
			}
		}
		if !w.now().Before(deadline) {
			if lastErr != nil {
				return 0, ofKind(errCrossForkHeadUnreadable,
					fmt.Errorf("chainsetup: cross-fork: %s reached no head within %s: %w", obs.NodeLabel(), timeout, lastErr))
			}
			return 0, ofKind(errCrossForkAlreadyPast,
				fmt.Errorf("chainsetup: cross-fork: %s is at block %d after %s, and the %q fork is at %d — the chain never reached the block before it", obs.NodeLabel(), last, timeout, f.Name, f.At))
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(crossForkPoll):
		}
	}
}

// forkObserver is a running node on the pre-fork side of the handover.
func (w *Workspace) forkObserver(f GenesisFork) (node.Record, bool) {
	for _, ns := range w.state.Nodes {
		// On a restart every running node is on the pre-fork side, so the first
		// one that answers is the one to read.
		if (f.Restart || ns.Binary != f.Binary) && ns.PID > 0 {
			return ns, true
		}
	}
	return node.Record{}, false
}

// handOver stops each successor, records that it produces now, and relaunches
// it.
//
// The role is what changes, and it changes here rather than at composition for
// one reason: at composition the role also decides the genesis. A wemix genesis
// takes its governance members from the nodes declared bp, and a successor
// listed there registers in governance, never joins etcd, and stalls the
// producer. By now the genesis exists and is not rebuilt, so the same word can
// mean the one thing it means at launch — this node seals.
//
// The config is re-rendered because it, too, follows the role: the miner
// section and the account the node unlocks. Rendering it through the shared
// path keeps a handed-over node's config and its provenance the same shape a
// composed node's has.
func (w *Workspace) handOver(ctx context.Context, f GenesisFork, successors []int) error {
	p, err := w.plugin()
	if err != nil {
		return err
	}
	preset, placed, peering, pubkey, err := w.peerPlan(p)
	if err != nil {
		return fmt.Errorf("chainsetup: cross-fork: %w", err)
	}
	bin, err := w.binary("")
	if err != nil {
		return err
	}
	for _, i := range successors {
		ns := w.state.Nodes[i]
		t, terr := w.machineFor(ns)
		if terr != nil {
			return terr
		}
		if ns.PID > 0 {
			if serr := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: ns.PID}); serr != nil {
				return fmt.Errorf("chainsetup: cross-fork: stop %s: %w", ns.NodeLabel(), serr)
			}
			w.clearPID(i)
		}
		// The recorded argv is what the node started with, and it is reused on
		// every later launch. Clearing it is what makes the relaunch re-read
		// what changed instead of repeating the pre-fork command line.
		if f.Restart {
			// The build changes and the work does not. The DECLARED name is
			// written, not a per-node key: the fork's binary is one the env
			// declared, so binaryFor finds its path, pluginFor finds its chain,
			// and genesisConfigFor finds the genesis its config has to carry —
			// the same three answers a node that declared the name gets.
			w.state.Nodes[i].Binary = f.Binary
		} else {
			w.state.Nodes[i].Role = string(node.RoleBP)
		}
		w.state.Nodes[i].Args = nil

		np, perr := w.pluginFor(w.state.Nodes[i])
		if perr != nil {
			return perr
		}
		prov, cerr := w.writeNodeConfig(ctx, np, preset, placed, peering, pubkey, w.state.Nodes[i], "cross-"+f.Name)
		if cerr != nil {
			return fmt.Errorf("chainsetup: cross-fork: %s: %w", ns.NodeLabel(), cerr)
		}
		w.addConfigProvenance(prov)

		staticNodes, serr := node.PeerList(placed, peering, ns.NodeLabel(), pubkey)
		if serr != nil {
			return fmt.Errorf("chainsetup: cross-fork: %s peers: %w", ns.NodeLabel(), serr)
		}
		spec := process.SpecOf(w.state.Nodes[i])
		spec.Binary = w.binaryFor(w.state.Nodes[i], bin)
		args, aerr := nodeconfig.Argv(process.NodeConfig(np, preset, spec, w.keysBase(), staticNodes))
		if aerr != nil {
			return fmt.Errorf("chainsetup: cross-fork: %s: %w", ns.NodeLabel(), aerr)
		}
		spec.Args = args
		w.state.Nodes[i].Args = args

		h, lerr := process.LaunchAndRecord(ctx, t.Driver, w.ledger, spec)
		if lerr != nil {
			return fmt.Errorf("chainsetup: cross-fork: launch %s: %w", ns.NodeLabel(), lerr)
		}
		w.state.Nodes[i].PID = h.PID
	}
	return nil
}

// confirmBeforeFork holds a restart to the thing that makes it a restart: the
// nodes were on the post-fork build BEFORE the chain reached the fork.
//
// Swapping an executable takes time, and an ordinary fork does not wait — the
// chain goes on sealing while the nodes come down and back up. If it passes the
// fork block in the middle of that, some blocks were crossed by the old build
// and the network is not the one the case describes. It can still look fine
// afterwards, which is why this is checked rather than assumed.
//
// The remedy is not here: it is to schedule the fork far enough out that the
// swap finishes first. So the refusal says where the chain got to and where the
// fork is, which is what the case author has to change.
func (w *Workspace) confirmBeforeFork(ctx context.Context, f GenesisFork) error {
	if !f.Restart {
		return nil
	}
	obs, ok := w.forkObserver(f)
	if !ok {
		return ofKind(errCrossForkNobodyCameBack,
			fmt.Errorf("chainsetup: cross-fork: no node came back from the %q restart", f.Name))
	}
	url, err := w.nodeHTTPURL(obs)
	if err != nil {
		return fmt.Errorf("chainsetup: cross-fork: %w", err)
	}
	head, err := rpc.Dial(url).BlockNumber(ctx)
	if err != nil {
		// Still coming up. The readiness gate is the one that judges that, and
		// a node mid-boot is not evidence the fork was missed.
		return nil //nolint:nilerr // readiness is the gate's question, not this one's
	}
	if at := int64(head); at >= f.At { //nolint:gosec // a chain head, compared against a declared block
		return ofKind(errCrossForkAlreadyPast,
			fmt.Errorf("chainsetup: cross-fork: the chain reached block %d while the nodes were being relaunched, and the %q fork is at %d — the fork has to be far enough out that every node is on the post-fork build before the chain gets there", at, f.Name, f.At))
	}
	return nil
}

// CrossForkFailure is which of crossing a fork's states a failure is.
//
// The four of its own say which moment it stopped in. The borrowed ones are the
// work it shares with the composition: it renders each node's config again and
// launches it, and those fail the way the config and launch stages fail.
func CrossForkFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errCrossForkNoFork):
		return lifecycle.ChainOpCrossForkFailNoFork
	case errors.Is(err, errCrossForkHeadUnreadable):
		return lifecycle.ChainOpCrossForkFailHeadUnreadable
	case errors.Is(err, errCrossForkAlreadyPast):
		return lifecycle.ChainOpCrossForkFailAlreadyPast
	case errors.Is(err, errCrossForkNobodyCameBack):
		return lifecycle.ChainOpCrossForkFailNobodyCameBack
	case errors.Is(err, errLaunchNoBinary):
		return lifecycle.ChainLaunchNodesFailNoBinary
	}
	return ConfigFailure(err)
}
