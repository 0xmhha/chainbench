package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
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

// webRemoteLogRounds bounds one drain of remote logs: each round reads at most
// webMonitorReadLimit from every node.
const webRemoteLogRounds = 64

// webCollectorError is the gap reason of a read that failed.
const webCollectorError = "collector_error"

// CollectRemoteLogsToEnd repeats CollectRemoteLogs until the logs are caught up,
// for a job whose logins end with it. A running node keeps logging, so caught up
// means no read filled its limit, not that nothing new arrived. It stops at
// webRemoteLogRounds, leaving the rest of a very long log unread.
func (m *WebMonitor) CollectRemoteLogsToEnd(ctx context.Context, network string, open func(context.Context, State, node.Record) (webLogSource, error)) error {
	return m.drainRemoteLogs(ctx, network, open, runningRemoteNodes)
}

// FinishRemoteLogs reads the logs of a network's stopped nodes to their end and
// forgets where those logs were. The network is being removed, so the next run's
// log is a first read and not a replacement of this one.
func (m *WebMonitor) FinishRemoteLogs(ctx context.Context, network string, open func(context.Context, State, node.Record) (webLogSource, error)) error {
	drainErr := m.drainRemoteLogs(ctx, network, open, everyRemoteNode)
	m.mu.Lock()
	defer m.mu.Unlock()
	cursor, err := m.readCursor(network)
	if err != nil {
		return errors.Join(drainErr, err)
	}
	cursor.Offsets, cursor.Files, cursor.Heads = map[string]int64{}, map[string]uint64{}, map[string]string{}
	cursor.Skipping, cursor.InSecret = map[string]bool{}, map[string]bool{}
	var raw bytes.Buffer
	if err = json.NewEncoder(&raw).Encode(cursor); err != nil {
		return errors.Join(drainErr, err)
	}
	return errors.Join(drainErr, m.store.WriteCursor(network, raw.Bytes()))
}

// remoteNodes selects whose logs a collection round reads.
type remoteNodes int

const (
	runningRemoteNodes remoteNodes = iota // Nodes recorded as launched.
	everyRemoteNode                       // Stopped nodes too, for their last lines.
)

func (m *WebMonitor) drainRemoteLogs(ctx context.Context, network string, open func(context.Context, State, node.Record) (webLogSource, error), which remoteNodes) error {
	for range webRemoteLogRounds {
		before, err := m.remotePositions(network)
		if err != nil {
			return err
		}
		if err = m.collectRemoteLogs(ctx, network, open, which); err != nil {
			return err
		}
		after, err := m.remotePositions(network)
		if err != nil {
			return err
		}
		if !filledARead(before, after) {
			return m.markRemoteDrained(network)
		}
	}
	return nil
}

// filledARead reports a node whose position moved by most of a read, which means
// more was waiting behind it. A read ends at a line break, so it stops just short
// of the limit; what a live node logs between two reads is far below half. A
// position that moved back started over in a new file, so it counts from zero.
func filledARead(before, after map[string]int64) bool {
	for label, position := range after {
		moved := position - before[label]
		if moved < 0 {
			moved = position
		}
		if moved >= webMonitorReadLimit/2 {
			return true
		}
	}
	return false
}

// markRemoteDrained records that the nodes read in the last round reached the
// end of their logs. A node whose read failed keeps its open gap and stays out;
// a replaced log was still read, so its gap does not.
func (m *WebMonitor) markRemoteDrained(network string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cursor, err := m.readCursor(network)
	if err != nil {
		return err
	}
	failed := map[string]bool{}
	for _, g := range cursor.RemoteOpen {
		failed[g.Node] = failed[g.Node] || g.Reason == webCollectorError
	}
	if cursor.RemoteDrained == nil {
		cursor.RemoteDrained = map[string]time.Time{}
	}
	now := m.now().UTC()
	for label := range cursor.Offsets {
		if !failed[label] {
			cursor.RemoteDrained[label] = now
		}
	}
	var raw bytes.Buffer
	if err = json.NewEncoder(&raw).Encode(cursor); err != nil {
		return err
	}
	return m.store.WriteCursor(network, raw.Bytes())
}

// remotePositions reads how far each node's log has been archived.
func (m *WebMonitor) remotePositions(network string) (map[string]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cursor, err := m.readCursor(network)
	if err != nil {
		return nil, err
	}
	return maps.Clone(cursor.Offsets), nil
}

// CollectRemoteLogs archives one round of remote node logs. open returns the
// caller's own access for a node. Remote reads run without the monitor lock so
// a slow host never stalls other collection. When the round is cancelled or
// access is refused, the positions of nodes already archived are kept and
// nothing else is recorded.
func (m *WebMonitor) CollectRemoteLogs(ctx context.Context, network string, open func(context.Context, State, node.Record) (webLogSource, error)) error {
	return m.collectRemoteLogs(ctx, network, open, runningRemoteNodes)
}

func (m *WebMonitor) collectRemoteLogs(ctx context.Context, network string, open func(context.Context, State, node.Record) (webLogSource, error), which remoteNodes) error {
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
		if ns.PID <= 0 && which == runningRemoteNodes {
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
			gaps = append(gaps, webMonitorGap{From: since, To: now, Node: label, Source: "logs", Reason: webCollectorError})
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
