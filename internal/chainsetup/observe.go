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

// LivePIDs asks each node's machine whether its recorded pid is still a
// process, for the nodes that have one.
//
// Best effort, and deliberately silent about its own failures: this answers
// "what is running", and a machine that cannot be reached has not told us the
// node is gone. Absent from the map means "not asked or could not ask", which
// a caller must not read as "dead" — the map only ever carries answers.
func (w *Workspace) LivePIDs(ctx context.Context) map[int]bool {
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
	if err := w.allow("Health"); err != nil {
		return nil, err
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

// Endpoints is every node's reachable RPC URL, in node order.
//
// The resolution is one helper shared with NodeSet and Health, so the three
// cannot disagree about where a node answers — which is why the loop lives here
// rather than in the verb that asks for it.
func (w *Workspace) Endpoints() ([]string, error) {
	if len(w.state.Nodes) == 0 {
		return nil, fmt.Errorf("chainsetup: endpoints: no node table — run `chain place` first")
	}
	urls := make([]string, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		url, err := w.nodeHTTPURL(ns)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}
