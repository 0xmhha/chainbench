package testengine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// failureLogHeadLines and failureLogTailLines are how much of each node's log a
// failure keeps, from each end.
//
// Both ends, because the two failures this evidence explains live at opposite
// ones. A node that refuses its genesis or cannot bind a port says so in its
// first lines and exits; a node that dies after an hour says so in its last.
// The tail alone used to be kept, and at one block per second a geth-family
// node writes several lines a second — so 200 lines was the last half minute,
// which is exactly the window a startup failure is not in.
const (
	failureLogHeadLines = 200
	failureLogTailLines = 200
)

// failureDir is where evidence lands when a run fails before it has a session.
const failureDir = "failures"

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
			excerpt, lerr := ws.LogExcerpt(ctx, n.Index, failureLogHeadLines, failureLogTailLines)
			if lerr != nil {
				note("%s: %v", name, lerr)
				continue
			}
			add(name, []byte(excerpt))
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

// saveFailureData writes the evidence into the workspace, for a failure that
// happened before any test record existed to hold it.
//
// It goes under the workspace rather than a session because there is no session
// yet: the engine creates that, and this is the failure that stops the engine
// from starting. The directory is stamped so two failed attempts on one
// workspace do not overwrite each other.
//
// Secrets are scrubbed the same way a record scrubs an observation — the same
// function, so the two paths cannot drift on what counts as a secret.
func saveFailureData(clock func() time.Time, dataDir string, ev []evidence) (string, error) {
	if len(ev) == 0 {
		return "", nil
	}
	now := time.Now().UTC()
	if clock != nil {
		now = clock().UTC()
	}
	dir := filepath.Join(dataDir, failureDir, now.Format("20060102-150405"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("testengine: save failure data: %w", err)
	}
	for _, e := range ev {
		p := filepath.Join(dir, e.Name)
		if err := session.WriteFileAtomic(p, session.Scrub(e.Data), 0o644); err != nil {
			return dir, fmt.Errorf("testengine: save failure data: %s: %w", e.Name, err)
		}
	}
	return dir, nil
}
