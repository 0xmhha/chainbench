package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"
)

// WebNodeObserved is the collector's latest look at one node's machine: what
// `chainbench chain observe` reported and when. It is a reading with a time,
// not proof the node is still in that state; controls still require a fresh
// observation made with the caller's own logins.
type WebNodeObserved struct {
	State      string
	Reason     string
	PID        int
	Executable string
	At         time.Time
}

// Observed returns the latest reading of every node in network, keyed by node
// label. A network the collector has not observed since this process started
// has none: a reading never outlives the process that took it.
func (m *WebMonitor) Observed(network string) map[string]WebNodeObserved {
	m.seenMu.Lock()
	defer m.seenMu.Unlock()
	out := map[string]WebNodeObserved{}
	for label, seen := range m.seen[network] {
		out[label] = seen
	}
	return out
}

// observeNetwork reads what each node's machine shows. Local nodes are read
// through the CLI; nodes on SSH targets need an operator's logins, which the
// collector does not hold, so they are recorded as unobserved with that reason.
func (m *WebMonitor) observeNetwork(ctx context.Context, network string, state State, local bool, now time.Time) {
	readings := map[string]WebNodeObserved{}
	unknown := func(reason string) {
		for _, ns := range state.Nodes {
			readings[string(ns.NodeLabel())] = WebNodeObserved{State: "unknown", Reason: reason, At: now}
		}
	}
	switch {
	case !local:
		unknown("remote_observation_unavailable")
	default:
		dir := filepath.Join(m.root, "networks", network)
		var out ChainObserveOut
		raw, err := m.cli.Run(ctx, "chain", "observe", "--json", "--workspace-dir", dir)
		if err == nil {
			err = json.Unmarshal(raw, &out)
		}
		if err != nil {
			unknown("probe_unavailable")
			break
		}
		unknown("probe_unavailable") // A node the CLI did not report stays unknown.
		ledgerMatches := verifyWebRecordLedger(state, dir) == nil
		for _, n := range out.Nodes {
			seen := WebNodeObserved{State: string(n.State), Reason: n.Reason, PID: n.PID, Executable: n.Executable, At: now}
			if !ledgerMatches && seen.State != "unrecorded_running" {
				seen.State, seen.Reason = "ownership_mismatch", "ledger_mismatch"
			}
			readings[n.Label] = seen
		}
	}
	m.seenMu.Lock()
	defer m.seenMu.Unlock()
	if m.seen == nil {
		m.seen = map[string]map[string]WebNodeObserved{}
	}
	m.seen[network] = readings
}
