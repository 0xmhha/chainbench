package testengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// evidence is one gathered file: what to call it and what is in it.
type evidence struct {
	Name string
	Data []byte
}

// gatherFailureData collects what can be said about a network that just failed:
// an RPC/block/peer snapshot (health.Run), the process ledger (pid + command per
// node), and each node's log.
//
// It returns the evidence rather than writing it, because the two callers put it
// in different places. A test that failed has a record to hang observations on;
// a run that failed BEFORE any test — the readiness gate refusing a node — has
// no record and no session at all, and used to leave nothing behind. That was
// the whole reason an intermittent startup failure could not be diagnosed: the
// run reported "1 node still not ready" and threw away the one thing that could
// say which node, and why.
//
// Every piece is independent. One that cannot be gathered does not stop the
// others, and it does not vanish either: what failed is recorded as evidence of
// its own, so a missing node log reads as a missing node log rather than as a
// node that was never there.
func gatherFailureData(ctx context.Context, sd chainsetup.Deps, dataDir string, nodes *node.NodeSet) []evidence {
	var out []evidence
	add := func(name string, b []byte) { out = append(out, evidence{Name: name, Data: b}) }
	var problems []string
	note := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if nodes != nil && len(nodes.Nodes) > 0 {
		rep, err := health.Run(ctx, *nodes, health.Options{}, nil)
		switch {
		case err != nil:
			note("health: %v", err)
		default:
			if b, merr := json.MarshalIndent(rep, "", "  "); merr == nil {
				add("health.json", b)
			} else {
				note("health: encode: %v", merr)
			}
		}
	} else {
		note("health: the run has no node table")
	}

	if led, err := process.OpenLedger(dataDir); err != nil {
		note("processes: %v", err)
	} else if b, merr := json.MarshalIndent(led.Recorded(), "", "  "); merr != nil {
		note("processes: encode: %v", merr)
	} else {
		add("processes.json", b)
	}

	// Each node's log is read from ITS machine, not this one: a network spread
	// across a server set (or docker) keeps every log on its own host, so a
	// local read would collect nothing for a remote run. w.Logs reads through the
	// node's machine and elevates through sudo where a root-owned log needs it.
	ws, err := chainsetup.Open(dataDir, sd.Clock)
	if err != nil {
		note("logs: open workspace: %v", err)
	} else {
		ws.SetEnv(sd.Env)
		ws.SetDriver(sd.Driver)
		for _, n := range ws.State().Nodes {
			name := "node" + strconv.Itoa(n.Index) + ".log"
			if n.LogPath == "" {
				note("%s: the node table records no log path", name)
				continue
			}
			// The whole log, as the node wrote it. Only its two ends used to be
			// kept, and a failure whose cause is in the middle — a node that
			// fell out of sync two minutes in, a round change that kept
			// repeating — left no trace of it; nor could two nodes be lined up
			// by time, because each was cut in a different place. The whole
			// file is read either way, so keeping it costs only the space.
			full, lerr := ws.Logs(ctx, n.Index, 0)
			if lerr != nil {
				note("%s: %v", name, lerr)
				continue
			}
			add(name, []byte(full+"\n"))
		}
	}

	if len(problems) > 0 {
		add("gather-problems.txt", []byte(strings.Join(problems, "\n")+"\n"))
	}
	return out
}

// collectFailureData hangs the evidence on a failed test's observations/.
func collectFailureData(ctx context.Context, sd chainsetup.Deps, dataDir string, nodes *node.NodeSet, rec session.TestRecord) {
	for _, e := range gatherFailureData(ctx, sd, dataDir, nodes) {
		rec.Observation(e.Name, e.Data)
	}
}
