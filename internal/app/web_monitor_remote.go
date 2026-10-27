package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// webRemoteNodeTimeout bounds one node's remote reads so an unresponsive host
// cannot hold a collection round indefinitely.
const webRemoteNodeTimeout = 30 * time.Second

// attachRemote marks a network as having an operator-started remote log
// collector; the returned function closes its open gaps and detaches it.
func (m *WebMonitor) attachRemote(network string) func() {
	m.mu.Lock()
	m.remote[network]++
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.remote[network]--
		cursor, err := m.readCursor(network)
		if err != nil || len(cursor.RemoteOpen) == 0 {
			return
		}
		if appendWebMonitorLines(m.store, network, "gaps.jsonl", cursor.RemoteOpen) != nil {
			return
		}
		ended := cursor.RemoteOpen
		cursor.RemoteOpen = nil
		if raw, err := json.Marshal(cursor); err == nil {
			_ = m.store.WriteCursor(network, raw)
		}
		m.publishRound(network, m.now().UTC(), nil, ended, nil, ended, nil)
	}
}

// CollectRemoteLogs archives one round of remote node logs. open returns the
// caller's own access for a node. Remote reads run without the monitor lock so
// a slow host never stalls other collection. When the round is cancelled or
// access is refused, the positions of nodes already archived are kept and
// nothing else is recorded.
func (m *WebMonitor) CollectRemoteLogs(ctx context.Context, network string, open func(context.Context, State, node.Record) (webLogSource, error)) error {
	m.mu.Lock()
	state, _, err := m.loadNetwork(ctx, network)
	var work webMonitorCursor
	if err == nil {
		work, err = m.readCursor(network)
	}
	m.mu.Unlock()
	if err != nil {
		return err
	}
	now := m.now().UTC()
	since := now.Add(-m.interval)
	if work.RemoteLast.Before(now) && now.Sub(work.RemoteLast) <= 3*m.interval {
		since = work.RemoteLast // Continue from this collector's previous round.
	}
	var gaps []webMonitorGap
	var done []string
	var pending []webPendingLines
	committed := 0 // pending lines belonging to nodes whose positions are kept
	var stop error
	for _, ns := range state.Nodes {
		if ns.PID <= 0 {
			continue
		}
		label := string(ns.NodeLabel())
		nodeCtx, cancel := context.WithTimeout(ctx, webRemoteNodeTimeout)
		source, err := open(nodeCtx, state, ns)
		if err == nil {
			var g []webMonitorGap
			before := len(pending)
			g, err = m.archiveLog(nodeCtx, network, label, ns.LogPath, source, &work, since, now, &pending)
			if err != nil {
				pending = pending[:before] // A failed read archives nothing for this node.
			}
			gaps = append(gaps, g...)
		}
		cancel()
		if ctx.Err() != nil || errors.Is(err, ErrDeploymentForbidden) {
			stop = errors.Join(ctx.Err(), err)
			pending = pending[:committed]
			break
		}
		done = append(done, label)
		committed = len(pending)
		if err != nil {
			gaps = append(gaps, webMonitorGap{From: since, To: now, Node: label, Source: "logs", Reason: "collector_error"})
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cursor, err := m.readCursor(network)
	if err != nil {
		return err
	}
	// Lines are appended under the lock so an administrative deletion that
	// rewrites the same segment cannot lose them.
	lines := map[string][]webMonitorLogLine{}
	for _, p := range pending {
		if err = appendWebMonitorLines(m.store, network, p.record, p.lines); err != nil {
			return err // Positions stay unmoved, so the lines are read again.
		}
		lines[p.label] = append(lines[p.label], p.lines...)
	}
	for _, label := range done { // The background collector never moves remote positions.
		cursor.Offsets[label], cursor.Files[label], cursor.Heads[label] = work.Offsets[label], work.Files[label], work.Heads[label]
		cursor.Skipping[label], cursor.InSecret[label] = work.Skipping[label], work.InSecret[label]
	}
	before := cursor.RemoteOpen
	var ended []webMonitorGap
	if stop == nil {
		cursor.RemoteOpen, ended = mergeWebMonitorGaps(cursor.RemoteOpen, gaps)
		if err = appendWebMonitorLines(m.store, network, "gaps.jsonl", ended); err != nil {
			return err
		}
		cursor.RemoteLast = now
	}
	var raw bytes.Buffer
	if err = json.NewEncoder(&raw).Encode(cursor); err != nil {
		return err
	}
	if err = m.store.WriteCursor(network, raw.Bytes()); err != nil {
		return err
	}
	// The lines and gap changes reach the events stream as a local round's do.
	m.publishRound(network, now, nil, before, cursor.RemoteOpen, ended, lines)
	return stop
}
