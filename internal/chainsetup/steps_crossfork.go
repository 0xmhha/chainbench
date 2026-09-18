package chainsetup

import (
	"context"
	"fmt"
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
func (w *Workspace) CrossFork(ctx context.Context, opts CrossForkOpts) (string, error) {
	f := w.state.Fork
	if f == nil {
		return "", fmt.Errorf("chainsetup: cross-fork: this network is composed to cross no fork — declare one under env.upgrade")
	}
	successors := w.forkSuccessors(*f)
	if len(successors) == 0 {
		return "", fmt.Errorf("chainsetup: cross-fork: no node runs binary %q, so the %q fork has nobody to hand over to", f.Binary, f.Name)
	}
	if w.alreadyCrossed(successors) {
		detail := fmt.Sprintf("%s already crossed: %d successor(s) already produce", f.Name, len(successors))
		w.markStep("cross-fork", detail)
		return detail, nil
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = crossForkTimeout
	}
	head, err := w.awaitForkBoundary(ctx, *f, timeout)
	if err != nil {
		return "", err
	}
	if err := w.handOver(ctx, *f, successors); err != nil {
		return "", err
	}
	detail := fmt.Sprintf("%s at %d: head %d, %d successor(s) now produce", f.Name, f.At, head, len(successors))
	w.markStep("cross-fork", detail)
	return detail, nil
}

// forkSuccessors is the node table's positions for the build that seals after
// the fork. By binary rather than by role, which is the same question
// binaryFor, genesisFor and pluginFor each answer — before the fork these nodes
// are endpoints, and after it they produce.
func (w *Workspace) forkSuccessors(f GenesisFork) []int {
	var out []int
	for i, ns := range w.state.Nodes {
		if ns.Binary == f.Binary {
			out = append(out, i)
		}
	}
	return out
}

// alreadyCrossed reports whether every successor already produces. A partially
// crossed network is not "already crossed": the remaining ones still have to be
// told, or the set that reaches quorum is smaller than the genesis says.
func (w *Workspace) alreadyCrossed(successors []int) bool {
	for _, i := range successors {
		if !node.Is(node.Role(w.state.Nodes[i].Role), node.RoleBP) {
			return false
		}
	}
	return true
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
		return 0, fmt.Errorf("chainsetup: cross-fork: no node runs the pre-fork build, so there is nothing to read the fork boundary from")
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
				return 0, fmt.Errorf("chainsetup: cross-fork: %s reached no head within %s: %w", obs.NodeLabel(), timeout, lastErr)
			}
			return 0, fmt.Errorf("chainsetup: cross-fork: %s is at block %d after %s, and the %q fork is at %d — the chain never reached the block before it", obs.NodeLabel(), last, timeout, f.Name, f.At)
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
		if ns.Binary != f.Binary && ns.PID > 0 {
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
		// the role instead of repeating the endpoint's command line.
		w.state.Nodes[i].Role = string(node.RoleBP)
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
