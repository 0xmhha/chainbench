package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Every start leaves a record: which chain was set up, from which inputs,
// into which layout. It is the debugging starting point — "what exactly did
// this run compose?" answered from a folder instead of from memory — kept
// under runs/<stamp>/ inside the workspace, one folder per run.
//
// The record carries NO credentials. The server set's ssh section never
// enters it: the record names the server-set file, and how to log in stays
// where it lives ("the pool says where nodes may run, not how to log in").

// runsDir is where a workspace keeps its run records.
const runsDir = "runs"

// runManifest is the record's index file.
type runManifest struct {
	StartedAt string            `json:"startedAt"`
	Chain     string            `json:"chain"`
	Binary    string            `json:"binary"`
	Peering   string            `json:"peering,omitempty"`
	KeysDir   string            `json:"keysDir"`
	ServerSet string            `json:"serverSet,omitempty"`
	Docker    bool              `json:"docker,omitempty"`
	Target    runTarget         `json:"target"`
	Nodes     []runNode         `json:"nodes"`
	Steps     map[string]string `json:"steps"`
}

// runTarget is where the data plane lived — addressing only, never a login.
type runTarget struct {
	Server   string `json:"server,omitempty"`
	Host     string `json:"host,omitempty"`
	DataRoot string `json:"dataRoot"`
	// Where is the human rendering ("server box1:/data/cb", "local /tmp/n1").
	Where string `json:"where"`
}

// runNode is one node as this run launched it.
type runNode struct {
	Label   string `json:"label"`
	Role    string `json:"role"`
	Host    string `json:"host"`
	P2P     int    `json:"p2p"`
	HTTP    int    `json:"http"`
	PID     int    `json:"pid,omitempty"`
	Command string `json:"command,omitempty"`
}

// recordRun writes this start's record folder: the manifest, the genesis the
// run composed (read back through the same boundary it was written through), and
// each node's launch command. Failures are reported, not fatal — a record
// must never take the network it records down with it.
func (w *Workspace) recordRun(ctx context.Context, t *resource.Access, bin string) (string, error) {
	stamp := w.now().UTC().Format("20060102-150405")
	dir := filepath.Join(w.comp.Dir(), runsDir, stamp)
	files := filestore.Local{}

	m := runManifest{
		StartedAt: w.now().UTC().Format(time.RFC3339),
		Chain:     w.state.Chain,
		Binary:    bin,
		Peering:   w.state.Peering,
		KeysDir:   w.state.KeysDir,
		ServerSet: w.state.ServerSet,
		Docker:    w.state.Docker,
		Target: runTarget{
			Server: w.state.Target.Server,
			Host:   w.state.Target.Host, DataRoot: w.state.Target.DataRoot,
			Where: w.state.Target.Describe(),
		},
		Steps: map[string]string{},
	}
	for step, mark := range w.state.Steps {
		m.Steps[step] = mark.Detail
	}
	var commands []string
	for _, ns := range w.state.Nodes {
		cmd := ""
		if p, ok := w.ledger.Get(string(ns.NodeLabel())); ok {
			cmd = p.Command
		}
		m.Nodes = append(m.Nodes, runNode{
			Label: string(ns.NodeLabel()), Role: ns.Role, Host: nodeHost(ns),
			P2P: ns.P2P, HTTP: ns.HTTP, PID: ns.PID, Command: cmd,
		})
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("chainsetup: record: %w", err)
	}
	if err := files.Write(ctx, filepath.Join(dir, "manifest.json"), b, 0o644); err != nil {
		return "", fmt.Errorf("chainsetup: record: %w", err)
	}
	if len(commands) > 0 {
		body := strings.Join(commands, "\n") + "\n"
		if err := files.Write(ctx, filepath.Join(dir, "launch-commands.txt"), []byte(body), 0o644); err != nil {
			return "", fmt.Errorf("chainsetup: record: %w", err)
		}
	}
	// The inputs this run actually composed, read back from the target through
	// the same boundary that wrote them.
	//
	// The path comes from the state, not from a freshly built Layout: a Layout
	// with only a Root resolves to the flat <dataRoot>/genesis.json, which is
	// not where a composition with a workspace-config puts it. That record
	// silently held no genesis at all — or, worse, a leftover from a different
	// composition sharing the data root, presented as this run's.
	//
	// A missing input is reported rather than skipped. The record exists to say
	// what ran; a record that quietly lacks the genesis looks complete and is
	// not, and finding that out later costs more than the note costs here.
	var missing []string
	paths := w.genesisPaths()
	if len(paths) == 0 {
		missing = append(missing, "genesis (no genesis step has run)")
	}
	// Every document, because a network can run two binaries that do not accept
	// the same one, and a record holding only the first answers for half the
	// nodes. The network's keeps the name a reader expects; the rest are named
	// for the file they came from.
	for i, p := range paths {
		name := "genesis.json"
		if i > 0 {
			name = "genesis-" + filepath.Base(p)
		}
		g, err := t.Files.Read(ctx, p)
		if err != nil {
			missing = append(missing, fmt.Sprintf("genesis (%s): %v", p, err))
			continue
		}
		if werr := files.Write(ctx, filepath.Join(dir, name), g, 0o644); werr != nil {
			return "", fmt.Errorf("chainsetup: record: %w", werr)
		}
	}

	// Each node's config, from the machine that node runs on. A run record
	// without them cannot answer what a node was actually started with, which
	// is the first question asked of it.
	for _, ns := range w.state.Nodes {
		if ns.ConfigPath == "" {
			continue
		}
		nt, err := w.machineFor(ns)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%s config: %v", ns.NodeLabel(), err))
			continue
		}
		cfg, err := nt.Files.Read(ctx, ns.ConfigPath)
		if err != nil {
			missing = append(missing, fmt.Sprintf("%s config (%s): %v", ns.NodeLabel(), ns.ConfigPath, err))
			continue
		}
		name := fmt.Sprintf("config_%s.toml", ns.NodeLabel())
		if err := files.Write(ctx, filepath.Join(dir, name), cfg, 0o644); err != nil {
			return "", fmt.Errorf("chainsetup: record: %w", err)
		}
	}
	if len(missing) > 0 {
		note := "these inputs were not collected:\n  " + strings.Join(missing, "\n  ") + "\n"
		if err := files.Write(ctx, filepath.Join(dir, "missing-inputs.txt"), []byte(note), 0o644); err != nil {
			return "", fmt.Errorf("chainsetup: record: %w", err)
		}
	}
	return dir, nil
}

// FirstUndone is the first composition step the workspace has not recorded
// as done, or empty when every step has.
func (w *Workspace) FirstUndone() string {
	stage := UpStart
	if w.state.Request != nil && w.state.Request.Stage != "" {
		stage = w.state.Request.Stage
	}
	for _, name := range UpStepNames {
		if stage == UpDeploy && (name == "init" || name == "start") {
			return ""
		}
		if !w.state.Steps[name].Done {
			return name
		}
	}
	return ""
}

// Reconcile makes the node records true against the resource. A recorded pid
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
		insp, ok := t.Driver.(process.ProcessInspector)
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
func (w *Workspace) orphanOf(ctx context.Context, t *resource.Access, rec node.Record) (int, error) {
	if w.state.Binary == "" || len(rec.Args) == 0 {
		return 0, nil
	}
	insp, ok := t.Driver.(process.ProcessInspector)
	if !ok {
		return 0, nil
	}
	cmdr, ok := t.Driver.(process.Commander)
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

// RecordRequest writes what the composition was asked for onto the workspace.
//
// The location is not part of it: the record is where the workspace is, so a
// workspace moved to another directory still reads as the request it was
// composed from rather than as one pointing somewhere that no longer exists.
func (w *Workspace) RecordRequest(in ChainUpIn) error {
	req := in
	req.DataDir = ""
	w.state.Request = &req
	return nil
}

// launchCommand is the one spelling of a launch, for comparing what is running
// against what this workspace would run.
func launchCommand(binary string, args []string) string {
	return strings.TrimSpace(binary + " " + strings.Join(args, " "))
}

// sameCommand reports whether two launches are the same one, ignoring the
// difference between a path and the name at the end of it: a node started from
// an absolute path and one started from PATH are the same node.
func sameCommand(got, want string) bool {
	if got == want {
		return true
	}
	gf, wf := strings.Fields(got), strings.Fields(want)
	if len(gf) == 0 || len(wf) == 0 || len(gf) != len(wf) {
		return false
	}
	if filepath.Base(gf[0]) != filepath.Base(wf[0]) {
		return false
	}
	return strings.Join(gf[1:], " ") == strings.Join(wf[1:], " ")
}
