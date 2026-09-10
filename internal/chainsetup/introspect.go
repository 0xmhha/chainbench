package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Recovering a running chain from the target.
//
// reuse-if-matching compares what a run would compose against what is already
// up. When this workspace ran the network itself, the baseline is its own
// record. When it did not — a first run against a target something is already
// running on — there is no record, so the baseline is read back from the live
// processes: each node's argv gives its --config, and that file's bytes on the
// machine hash to the same value the compose steps record. The comparison then
// runs on equal terms.

// nodeAddr identifies a node the way the target does: which server it runs on
// and the datadir it runs out of.
//
// The label alone does not identify anything. Every composition names its nodes
// node1..nodeN, and the datadir's last element is that label, so keying by it
// merges the node1 of every server AND the node1 of every composition sharing a
// server — the composition id in the path is exactly what distinguishes them.
// Whichever process was found last used to win, which is how a foreign pid could
// be adopted and later stopped as if it were ours.
type nodeAddr struct {
	// Server is the server-set name the process was found on; empty is the
	// target itself (a local composition).
	Server string
	// DataDir is the full --datadir, not its last element.
	DataDir string
}

// runningNode is what introspection recovered about one node already up on the
// target. It keeps where it was found, so a caller can check that a candidate is
// the node it asked about rather than one that merely shares a name.
type runningNode struct {
	Addr       nodeAddr
	ConfigPath string
	PID        int
	Binary     string
	ConfigHash string
}

// introspectRunning finds every process of binaryName on the target and, per
// node label (the last element of its --datadir), recovers the config it runs
// with: it reads the process's argv, parses out the --config path, and hashes
// that file's bytes ON the machine — the same hash the compose steps record. A
// target whose driver cannot read cmdlines, or a process whose config cannot be
// read, contributes nothing rather than failing the run: the affected node then
// composes fresh, which is the safe default.
func (w *Workspace) introspectRunning(ctx context.Context, binaryName string) (map[nodeAddr]runningNode, error) {
	found := map[nodeAddr]runningNode{}
	if binaryName == "" {
		return found, nil
	}
	err := w.eachMachine(func(t *resource.Access, _ []node.Record) error {
		insp, ok := t.Driver.(process.ProcessInspector)
		if !ok {
			return nil
		}
		cmd, ok := t.Driver.(process.CmdlineInspector)
		if !ok {
			return nil
		}
		pids, err := insp.FindBinary(ctx, binaryName)
		if err != nil {
			return err
		}
		for _, pid := range pids {
			argv, err := cmd.Cmdline(ctx, pid)
			if err != nil {
				continue // exited between listing and reading — skip it
			}
			view := nodeconfig.ParseArgv(argv)
			if view.DataDir == "" || view.ConfigPath == "" {
				continue // not a node this compares by label + config
			}
			cfg, err := t.Files.Read(ctx, view.ConfigPath)
			if err != nil {
				continue // config gone or unreadable — cannot compare it
			}
			addr := nodeAddr{Server: t.Spec.Server, DataDir: view.DataDir}
			// Two live processes out of one datadir is not something to pick a
			// winner from: adopting either would bind this composition to a pid
			// chosen by listing order.
			if prev, dup := found[addr]; dup {
				return fmt.Errorf(
					"chainsetup: reuse: two processes are running out of %s on %s (pids %d and %d) — stop one before reusing this composition",
					view.DataDir, serverLabel(t.Spec.Server), prev.PID, pid)
			}
			found[addr] = runningNode{
				Addr:       addr,
				ConfigPath: view.ConfigPath,
				PID:        pid,
				Binary:     view.Binary,
				ConfigHash: filestore.Hash(cfg),
			}
		}
		return nil
	})
	return found, err
}

// serverLabel names a server for a message; a local target has no name.
func serverLabel(server string) string {
	if server == "" {
		return "this machine"
	}
	return server
}
