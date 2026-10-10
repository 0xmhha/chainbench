package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webMonitorInterval is the default collection period from the Web UI spec.
const webMonitorInterval = 5 * time.Second

// webMetricUnits lists the metrics-endpoint gauges charted per node. Names are
// the Prometheus spellings of the geth-family gauges documented in each
// docs/chain-analysis/*/rpc-metrics-graph.md; a gauge a binary does not expose
// becomes a not_exposed gap instead of a zero.
var webMetricUnits = map[string]string{
	"chain_head_block": "blocks",
	"p2p_peers":        "peers",
	"txpool_pending":   "transactions",
	"txpool_queued":    "transactions",
}

var webMetricOrder = []string{"chain_head_block", "p2p_peers", "txpool_pending", "txpool_queued"}

// WebMonitor collects node observations for owned networks into an archive
// that never expires. Collection only reads: it never changes chain records,
// signals processes or treats recorded PIDs as proof of liveness.
type WebMonitor struct {
	root     string
	store    *session.ObservationStore
	redact   func(string) string
	now      func() time.Time
	interval time.Duration
	mu       sync.Mutex
	resumed  map[string]bool
	remote   map[string]int
	runs     func(string) (webObservationWindow, error)
}

type webMonitorSample struct {
	Time   time.Time `json:"t"`
	Node   string    `json:"n"`
	Name   string    `json:"m"`
	Unit   string    `json:"u"`
	Source string    `json:"s"`
	Value  float64   `json:"v"`
}

type webMonitorGap struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Node   string    `json:"n,omitempty"`
	Source string    `json:"s"`
	Reason string    `json:"r"`
}

// webMonitorCursor is the replaceable collection position. Ongoing gaps stay
// here and grow each round; a gap reaches gaps.jsonl only once it ends, so an
// outage of days is one archived interval rather than one line per round.
type webMonitorCursor struct {
	FirstCollected time.Time         `json:"firstCollected"`
	LastCollected  time.Time         `json:"lastCollected"`
	Offsets        map[string]int64  `json:"offsets"`
	Files          map[string]uint64 `json:"files,omitempty"`
	Heads          map[string]string `json:"heads,omitempty"`
	Skipping       map[string]bool   `json:"skipping,omitempty"`
	InSecret       map[string]bool   `json:"inSecret,omitempty"`
	Open           []webMonitorGap   `json:"open,omitempty"`
	RemoteLast     time.Time         `json:"remoteLast,omitempty"`
	RemoteOpen     []webMonitorGap   `json:"remoteOpen,omitempty"`
}

func OpenWebMonitor(root string, redact func(string) string) (*WebMonitor, error) {
	store, err := session.OpenObservationStore(filepath.Join(root, "observations"))
	if err != nil {
		return nil, err
	}
	if redact == nil {
		redact = func(s string) string { return s }
	}
	return &WebMonitor{root: root, store: store, redact: redact, now: time.Now, interval: webMonitorInterval, resumed: map[string]bool{}, remote: map[string]int{}}, nil
}

// Run collects until ctx ends. Failures are archived as coverage gaps.
func (m *WebMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		_ = m.CollectOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// CollectOnce samples every owned network once.
func (m *WebMonitor) CollectOnce(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(filepath.Join(m.root, "networks"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, entry := range entries {
		if entry.IsDir() {
			errs = append(errs, m.collectNetwork(ctx, entry.Name()))
		}
	}
	return errors.Join(errs...)
}

func (m *WebMonitor) loadNetwork(network string) (State, resource.Inspection, error) {
	dir := filepath.Join(m.root, "networks", network)
	var state State
	var target resource.Inspection
	raw, err := os.ReadFile(filepath.Join(dir, "chain-record.json"))
	if err != nil {
		return state, target, err
	}
	if err = json.Unmarshal(raw, &state); err != nil {
		return state, target, err
	}
	raw, err = os.ReadFile(filepath.Join(dir, "web-target.json"))
	if err != nil {
		return state, target, err
	}
	return state, target, json.Unmarshal(raw, &target)
}

func (m *WebMonitor) collectNetwork(ctx context.Context, network string) error {
	state, target, err := m.loadNetwork(network)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("monitor %s: %w", network, err)
	}
	cursor, err := m.readCursor(network)
	if err != nil {
		return err
	}
	now := m.now().UTC()
	since := now.Add(-m.interval)
	var closed, gaps []webMonitorGap
	if m.resumed[network] && !cursor.LastCollected.IsZero() && cursor.LastCollected.Before(now) {
		since = cursor.LastCollected // Everything since the last round is unobserved.
	}
	if !m.resumed[network] && !cursor.LastCollected.IsZero() && now.After(cursor.LastCollected) {
		// A new collector process cannot vouch for anything since the previous
		// process's last round, however short the restart was.
		closed = append(closed, webMonitorGap{From: cursor.LastCollected, To: now, Source: "collector", Reason: "collector_stopped"})
	}
	m.resumed[network] = true
	if cursor.FirstCollected.IsZero() {
		cursor.FirstCollected = now
	}
	var samples []webMonitorSample
	var errs []error
	for _, ns := range state.Nodes {
		label := string(ns.NodeLabel())
		if ns.PID <= 0 {
			continue // Not recorded as launched; nothing is expected to answer.
		}
		s, g := m.sampleNode(ctx, ns, now, since)
		samples, gaps = append(samples, s...), append(gaps, g...)
		if target.Transport != "local" {
			if m.remote[network] == 0 { // No operator-started collection job is running.
				gaps = append(gaps, webMonitorGap{From: since, To: now, Node: label, Source: "logs", Reason: "remote_log_collection_unavailable"})
			}
			continue
		}
		g, err := m.archiveLog(ctx, network, label, ns.LogPath, webLocalLogs{}, &cursor, since, now, nil)
		if err != nil {
			errs = append(errs, err)
			g = append(g, webMonitorGap{From: since, To: now, Node: label, Source: "logs", Reason: "collector_error"})
		}
		gaps = append(gaps, g...)
	}
	if err = appendWebMonitorLines(m.store, network, webMonitorDay("samples", now)+".jsonl", samples); err != nil {
		errs = append(errs, err)
		gaps = append(gaps, webMonitorGap{From: since, To: now, Source: "collector", Reason: "collector_error"})
	}
	open, ended := mergeWebMonitorGaps(cursor.Open, gaps)
	if err = appendWebMonitorLines(m.store, network, "gaps.jsonl", append(closed, ended...)); err != nil {
		return errors.Join(append(errs, err)...) // Keep the cursor: those gaps stay open.
	}
	cursor.Open, cursor.LastCollected = open, now
	raw, err := json.Marshal(cursor)
	if err == nil {
		err = m.store.WriteCursor(network, raw)
	}
	return errors.Join(append(errs, err)...)
}

// mergeWebMonitorGaps extends an open gap the same failure continues; an open
// gap that did not recur this round has ended.
func mergeWebMonitorGaps(open, current []webMonitorGap) (next, ended []webMonitorGap) {
	key := func(g webMonitorGap) string { return g.Node + "\x00" + g.Source + "\x00" + g.Reason }
	continuing := map[string]webMonitorGap{}
	for _, g := range current {
		continuing[key(g)] = g
	}
	seen := map[string]bool{}
	for _, g := range open {
		if c, ok := continuing[key(g)]; ok && !c.From.After(g.To.Add(time.Second)) {
			g.To = c.To
			next = append(next, g)
			seen[key(g)] = true
			continue
		}
		ended = append(ended, g)
	}
	for _, g := range current {
		if !seen[key(g)] {
			next = append(next, g)
			seen[key(g)] = true
		}
	}
	return next, ended
}

func (m *WebMonitor) sampleNode(ctx context.Context, ns node.Record, now, from time.Time) ([]webMonitorSample, []webMonitorGap) {
	label := string(ns.NodeLabel())
	host := ns.Host
	if host == "" {
		host = "127.0.0.1"
	}
	var samples []webMonitorSample
	var gaps []webMonitorGap
	gap := func(source, reason string) {
		gaps = append(gaps, webMonitorGap{From: from, To: now, Node: label, Source: source, Reason: reason})
	}
	if ns.HTTP <= 0 {
		gap("rpc", "rpc_disabled")
	} else {
		probe, cancel := context.WithTimeout(ctx, 2*time.Second)
		client := rpc.Dial(fmt.Sprintf("http://%s:%d", host, ns.HTTP))
		height, herr := client.BlockNumber(probe)
		peers, perr := client.PeerCount(probe)
		cancel()
		if herr != nil || perr != nil {
			gap("rpc", "rpc_unavailable")
		} else {
			samples = append(samples,
				webMonitorSample{Time: now, Node: label, Name: "block_height", Unit: "blocks", Source: "rpc", Value: float64(height)},
				webMonitorSample{Time: now, Node: label, Name: "peer_count", Unit: "peers", Source: "rpc", Value: float64(peers)})
		}
	}
	if ns.Metrics <= 0 {
		gap("metrics", "metrics_disabled")
		return samples, gaps
	}
	values, err := collector.ScrapeMetrics(ctx, collector.MetricsURL(host, ns.Metrics))
	if err != nil {
		gap("metrics", "metrics_unavailable")
		return samples, gaps
	}
	for _, name := range webMetricOrder {
		value, ok := values[name]
		if !ok {
			gap("metrics", name+" not_exposed")
			continue
		}
		samples = append(samples, webMonitorSample{Time: now, Node: label, Name: name, Unit: webMetricUnits[name], Source: "metrics", Value: value})
	}
	return samples, gaps
}

func (m *WebMonitor) readCursor(network string) (webMonitorCursor, error) {
	cursor := webMonitorCursor{}
	raw, err := m.store.ReadCursor(network)
	if err != nil {
		return cursor, err
	}
	if raw != nil {
		if err = json.Unmarshal(raw, &cursor); err != nil {
			return cursor, err
		}
	}
	if cursor.Offsets == nil {
		cursor.Offsets = map[string]int64{}
	}
	if cursor.Files == nil {
		cursor.Files = map[string]uint64{}
	}
	if cursor.Heads == nil {
		cursor.Heads = map[string]string{}
	}
	if cursor.Skipping == nil {
		cursor.Skipping = map[string]bool{}
	}
	if cursor.InSecret == nil {
		cursor.InSecret = map[string]bool{}
	}
	return cursor, nil
}

func appendWebMonitorLines[T any](store *session.ObservationStore, network, record string, items []T) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // Keep stored lines close to their source size.
	for _, item := range items {
		if err := encoder.Encode(item); err != nil {
			return err
		}
	}
	return store.Append(network, record, buf.Bytes())
}

// webMonitorDay names the UTC daily segment of an archive record.
func webMonitorDay(prefix string, t time.Time) string {
	return prefix + "-" + t.UTC().Format("20060102")
}
