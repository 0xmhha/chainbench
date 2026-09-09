package chainsetup

import (
	"context"
	"path"

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

// runningNode is what introspection recovered about one node already up on the
// target, keyed for comparison by the node label its datadir carries.
type runningNode struct {
	Label      string
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
func (w *Workspace) introspectRunning(ctx context.Context, binaryName string) (map[string]runningNode, error) {
	found := map[string]runningNode{}
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
			label := path.Base(view.DataDir)
			found[label] = runningNode{
				Label:      label,
				PID:        pid,
				Binary:     view.Binary,
				ConfigHash: filestore.Hash(cfg),
			}
		}
		return nil
	})
	return found, err
}
