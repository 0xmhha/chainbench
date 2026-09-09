package testengine

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// failureLogTailLines is how much of each node's log a failed test keeps.
const failureLogTailLines = 200

// collectFailureData gathers a failed test's evidence into its observations/: an
// RPC/block/peer snapshot (health.Run), the process ledger (pid + command per
// node), and the tail of each node's log. It reuses the existing observation
// modules and writes nothing a node reads. Best-effort and independent: a piece
// that cannot be gathered (no network, no workspace, no log) is skipped without
// stopping the others. Observations are scrubbed by the record, so no secret
// from a command or log reaches the evidence.
func collectFailureData(ctx context.Context, sd chainsetup.Deps, dataDir string, nodes *node.NodeSet, rec session.TestRecord) {
	if nodes != nil && len(nodes.Nodes) > 0 {
		if rep, err := health.Run(ctx, *nodes, health.Options{}, nil); err == nil {
			if b, err := json.MarshalIndent(rep, "", "  "); err == nil {
				rec.Observation("health.json", b)
			}
		}
	}
	if led, err := process.OpenLedger(dataDir); err == nil {
		if b, err := json.MarshalIndent(led.Recorded(), "", "  "); err == nil {
			rec.Observation("processes.json", b)
		}
	}
	// Each node's log is read from ITS machine, not this one: a network spread
	// across a server set (or docker) keeps every log on its own host, so a
	// local read would collect nothing for a remote run. w.Logs reads through the
	// node's machine and elevates through sudo where a root-owned log needs it.
	if ws, err := chainsetup.Open(dataDir, sd.Clock); err == nil {
		ws.SetEnv(sd.Env)
		ws.SetDriver(sd.Driver)
		for _, n := range ws.State().Nodes {
			if n.LogPath == "" {
				continue
			}
			if tail, err := ws.Logs(ctx, n.Index, failureLogTailLines); err == nil {
				rec.Observation("node"+strconv.Itoa(n.Index)+".log", []byte(tail))
			}
		}
	}
}
