package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/machine"

	netmap "github.com/0xmhha/chainbench/internal/core/netmap"
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
func (w *Workspace) recordRun(ctx context.Context, t *machine.Access, bin string) (string, error) {
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
	// The genesis this run composed, read back from the target through the
	// same boundary that wrote it.
	layout := netmap.Layout{Root: w.state.Target.DataRoot}
	if g, err := t.Files.Read(ctx, layout.GenesisPath()); err == nil {
		if err := files.Write(ctx, filepath.Join(dir, "genesis.json"), g, 0o644); err != nil {
			return "", fmt.Errorf("chainsetup: record: %w", err)
		}
	}
	return dir, nil
}
