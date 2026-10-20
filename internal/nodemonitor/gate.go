package nodemonitor

import (
	"context"
	"fmt"
	"time"
)

// Observer gathers the current facts for every gated node. It composes the
// existing observation modules (process/inspect, health, collector, preflight);
// nodemonitor does not re-observe. Observe is called once per round.
type Observer interface {
	Observe(ctx context.Context) ([]Facts, error)
}

// Restarter restarts one node by its index, reusing the existing restart verb.
// It is the only external mutation the gate performs, and only for a
// RESTARTABLE node within the restart cap.
type Restarter interface {
	Restart(ctx context.Context, node int) error
}

// Clock waits d, or returns ctx.Err() if the context ends first. It is injected
// so tests do not sleep in real time.
type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
}

// EvidenceSink records every verdict and every recovery attempt, so a run's
// artifacts show why the gate let a test start, waited, restarted, or
// terminated. A nil sink is allowed (the gate uses noopSink).
type EvidenceSink interface {
	// Verdict records one node's verdict in a round (1-based).
	Verdict(round int, r NodeReport)
	// Recovery records a restart attempt (1-based) and its error (nil on success).
	Recovery(node, attempt int, err error)
}

// Options tunes the gate. Zero fields take the package defaults.
type Options struct {
	// MaxNodeMonitorTimeout bounds the entire readiness gate, including
	// observation, recovery and waits across rounds. A cluster whose nodes join on a
	// schedule (e.g. go-wemix etcd) should set it at least as large as that
	// join window.
	MaxNodeMonitorTimeout time.Duration
	// WaitInterval is how long to wait between re-observations while nodes are
	// WAITABLE.
	WaitInterval time.Duration
	// MaxRestarts caps how many times each node may be restarted before the gate
	// gives up and terminates.
	MaxRestarts int
}

// Defaults for Options' zero fields.
const (
	DefaultMaxNodeMonitorTimeout = 90 * time.Second
	DefaultWaitInterval          = 3 * time.Second
	DefaultMaxRestarts           = 1
)

func (o Options) withDefaults() Options {
	if o.MaxNodeMonitorTimeout <= 0 {
		o.MaxNodeMonitorTimeout = DefaultMaxNodeMonitorTimeout
	}
	if o.WaitInterval <= 0 {
		o.WaitInterval = DefaultWaitInterval
	}
	// Zero means unset here, as it does for the other two: a caller writing
	// Options{} is asking for defaults. Treating it as "never restart" made
	// that literal contradict the one call site's own comment ("restarting
	// RESTARTABLE ones within limits") and terminate on the first node that
	// needed one.
	if o.MaxRestarts <= 0 {
		o.MaxRestarts = DefaultMaxRestarts
	}
	return o
}

// Result is the gate's outcome. OK is true only when every gated node reached
// READY. Otherwise Terminate names why the caller must tear the network down;
// the gate never applies a destructive remedy itself.
type Result struct {
	OK        bool
	Reports   []NodeReport
	Terminate string
}

// Gate observes, classifies, and applies limited recovery until every node is
// READY or the budget/cap is exhausted. It returns OK with the final reports,
// or a Terminate reason (a FATAL node, a restart cap or wait budget reached, or
// a restart that itself failed). It restarts nodes only through restart and
// never deletes data, rewinds, or swaps genesis.
//
// An error is returned only when observation itself fails; a network that
// cannot be made ready is a Result with OK false and a Terminate reason, not an
// error.
func Gate(ctx context.Context, obs Observer, restart Restarter, clock Clock, sink EvidenceSink, opts Options) (Result, error) {
	opts = opts.withDefaults()
	gateCtx, cancel := context.WithTimeout(ctx, opts.MaxNodeMonitorTimeout)
	defer cancel()
	if sink == nil {
		sink = noopSink{}
	}
	restarts := map[int]int{}
	remainingWait := opts.MaxNodeMonitorTimeout
	var lastReports []NodeReport
	finishError := func(err error) (Result, error) {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		if gateCtx.Err() != nil {
			return Result{Reports: lastReports, Terminate: fmt.Sprintf("node readiness deadline reached after %s", opts.MaxNodeMonitorTimeout)}, nil
		}
		return Result{}, err
	}

	for round := 1; ; round++ {
		if err := gateCtx.Err(); err != nil {
			return finishError(err)
		}
		facts, err := obs.Observe(gateCtx)
		if err != nil {
			return finishError(fmt.Errorf("nodemonitor: observe: %w", err))
		}
		if err := gateCtx.Err(); err != nil {
			return finishError(err)
		}
		reports := make([]NodeReport, len(facts))
		var fatal *NodeReport
		var restartable, waitable []NodeReport
		for i, f := range facts {
			r := Classify(f)
			reports[i] = r
			sink.Verdict(round, r)
			switch r.Verdict {
			case Fatal:
				if fatal == nil {
					fatal = &reports[i]
				}
			case Restartable:
				restartable = append(restartable, r)
			case Waitable:
				waitable = append(waitable, r)
			}
		}
		lastReports = reports

		if fatal != nil {
			return Result{Reports: reports, Terminate: fmt.Sprintf("node%d FATAL: %s", fatal.Node, reasons(*fatal))}, nil
		}
		if len(restartable) == 0 && len(waitable) == 0 {
			return Result{OK: true, Reports: reports}, nil
		}

		// RESTARTABLE first: a restart may also clear what looks WAITABLE later.
		if len(restartable) > 0 {
			for _, r := range restartable {
				if restarts[r.Node] >= opts.MaxRestarts {
					return Result{Reports: reports, Terminate: fmt.Sprintf("node%d exhausted %d restart(s): %s", r.Node, opts.MaxRestarts, reasons(r))}, nil
				}
				restarts[r.Node]++
				rerr := restart.Restart(gateCtx, r.Node)
				sink.Recovery(r.Node, restarts[r.Node], rerr)
				if err := gateCtx.Err(); err != nil {
					return finishError(err)
				}
				if rerr != nil {
					return Result{Reports: reports, Terminate: fmt.Sprintf("node%d restart failed: %v", r.Node, rerr)}, nil
				}
			}
			continue // re-observe immediately after restarting
		}

		// Only WAITABLE remain: wait if the budget allows.
		if remainingWait < opts.WaitInterval {
			return Result{Reports: reports, Terminate: fmt.Sprintf("%d node(s) still not ready after %s", len(waitable), opts.MaxNodeMonitorTimeout)}, nil
		}
		if err := clock.Sleep(gateCtx, opts.WaitInterval); err != nil {
			return finishError(err)
		}
		remainingWait -= opts.WaitInterval
	}
}

// reasons joins a report's reasons for a terminate message.
func reasons(r NodeReport) string {
	if len(r.Reasons) == 0 {
		return r.Verdict.String()
	}
	out := r.Reasons[0]
	for _, s := range r.Reasons[1:] {
		out += "; " + s
	}
	return out
}

// noopSink discards evidence, for callers that pass no sink.
type noopSink struct{}

func (noopSink) Verdict(int, NodeReport)  {}
func (noopSink) Recovery(int, int, error) {}
