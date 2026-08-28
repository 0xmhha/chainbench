package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// Recovery (F1). A chainbench run can die between steps — killed, crashed,
// the machine rebooted. What it leaves is the workspace: which steps are
// done, which nodes were recorded with which pids, and what it was asked to
// compose. Resume reads that record, makes it true again against the
// machine, and continues from the first step that never finished. It adds
// no record of its own.

// NetResumeIn identifies the workspace to resume.
type NetResumeIn struct {
	// DataDir is the workspace directory.
	DataDir string
	// Binary overrides the recorded node binary for the steps that run.
	Binary string
}

// NetResumeOut is what the resume found and did.
type NetResumeOut struct {
	// Reconciled is one line per node: what its record said, what the
	// machine said, and what was done about it.
	Reconciled []string
	// Resumed is the first step that had not finished, or empty when every
	// step had.
	Resumed string
	// Steps are the composition steps that ran, in order, with their detail.
	Steps []string
	// Started are the nodes brought back because they were recorded as
	// started and were not running.
	Started []string
	// Nodes is the network afterwards.
	Nodes NetworkStatusOut
}

// ErrNoRequest refuses to resume a workspace that never recorded what it was
// asked to compose — one composed before requests were recorded, or by hand
// step by step.
var ErrNoRequest = errors.New("chainsetup: resume: the workspace records no request — compose it with `net up`, or finish the steps by hand")

// NetResume recovers a workspace whose run died: reconcile the recorded pids
// with the machine, continue the composition from the first step that never
// finished, bring back the nodes that should be running, and read the
// network back.
func NetResume(ctx context.Context, d Deps, in NetResumeIn) (NetResumeOut, error) {
	if in.DataDir == "" {
		return NetResumeOut{}, ErrNoDataDir
	}
	var out NetResumeOut
	var req *NetUpIn
	var first string
	// withWorkspace takes over a stale lock and refuses a live one, which is
	// exactly resume's rule: a run that is still going is not resumed.
	_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		lines, err := ws.Reconcile(ctx)
		out.Reconciled = lines
		if err != nil {
			return "", err
		}
		req = ws.State().Request
		first = ws.firstUndone()
		return "", nil
	})
	if err != nil {
		return out, err
	}
	if req == nil {
		return out, ErrNoRequest
	}
	up := *req
	up.DataDir = in.DataDir
	if in.Binary != "" {
		up.Binary = in.Binary
	}

	if first != "" {
		out.Resumed = first
		res, err := netUpFrom(ctx, d, up, first)
		out.Steps = res.Steps
		if err != nil {
			return out, err
		}
	}
	stage := up.Stage
	if stage == "" {
		stage = UpStart
	}
	if stage == UpStart {
		started, err := startMissing(ctx, d, in.DataDir, up.Binary)
		out.Started = started
		if err != nil {
			return out, err
		}
	}
	nodes, err := NetworkStatus(ctx, d, NetworkStatusIn{DataDir: in.DataDir})
	if err != nil {
		return out, err
	}
	out.Nodes = nodes
	return out, nil
}

// firstUndone is the first composition step the workspace has not recorded
// as done, or empty when every step has.
func (w *Workspace) firstUndone() string {
	stage := UpStart
	if w.state.Request != nil && w.state.Request.Stage != "" {
		stage = w.state.Request.Stage
	}
	for _, name := range upStepNames {
		if stage == UpProvision && (name == "init" || name == "start") {
			return ""
		}
		if !w.state.Steps[name].Done {
			return name
		}
	}
	return ""
}

// Reconcile makes the node records true against the machine. A recorded pid
// that is gone is cleared; a node with no pid whose process is nevertheless
// running — launched by a run that died before it could record — is adopted
// when its command line is the one this workspace would have launched it
// with. It reports one line per node and changes nothing else.
func (w *Workspace) Reconcile(ctx context.Context) ([]string, error) {
	lines := make([]string, 0, len(w.state.Nodes))
	for i, rec := range w.state.Nodes {
		t, err := w.machineFor(rec)
		if err != nil {
			return lines, err
		}
		insp, ok := t.Driver.(driver.ProcessInspector)
		if !ok {
			lines = append(lines, fmt.Sprintf("node%d: pid %d (machine cannot be asked; left as recorded)", rec.Index, rec.PID))
			continue
		}
		if rec.PID > 0 {
			alive, err := insp.PIDAlive(ctx, rec.PID)
			if err != nil {
				return lines, fmt.Errorf("chainsetup: reconcile node%d: %w", rec.Index, err)
			}
			if alive {
				lines = append(lines, fmt.Sprintf("node%d: pid %d alive", rec.Index, rec.PID))
				continue
			}
			w.clearPID(i)
			lines = append(lines, fmt.Sprintf("node%d: pid %d dead, cleared", rec.Index, rec.PID))
			continue
		}
		pid, err := w.orphanOf(ctx, t, rec)
		if err != nil {
			return lines, err
		}
		if pid == 0 {
			lines = append(lines, fmt.Sprintf("node%d: not running", rec.Index))
			continue
		}
		if err := w.recordLaunch(i, pid, w.state.Binary); err != nil {
			return lines, fmt.Errorf("chainsetup: reconcile node%d: %w", rec.Index, err)
		}
		lines = append(lines, fmt.Sprintf("node%d: pid %d running unrecorded, adopted", rec.Index, pid))
	}
	return lines, nil
}

// orphanOf finds a process of this workspace's binary that nobody recorded
// and whose command line is the one rec would launch with. It answers the
// pid, or 0 when there is none — a process running the same binary with
// another command line belongs to somebody else.
func (w *Workspace) orphanOf(ctx context.Context, t *machine.Access, rec node.Record) (int, error) {
	if w.state.Binary == "" || len(rec.Args) == 0 {
		return 0, nil
	}
	insp, ok := t.Driver.(driver.ProcessInspector)
	if !ok {
		return 0, nil
	}
	cmdr, ok := t.Driver.(driver.Commander)
	if !ok {
		return 0, nil
	}
	pids, err := insp.FindBinary(ctx, filepath.Base(w.state.Binary))
	if err != nil {
		return 0, fmt.Errorf("chainsetup: reconcile: %w", err)
	}
	known := map[int]bool{}
	for _, p := range w.ledger.Recorded() {
		known[p.PID] = true
	}
	want := launchCommand(w.state.Binary, rec.Args)
	for _, pid := range pids {
		if known[pid] {
			continue
		}
		out, err := cmdr.Run(ctx, fmt.Sprintf("ps -o command= -p %d", pid))
		if err != nil {
			continue
		}
		if sameCommand(strings.TrimSpace(out), want) {
			return pid, nil
		}
	}
	return 0, nil
}

// launchCommand renders the command line a node is launched with — the same
// rendering the ledger records.
func launchCommand(binary string, args []string) string {
	return strings.Join(append([]string{binary}, args...), " ")
}

// sameCommand compares two command lines by their fields, so the shell's
// spacing does not decide whether a process is ours. The binary is compared
// by its base name: ps reports the path the process was started by, which
// may be the resolved one.
func sameCommand(got, want string) bool {
	g, w := strings.Fields(got), strings.Fields(want)
	if len(g) != len(w) || len(g) == 0 {
		return false
	}
	if filepath.Base(g[0]) != filepath.Base(w[0]) {
		return false
	}
	for i := 1; i < len(g); i++ {
		if g[i] != w[i] {
			return false
		}
	}
	return true
}

// startMissing brings back every node the workspace records as composed but
// not running, with the argv it was armed with.
func startMissing(ctx context.Context, d Deps, dataDir, binary string) ([]string, error) {
	var started []string
	_, err := withWorkspace(d, dataDir, func(ws *Workspace) (string, error) {
		for _, rec := range ws.State().Nodes {
			if rec.PID > 0 {
				continue
			}
			if len(rec.Args) == 0 {
				// Never armed: the start step is what arms it, and that step
				// ran (or will run) through the composition, not here.
				continue
			}
			bin := binary
			if bin == "" {
				bin = ws.State().Binary
			}
			if bin != "" {
				ws.state.Binary = bin
			}
			detail, err := ws.StartNode(ctx, rec.Index)
			if err != nil {
				return "", err
			}
			started = append(started, detail)
		}
		return "", nil
	})
	return started, err
}
