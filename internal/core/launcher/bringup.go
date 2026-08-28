package launcher

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// defaultLeaderWindow is how long the leader gate may take when the caller did
// not ask for a cluster-size-derived window.
const defaultLeaderWindow = 60 * time.Second

// Result is what a launch produced: the node set and the processes to
// track (pid + datadir + host) for verified teardown.
type Result struct {
	Nodes node.NodeSet
	Procs []process.Proc
}

// Deps injects the launcher's collaborators, so its orchestration is testable
// without real chain binaries. A production wiring supplies a driver-backed
// launcher and an RPC/etcd health gate.
type Deps struct {
	// Launch starts the plan's nodes and returns the node set plus the
	// processes to track. nodes lists the 1-based indices to start; nil or
	// empty means the whole plan, so a caller with nothing to phase passes nil
	// and keeps the behaviour it had.
	Launch func(ctx context.Context, plan driver.Plan, nodes []int) (Result, error)
	// Action performs one named bring-up action against a node, between two
	// phases. What a name means is chain-specific; the launcher owns when it
	// runs and how a failure is classified. Naming an action in a phase without
	// wiring this is an error — the same contract as LeaderGate and
	// SwapBinary, because a bootstrap that is quietly skipped produces a
	// network that starts and then does nothing.
	Action func(ctx context.Context, name string, plan driver.Plan, on node.Node) error
	// HealthGate blocks until the network is healthy, or returns a classified
	// non-OK Diagnosis (etcd leader, block production, fork crossing).
	HealthGate func(ctx context.Context, ns node.NodeSet) (Diagnosis, error)
	// LeaderGate waits for the producer's etcd to have a ready leader, within
	// the given window. It runs before HealthGate when Options.LeaderGate is
	// set. What "leader ready" means is chain-specific (go-wemix's embedded
	// etcd), so the check is injected; the launcher owns when it runs, how
	// long it may take, and how a failure is classified. Leaving it nil while
	// asking for the gate is an error, not a silent pass.
	LeaderGate func(ctx context.Context, ns node.NodeSet, window time.Duration) (Diagnosis, error)
	// SwapBinary performs one scheduled fork swap: it must relaunch the named
	// node on the new binary before the chain reaches the fork block. Like the
	// leader gate this is injected, because "relaunch on a different binary" is
	// a launcher concern; the launcher owns the schedule and the diagnosis.
	// Declaring Options.ForkSwaps without wiring this is an error.
	SwapBinary func(ctx context.Context, ns node.NodeSet, swap ForkSwap) error
	// Procman tracks launched processes for verified teardown.
	Procman *process.Manager
	// Sleep is the backoff sleeper, injected so tests do not wait. Defaults to
	// time.Sleep.
	Sleep func(time.Duration)
}

// impl is the concrete Launcher.
type impl struct {
	// teardownHook observes teardowns in tests. Production leaves it nil: the
	// real evidence is that no process survives, which a unit test cannot see.
	teardownHook func(node.NodeSet)
	deps         Deps
}

// New returns a Launcher over deps.
func New(deps Deps) Launcher {
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}
	if deps.Procman == nil {
		deps.Procman = process.New()
	}
	return &impl{deps: deps}
}

// BringUp launches the plan and gates on health, retrying with backoff. On
// success it returns the node set and an OK diagnosis; on exhaustion it tears
// down and returns the last classified diagnosis and an error.
func (s *impl) BringUp(ctx context.Context, plan driver.Plan, opts Options) (node.NodeSet, Diagnosis, error) {
	attempts := opts.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var last Diagnosis
	for attempt := 1; attempt <= attempts; attempt++ {
		res, err := s.launchPhases(ctx, plan, opts)
		if err != nil {
			// A launch failure is classified from its own text: reporting every
			// one as RPCUnready hid the etcd problems that cause most of them.
			last = Diagnosis{Mode: Classify(err), Detail: err.Error()}
			// Whatever did start is torn down. A launch that fails part way
			// through still leaves processes — with phases it always does, since
			// the group before the failing one is already up — and leaving them
			// holds the ports the next attempt needs, which then fails for a
			// reason that has nothing to do with the first.
			for _, pr := range res.Procs {
				s.deps.Procman.TrackProc(pr)
			}
			if len(res.Nodes.Nodes) > 0 {
				_ = s.Teardown(ctx, res.Nodes, TeardownOpts{RemoveDataDir: true, Grace: graceOr(opts)})
			}
		} else {
			for _, p := range res.Procs {
				s.deps.Procman.TrackProc(p)
			}
			diag, gateErr := s.gate(ctx, res.Nodes, opts)
			if gateErr == nil && diag.OK {
				return res.Nodes, diag, nil
			}
			if gateErr != nil {
				diag.Detail = gateErr.Error()
				if diag.Mode == UnknownFailure {
					diag.Mode = Classify(gateErr)
				}
			}
			last = diag
			// Stop and clean before retrying so ports/datadirs are free. Removing
			// the datadir is what clears a stale etcd cluster state, which is the
			// failure that otherwise repeats on every attempt (design S2).
			_ = s.Teardown(ctx, res.Nodes, TeardownOpts{RemoveDataDir: true, Grace: graceOr(opts)})
		}

		if attempt < attempts && opts.Backoff != nil {
			s.deps.Sleep(opts.Backoff(attempt))
		}
	}
	return node.NodeSet{}, last, fmt.Errorf("launcher: bring-up failed after %d attempt(s): %s: %s", attempts, last.Mode, last.Detail)
}

// gate runs the requested gates in order: the etcd leader gate first (a node
// cannot be healthy before its cluster has a leader), then the general health
// gate. With AlignJoinGap the leader gate's window is derived from the cluster
// size, so a join that is merely waiting for its slot is not called a failure.
func (s *impl) gate(ctx context.Context, ns node.NodeSet, opts Options) (Diagnosis, error) {
	if opts.LeaderGate {
		if s.deps.LeaderGate == nil {
			return Diagnosis{Mode: EtcdJoinFailed},
				fmt.Errorf("launcher: LeaderGate was requested but no leader gate is wired")
		}
		window := defaultLeaderWindow
		if opts.AlignJoinGap {
			window = JoinWindow(len(ns.Nodes))
		}
		diag, err := s.deps.LeaderGate(ctx, ns, window)
		if err != nil || !diag.OK {
			if err != nil && diag.Mode == UnknownFailure {
				diag.Mode = Classify(err)
			}
			if diag.Mode == UnknownFailure {
				diag.Mode = EtcdJoinFailed
			}
			return diag, err
		}
	}
	if s.deps.HealthGate != nil {
		if diag, err := s.deps.HealthGate(ctx, ns); err != nil || !diag.OK {
			return diag, err
		}
	}
	return s.applyForkSwaps(ctx, ns, opts)
}

// applyForkSwaps performs each declared type-2 binary swap. A declared swap with
// no implementation wired is a failure: silently skipping it would let the chain
// cross the fork on the wrong binary and fail later, somewhere less obvious.
func (s *impl) applyForkSwaps(ctx context.Context, ns node.NodeSet, opts Options) (Diagnosis, error) {
	if len(opts.ForkSwaps) == 0 {
		return Diagnosis{OK: true}, nil
	}
	if s.deps.SwapBinary == nil {
		return Diagnosis{Mode: ForkNotCrossed},
			fmt.Errorf("launcher: %d fork swap(s) declared but no swap implementation is wired", len(opts.ForkSwaps))
	}
	for _, swap := range opts.ForkSwaps {
		if err := s.deps.SwapBinary(ctx, ns, swap); err != nil {
			return Diagnosis{Mode: ForkNotCrossed},
				fmt.Errorf("launcher: fork swap %s -> %s before %s (block %d): %w",
					swap.Node, swap.ToBinary, swap.Fork, swap.AtBlock, err)
		}
	}
	return Diagnosis{OK: true}, nil
}

// Teardown stops the tracked processes (verifying local ones are gone) and, when
// requested, removes their data directories.
func (s *impl) Teardown(_ context.Context, ns node.NodeSet, opts TeardownOpts) error {
	if s.teardownHook != nil {
		s.teardownHook(ns)
	}
	// Track the node set's local pids too, so a standalone Teardown still acts.
	for _, n := range ns.Nodes {
		s.deps.Procman.TrackProc(process.Proc{PID: n.PID, Label: "node" + strconv.Itoa(n.Index)})
	}

	var errs []error
	if leaks := s.deps.Procman.StopAll(opts.Grace); len(leaks) > 0 {
		errs = append(errs, fmt.Errorf("launcher: orphan pids after stop: %v", leaks))
	}
	if opts.RemoveDataDir {
		errs = append(errs, s.deps.Procman.RemoveDataDirs()...)
	}
	return errors.Join(errs...)
}

// graceOr returns a sane teardown grace for retries.
func graceOr(_ Options) time.Duration { return 5 * time.Second }

// launchPhases starts the plan in the order the family declared, running the
// actions between groups. With no phases — or one phase that names no nodes —
// it is the single launch it replaced.
//
// Each phase's nodes join the accumulated set, so a later phase's health is
// judged against the whole network rather than against itself.
func (s *impl) launchPhases(ctx context.Context, plan driver.Plan, opts Options) (Result, error) {
	phases := opts.Phases
	if len(phases) == 0 {
		return s.deps.Launch(ctx, plan, nil)
	}
	var all Result
	for _, phase := range phases {
		res, err := s.deps.Launch(ctx, plan, phase.Nodes)
		if err != nil {
			return all, fmt.Errorf("launcher: phase %q: %w", phase.Name, err)
		}
		all.Nodes.Chain = res.Nodes.Chain
		all.Nodes.Network = res.Nodes.Network
		all.Nodes.Nodes = append(all.Nodes.Nodes, res.Nodes.Nodes...)
		all.Procs = append(all.Procs, res.Procs...)

		for _, name := range phase.Actions {
			if s.deps.Action == nil {
				return all, fmt.Errorf("launcher: phase %q needs action %q but no action executor is wired", phase.Name, name)
			}
			on, ok := actionNode(phase, res.Nodes, all.Nodes)
			if !ok {
				return all, fmt.Errorf("launcher: phase %q needs action %q but launched no node to run it on", phase.Name, name)
			}
			if err := s.deps.Action(ctx, name, plan, on); err != nil {
				return all, fmt.Errorf("launcher: phase %q action %q: %w", phase.Name, name, err)
			}
		}
	}
	return all, nil
}

// actionNode is the node a phase's actions run against. A phase that names one
// gets that node, looked up among everything launched so far — the phase whose
// actions concern the boot node runs after the boot node's own phase, so it is
// there. Otherwise it is the first node this phase launched, which for a
// bootstrap phase is the producer that is alone.
func actionNode(phase registry.Phase, launched, sofar node.NodeSet) (node.Node, bool) {
	if phase.ActionsOn > 0 {
		for _, n := range sofar.Nodes {
			if n.Index == phase.ActionsOn {
				return n, true
			}
		}
		return node.Node{}, false
	}
	if len(launched.Nodes) == 0 {
		return node.Node{}, false
	}
	return launched.Nodes[0], true
}
