package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// webRemoteNodeTimeout bounds one node's remote reads so an unresponsive host
// cannot hold a collection round indefinitely.
const webRemoteNodeTimeout = 30 * time.Second

// webRemoteLogs reads a node log over one SSH runner. Each command is a new
// connection, so the runner's credential guard rechecks revocation every time.
type webRemoteLogs struct{ run process.Runner }

// Snapshot reports identity, size and bytes from offset in one command, so a
// rotation between separate stat and read commands cannot mix two files.
func (r webRemoteLogs) Snapshot(ctx context.Context, path string, offset int64, limit int) (uint64, int64, []byte, error) {
	q := remote.ShellQuote(path)
	cmd := "test -f " + q + " || exit 3; set -- $(ls -diL " + q + "); s=$(wc -c < " + q + "); echo \"$1 $s\"; tail -c +" +
		strconv.FormatInt(offset+1, 10) + " " + q + " | head -c " + strconv.Itoa(limit)
	res, err := r.run(ctx, cmd)
	if err != nil {
		return 0, 0, nil, err
	}
	if res.ExitCode == 3 {
		return 0, 0, nil, os.ErrNotExist
	}
	header, data, found := strings.Cut(res.Stdout, "\n")
	fields := strings.Fields(header)
	if res.ExitCode != 0 || !found || len(fields) != 2 {
		return 0, 0, nil, fmt.Errorf("remote log read %s: exit %d", path, res.ExitCode)
	}
	id, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, 0, nil, err
	}
	size, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, 0, nil, err
	}
	return id, size, []byte(data), nil
}

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
		cursor.RemoteOpen = nil
		if raw, err := json.Marshal(cursor); err == nil {
			_ = m.store.WriteCursor(network, raw)
		}
	}
}

// CollectRemoteLogs archives one round of remote node logs. open returns the
// caller's own access for a node. Remote reads run without the monitor lock so
// a slow host never stalls other collection. When the round is cancelled or
// access is refused, the positions of nodes already archived are kept and
// nothing else is recorded.
func (m *WebMonitor) CollectRemoteLogs(ctx context.Context, network string, open func(context.Context, State, node.Record) (webLogSource, error)) error {
	m.mu.Lock()
	state, _, err := m.loadNetwork(network)
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
			g, err = m.archiveLog(nodeCtx, network, label, ns.LogPath, source, &work, since, now)
			gaps = append(gaps, g...)
		}
		cancel()
		if ctx.Err() != nil || errors.Is(err, ErrDeploymentForbidden) {
			stop = errors.Join(ctx.Err(), err)
			break
		}
		done = append(done, label)
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
	for _, label := range done { // The background collector never moves remote positions.
		cursor.Offsets[label], cursor.Files[label] = work.Offsets[label], work.Files[label]
		cursor.Skipping[label], cursor.InSecret[label] = work.Skipping[label], work.InSecret[label]
	}
	if stop == nil {
		var ended []webMonitorGap
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
	return stop
}
