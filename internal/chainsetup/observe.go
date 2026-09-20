package chainsetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Observation: what a composed network is doing right now.
//
// Logs are read from the machine the node runs on, through the same access the
// launch used, so a remote node's log needs no second credential. Health asks
// each node's RPC rather than its process table: a process that is alive and
// not answering is not healthy, and only the endpoint knows.

func (w *Workspace) Logs(ctx context.Context, index, n int) (string, error) {
	for _, ns := range w.state.Nodes {
		if ns.Index != index {
			continue
		}
		t, err := w.machineFor(ns)
		if err != nil {
			return "", err
		}
		// A node's log lives on its machine, and may be root-owned there, so it is
		// read through the machine (remote or local) and elevated through sudo
		// where the login user cannot reach it and the server set permits it.
		b, err := t.ReadMaybeElevated(ctx, ns.LogPath)
		if err != nil {
			return "", fmt.Errorf("chainsetup: logs: %w", err)
		}
		lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if n > 0 && len(lines) > n {
			lines = lines[len(lines)-n:]
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("chainsetup: logs: no node %d in the table", index)
}

// LogExcerpt returns the first head lines and the last tail lines of one node's
// log, with a line in between saying how much was left out.
//
// The tail alone is not enough for the failure this exists to explain. A node
// that refuses its genesis, or cannot bind a port, says so in its first few
// lines and then exits; a node that dies after an hour says so in its last. At
// one block per second a geth-family node writes several lines a second, so a
// 200-line tail is the last half minute — which is exactly the window that does
// NOT contain a startup failure.
//
// A log shorter than head+tail is returned whole: eliding nothing is not worth
// a marker saying so.
func (w *Workspace) LogExcerpt(ctx context.Context, index, head, tail int) (string, error) {
	full, err := w.Logs(ctx, index, 0)
	if err != nil {
		return "", err
	}
	return excerpt(full, head, tail), nil
}

// excerpt keeps both ends of s and says what it dropped.
func excerpt(s string, head, tail int) string {
	if head <= 0 && tail <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= head+tail {
		return s
	}
	out := make([]string, 0, head+tail+1)
	out = append(out, lines[:head]...)
	out = append(out, fmt.Sprintf("... %d line(s) elided ...", len(lines)-head-tail))
	out = append(out, lines[len(lines)-tail:]...)
	return strings.Join(out, "\n")
}

// livePIDs asks each node's machine whether its recorded pid is still a
// process, for the nodes that have one.
//
// Best effort, and deliberately silent about its own failures: this answers
// "what is running", and a machine that cannot be reached has not told us the
// node is gone. Absent from the map means "not asked or could not ask", which
// a caller must not read as "dead" — the map only ever carries answers.
func (w *Workspace) livePIDs(ctx context.Context) map[int]bool {
	out := map[int]bool{}
	for _, ns := range w.state.Nodes {
		if ns.PID <= 0 {
			continue
		}
		t, err := w.machineFor(ns)
		if err != nil {
			continue
		}
		insp, ok := t.Driver.(process.ProcessInspector)
		if !ok {
			continue
		}
		alive, err := insp.PIDAlive(ctx, ns.PID)
		if err != nil {
			continue
		}
		out[ns.Index] = alive
	}
	return out
}

// NodeHealth is one node's health probe result.
type NodeHealth struct {
	Index int    `json:"index"`
	PID   int    `json:"pid"`
	Block uint64 `json:"block"`
	Err   string `json:"error,omitempty"`
}

// Health probes every node's HTTP RPC for its latest block height. It does not
// mark a step — it is a read, re-runnable at any time.
func (w *Workspace) Health(ctx context.Context) ([]NodeHealth, error) {
	if len(w.state.Nodes) == 0 {
		return nil, fmt.Errorf("chainsetup: health: no node table — run `chain place` first")
	}
	out := make([]NodeHealth, len(w.state.Nodes))
	for i, ns := range w.state.Nodes {
		h := NodeHealth{Index: ns.Index, PID: ns.PID}
		// A network spread across a set places each node on its own address, so the probe asks the
		// node's recorded host, not the target-level one; the resource layer turns
		// that into the address this machine actually dials.
		url, err := w.nodeHTTPURL(ns)
		if err != nil {
			return nil, err
		}
		c := rpc.Dial(url)
		if bn, err := c.BlockNumber(ctx); err != nil {
			h.Err = err.Error()
		} else {
			h.Block = bn
		}
		out[i] = h
	}
	return out, nil
}

// Preflight is the check-only entry: the same pre-launch inspection Start
// runs, callable without composing anything. It answers "may a network of
// this shape start here right now?" with the refusal Start would give — port
// occupancy plus unmanaged copies of the binary already on the resource.
