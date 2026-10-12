package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// stopOwned stops ns's recorded process at pid, but only once the machine
// shows that pid running the launch this workspace recorded: the binary it
// started and the exact argv. A recorded pid is not proof of whose process it
// is; one that was reused by something else is refused rather than signalled.
// It reports false when the pid is already gone, so there was nothing to stop.
//
// A driver that cannot read a process table (a test's or an injected
// transport) is trusted as before; the real local and SSH drivers can.
func (w *Workspace) stopOwned(ctx context.Context, t *resource.Access, ns node.Record, pid int) (bool, error) {
	if o := process.ProcessTableOf(ctx, t.Driver); o != nil {
		switch obs := process.ObserveLaunch(ctx, o, pid, w.launchBinary(ns), ns.Args); obs.State {
		case process.OwnedRunning:
		case process.OwnedMissing:
			return false, nil
		default:
			return false, fmt.Errorf("chainsetup: refusing to signal node%d's recorded pid %d: it is not the recorded launch (%s)", ns.Index, pid, obs.Reason)
		}
	}
	if err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: pid}); err != nil {
		return false, err
	}
	return true, nil
}

// launchBinary is the executable ns was launched with: the record's binary
// for it, placed the way a launch places it. (The ledger keeps only the
// binary's name, which cannot tell one build from another.)
func (w *Workspace) launchBinary(ns node.Record) string {
	bin, err := w.binary("")
	if err != nil {
		bin = w.state.Binary
	}
	return w.binaryFor(ns, bin)
}
