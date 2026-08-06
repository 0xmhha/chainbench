package supervisor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/procman"
)

// LaunchResult is what a launch produced: the node set and the processes to
// track (pid + datadir + host) for verified teardown.
type LaunchResult struct {
	Nodes node.NodeSet
	Procs []procman.Proc
}

// Deps injects the supervisor's collaborators, so its orchestration is testable
// without real chain binaries. A production wiring supplies a driver-backed
// launcher and an RPC/etcd health gate.
type Deps struct {
	// Launch starts the plan's nodes and returns the node set plus the
	// processes to track.
	Launch func(ctx context.Context, plan setup.Plan) (LaunchResult, error)
	// HealthGate blocks until the network is healthy, or returns a classified
	// non-OK Diagnosis (etcd leader, block production, fork crossing).
	HealthGate func(ctx context.Context, ns node.NodeSet) (Diagnosis, error)
	// Procman tracks launched processes for verified teardown.
	Procman *procman.Manager
	// Sleep is the backoff sleeper, injected so tests do not wait. Defaults to
	// time.Sleep.
	Sleep func(time.Duration)
}

// sup is the concrete Supervisor.
type sup struct {
	deps Deps
}

// New returns a Supervisor over deps.
func New(deps Deps) Supervisor {
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}
	if deps.Procman == nil {
		deps.Procman = procman.New()
	}
	return &sup{deps: deps}
}

// BringUp launches the plan and gates on health, retrying with backoff. On
// success it returns the node set and an OK diagnosis; on exhaustion it tears
// down and returns the last classified diagnosis and an error.
func (s *sup) BringUp(ctx context.Context, plan setup.Plan, opts Options) (node.NodeSet, Diagnosis, error) {
	attempts := opts.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var last Diagnosis
	for attempt := 1; attempt <= attempts; attempt++ {
		res, err := s.deps.Launch(ctx, plan)
		if err != nil {
			last = Diagnosis{Mode: RPCUnready, Detail: err.Error()}
		} else {
			for _, p := range res.Procs {
				s.deps.Procman.TrackProc(p)
			}
			diag, gateErr := s.deps.HealthGate(ctx, res.Nodes)
			if gateErr == nil && diag.OK {
				return res.Nodes, diag, nil
			}
			if gateErr != nil {
				diag.Detail = gateErr.Error()
			}
			last = diag
			// Stop and clean before retrying so ports/datadirs are free.
			_ = s.Teardown(ctx, res.Nodes, TeardownOpts{RemoveDataDir: true, Grace: graceOr(opts)})
		}

		if attempt < attempts && opts.Backoff != nil {
			s.deps.Sleep(opts.Backoff(attempt))
		}
	}
	return node.NodeSet{}, last, fmt.Errorf("supervisor: bring-up failed after %d attempt(s): %s: %s", attempts, last.Mode, last.Detail)
}

// Teardown stops the tracked processes (verifying local ones are gone) and, when
// requested, removes their data directories.
func (s *sup) Teardown(_ context.Context, ns node.NodeSet, opts TeardownOpts) error {
	// Track the node set's local pids too, so a standalone Teardown still acts.
	for _, n := range ns.Nodes {
		s.deps.Procman.TrackProc(procman.Proc{PID: n.PID, Label: "node" + strconv.Itoa(n.Index)})
	}

	var errs []error
	if leaks := s.deps.Procman.StopAll(opts.Grace); len(leaks) > 0 {
		errs = append(errs, fmt.Errorf("supervisor: orphan pids after stop: %v", leaks))
	}
	if opts.RemoveDataDir {
		errs = append(errs, s.deps.Procman.RemoveDataDirs()...)
	}
	return errors.Join(errs...)
}

// graceOr returns a sane teardown grace for retries.
func graceOr(_ Options) time.Duration { return 5 * time.Second }
