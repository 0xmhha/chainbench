package chainsetup

import (
	"context"
	"fmt"
	"sort"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Stopping the nodes a run started but never got to record.
//
// stopNodes can only stop what the record names, and the record is written
// when the step that launched finishes. A run killed between the launch and
// that write leaves nodes running with nothing pointing at them: `chain stop`
// answers "0 node(s) stopped" while four are up, and the next composition on
// the same machine is refused for ports that are held by a network nobody can
// name. Measured 2026-09-28 on the common-set sweep, where one interrupted
// case blocked the twenty-seven that followed it.
//
// The pid is gone but the node is not anonymous: it was launched with
// --datadir, and that path is this workspace's. So the machine is asked which
// processes run this composition's binaries, each one's argv is read back, and
// any whose datadir belongs to a node of this workspace is stopped.
//
// Both questions are capabilities a driver may already answer — ProcessInspector
// for "which pids run this binary" and CmdlineInspector for "what argv did this
// pid get" — and reuse reads them ON the machine for the same reason: an
// operator-side guess about a remote process is a guess about the wrong
// resource. A driver that answers neither contributes nothing here rather than
// being guessed at.

// orphan is a node process this workspace started and cannot name.
type orphan struct {
	// Label is the node the datadir belongs to, which is how it is reported.
	Label node.Label
	// Index is the node's position, which the driver's Stop takes.
	Index int
	// PID is what will be signalled.
	PID int
	// Access is the machine it runs on, and the driver that can stop it.
	Access *resource.Access
}

// StopUnrecorded stops the node processes running out of this workspace's
// datadirs that the record holds no pid for, and says how many went down.
//
// It is the second half of Stop, not a separate verb: "stop this composition"
// means every node of it, and a node whose pid never reached the record is
// still one of them.
func (w *Workspace) StopUnrecorded(ctx context.Context) (int, error) {
	found, err := w.unrecordedNodes(ctx)
	if err != nil {
		return 0, err
	}
	stopped := 0
	var failures []string
	for _, o := range found {
		if serr := o.Access.Driver.Stop(ctx, process.Handle{Index: o.Index, PID: o.PID}); serr != nil {
			failures = append(failures, fmt.Sprintf("%s (pid %d): %v", o.Label, o.PID, serr))
			continue
		}
		stopped++
	}
	if len(failures) > 0 {
		return stopped, fmt.Errorf("chainsetup: stop: %d unrecorded node(s) stopped, %d would not: %s",
			stopped, len(failures), joinAnd(failures))
	}
	return stopped, nil
}

// unrecordedNodes is every process running out of one of this workspace's
// datadirs whose pid the record does not hold.
//
// A pid the record DOES hold is left to stopNodes, which knows which node it
// is without reading anything back. Skipping it here also keeps a node from
// being signalled twice, which on a driver that waits for the grace period
// would pay it twice.
func (w *Workspace) unrecordedNodes(ctx context.Context) ([]orphan, error) {
	lay, err := w.layout()
	if err != nil {
		return nil, err
	}
	// The datadir a node of this workspace would have been launched with,
	// against the node it belongs to. A path outside this set belongs to some
	// other composition and is not ours to stop.
	type slot struct {
		label node.Label
		index int
	}
	mine := make(map[string]slot, len(w.state.Nodes))
	recorded := make(map[int]bool, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		mine[lay.DataDir(ns.NodeLabel())] = slot{label: ns.NodeLabel(), index: ns.Index}
		if ns.PID > 0 {
			recorded[ns.PID] = true
		}
	}
	if len(mine) == 0 {
		return nil, nil
	}

	bin, err := w.binary("")
	if err != nil {
		// No binary named yet means nothing was launched from here.
		return nil, nil //nolint:nilerr // an unresolvable binary is "nothing to find", not a failure to stop
	}
	var found []orphan
	err = w.eachMachine(func(t *resource.Access, nodes []node.Record) error {
		insp, ok := t.Driver.(process.ProcessInspector)
		if !ok {
			return nil
		}
		cmd, ok := t.Driver.(process.CmdlineInspector)
		if !ok {
			return nil
		}
		seen := map[int]bool{}
		for _, name := range w.binaryNamesOn(nodes, bin) {
			pids, perr := insp.FindBinary(ctx, name)
			if perr != nil {
				return perr
			}
			for _, pid := range pids {
				if seen[pid] || recorded[pid] {
					continue
				}
				seen[pid] = true
				argv, cerr := cmd.Cmdline(ctx, pid)
				if cerr != nil {
					continue // exited between listing and reading
				}
				at, ours := mine[nodeconfig.ParseArgv(argv).DataDir]
				if !ours {
					continue
				}
				found = append(found, orphan{Label: at.label, Index: at.index, PID: pid, Access: t})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Sorted so a run that stops several reports them the same way twice.
	sort.Slice(found, func(i, j int) bool {
		if found[i].Label != found[j].Label {
			return found[i].Label < found[j].Label
		}
		return found[i].PID < found[j].PID
	})
	return found, nil
}

// joinAnd lists reasons in one sentence.
func joinAnd(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += "; "
		}
		out += s
	}
	return out
}
