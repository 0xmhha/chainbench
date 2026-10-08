package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// webMonitorWindow is the default chart and log window; a request may cover at
// most webMonitorMaxWindow. A series above webMonitorMaxPoints samples is
// averaged into equal time buckets.
const (
	webMonitorWindow    = 2 * time.Hour
	webMonitorMaxWindow = 31 * 24 * time.Hour
	webMonitorMaxPoints = 720
	webMonitorLogLimit  = 100
)

type WebMonitorQuery struct {
	From, To time.Time
	NodeID   string
	Metric   string
	Cursor   string
	Limit    int
}

type WebCoverageGap struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Reason string    `json:"reason"`
}

type WebCoverage struct {
	From                  time.Time        `json:"from"`
	To                    time.Time        `json:"to"`
	SampleIntervalSeconds float64          `json:"sampleIntervalSeconds"`
	Complete              bool             `json:"complete"`
	Gaps                  []WebCoverageGap `json:"gaps"`
	Downsampled           bool             `json:"downsampled"`
	TimestampSource       string           `json:"timestampSource,omitempty"`
}

type WebMetricSample struct {
	Time  time.Time `json:"time"`
	Value *float64  `json:"value"`
}

type WebMetricSeries struct {
	NodeID  string            `json:"nodeId"`
	Name    string            `json:"name"`
	Unit    string            `json:"unit,omitempty"`
	Source  string            `json:"source"`
	Labels  map[string]string `json:"labels,omitempty"`
	Samples []WebMetricSample `json:"samples"`
}

type WebMetrics struct {
	NetworkID string            `json:"networkId"`
	Series    []WebMetricSeries `json:"series"`
	Coverage  WebCoverage       `json:"coverage"`
}

type WebLogEntry struct {
	Time            time.Time `json:"time"`
	Text            string    `json:"text"`
	TimestampSource string    `json:"-"`
}

type WebLogs struct {
	NodeID     string        `json:"nodeId"`
	RunID      string        `json:"runId,omitempty"`
	Entries    []WebLogEntry `json:"entries"`
	Coverage   WebCoverage   `json:"coverage"`
	NextCursor *string       `json:"nextCursor"`
}

// ErrWebMonitorQuery reports an unusable time window, cursor or limit.
var ErrWebMonitorQuery = errors.New("invalid observation query")

// readable admits every active role: collected observations are shared and
// redacted before they are archived.
func (m *WebMonitor) readable(a DeploymentActor, network string) error {
	switch a.Role {
	case "admin", "operator", "viewer":
	default:
		return ErrDeploymentForbidden
	}
	if a.ID == "" {
		return ErrDeploymentForbidden
	}
	if network == "" || strings.ContainsAny(network, "/\\") || network == "." || network == ".." {
		return ErrDeploymentNotFound
	}
	if _, err := os.Stat(filepath.Join(m.root, "networks", network, "chain-record.json")); err != nil {
		return ErrDeploymentNotFound
	}
	return nil
}

func (m *WebMonitor) window(q WebMonitorQuery) (time.Time, time.Time, error) {
	to := q.To
	if to.IsZero() {
		to = m.now()
	}
	from := q.From
	if from.IsZero() {
		from = to.Add(-webMonitorWindow)
	}
	if to.Before(from) || to.Sub(from) > webMonitorMaxWindow {
		return from, to, ErrWebMonitorQuery
	}
	return from.UTC(), to.UTC(), nil
}

// Metrics returns archived samples in the window with merged collection gaps.
func (m *WebMonitor) Metrics(a DeploymentActor, network string, q WebMonitorQuery) (WebMetrics, error) {
	if err := m.readable(a, network); err != nil {
		return WebMetrics{}, err
	}
	from, to, err := m.window(q)
	if err != nil {
		return WebMetrics{}, err
	}
	bySeries := map[string]*WebMetricSeries{}
	corrupt := 0
	for _, day := range webMonitorDays(from, to) {
		n, err := scanWebMonitorLines(m.store, network, webMonitorDay("samples", day)+".jsonl", func(s webMonitorSample) {
			if s.Time.Before(from) || s.Time.After(to) || (q.NodeID != "" && s.Node != q.NodeID) || (q.Metric != "" && s.Name != q.Metric) {
				return
			}
			key := s.Node + "\x00" + s.Source + "\x00" + s.Name
			series, ok := bySeries[key]
			if !ok {
				series = &WebMetricSeries{NodeID: s.Node, Name: s.Name, Unit: s.Unit, Source: s.Source, Samples: []WebMetricSample{}}
				bySeries[key] = series
			}
			value := s.Value
			series.Samples = append(series.Samples, WebMetricSample{Time: s.Time, Value: &value})
		})
		if err != nil {
			return WebMetrics{}, err
		}
		corrupt += n
	}
	keys := make([]string, 0, len(bySeries))
	for key := range bySeries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := WebMetrics{NetworkID: network, Series: []WebMetricSeries{}}
	downsampled := false
	for _, key := range keys {
		series := *bySeries[key]
		if len(series.Samples) > webMonitorMaxPoints {
			series.Samples = downsampleWebMetric(series.Samples, from, to)
			downsampled = true
		}
		out.Series = append(out.Series, series)
	}
	gaps, err := m.coverageGaps(network, from, to, corrupt, func(g webMonitorGap) bool {
		return g.Source != "logs" && (q.NodeID == "" || g.Node == "" || g.Node == q.NodeID) && (q.Metric == "" || g.Source == "collector" || !strings.HasSuffix(g.Reason, " not_exposed") || strings.HasPrefix(g.Reason, q.Metric+" "))
	})
	if err != nil {
		return WebMetrics{}, err
	}
	out.Coverage = WebCoverage{From: from, To: to, SampleIntervalSeconds: m.interval.Seconds(), Complete: len(gaps) == 0, Gaps: gaps, Downsampled: downsampled, TimestampSource: "collection"}
	return out, nil
}

// Logs returns archived node lines in the window, in node log order. nodeID is
// "<network>.<label>". A cursor carries the window it was issued for.
func (m *WebMonitor) Logs(a DeploymentActor, nodeID string, q WebMonitorQuery) (WebLogs, error) {
	cut := strings.LastIndexByte(nodeID, '.')
	if cut <= 0 || cut == len(nodeID)-1 {
		return WebLogs{}, ErrDeploymentNotFound
	}
	network, label := nodeID[:cut], nodeID[cut+1:]
	if err := m.readable(a, network); err != nil {
		return WebLogs{}, err
	}
	state, _, err := m.loadNetwork(network)
	if err != nil {
		return WebLogs{}, err
	}
	known := false
	for _, ns := range state.Nodes {
		known = known || string(ns.NodeLabel()) == label
	}
	if !known {
		return WebLogs{}, ErrDeploymentNotFound
	}
	start := 0
	var from, to time.Time
	if q.Cursor != "" {
		if start, from, to, err = parseWebLogCursor(q.Cursor); err != nil {
			return WebLogs{}, err
		}
		if (!q.From.IsZero() && !q.From.Equal(from)) || (!q.To.IsZero() && !q.To.Equal(to)) {
			return WebLogs{}, ErrWebMonitorQuery
		}
	} else if from, to, err = m.window(q); err != nil {
		return WebLogs{}, err
	}
	limit := q.Limit
	if limit <= 0 || limit > webMonitorLogLimit {
		limit = webMonitorLogLimit
	}
	out := WebLogs{NodeID: nodeID, Entries: []WebLogEntry{}}
	sources := map[string]bool{}
	matched, corrupt := 0, 0
	for _, day := range webMonitorDays(from, to) {
		n, err := scanWebMonitorLines(m.store, network, webMonitorDay("logs/"+label, day)+".jsonl", func(line webMonitorLogLine) {
			if line.Time.Before(from) || line.Time.After(to) {
				return
			}
			matched++
			if matched <= start || out.NextCursor != nil {
				return
			}
			if len(out.Entries) == limit {
				next := formatWebLogCursor(start+limit, from, to)
				out.NextCursor = &next
				return
			}
			out.Entries = append(out.Entries, WebLogEntry{Time: line.Time, Text: line.Text, TimestampSource: line.TimestampSource})
			sources[line.TimestampSource] = true
		})
		if err != nil {
			return WebLogs{}, err
		}
		corrupt += n
	}
	gaps, err := m.coverageGaps(network, from, to, corrupt, func(g webMonitorGap) bool {
		return (g.Source == "logs" && g.Node == label) || g.Source == "collector"
	})
	if err != nil {
		return WebLogs{}, err
	}
	stamp := "collection"
	switch {
	case sources["source"] && sources["collection"]:
		stamp = "mixed"
	case sources["source"]:
		stamp = "source"
	}
	out.Coverage = WebCoverage{From: from, To: to, Complete: len(gaps) == 0, Gaps: gaps, TimestampSource: stamp}
	return out, nil
}

func formatWebLogCursor(index int, from, to time.Time) string {
	return strconv.Itoa(index) + "-" + strconv.FormatInt(from.UnixNano(), 10) + "-" + strconv.FormatInt(to.UnixNano(), 10)
}

func parseWebLogCursor(raw string) (int, time.Time, time.Time, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 3 {
		return 0, time.Time{}, time.Time{}, ErrWebMonitorQuery
	}
	index, err1 := strconv.Atoi(parts[0])
	from, err2 := strconv.ParseInt(parts[1], 10, 64)
	to, err3 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || index < 0 || to < from || time.Duration(to-from) > webMonitorMaxWindow {
		return 0, time.Time{}, time.Time{}, ErrWebMonitorQuery
	}
	return index, time.Unix(0, from).UTC(), time.Unix(0, to).UTC(), nil
}

// coverageGaps lists archived and still-open gaps in the window, merging
// touching intervals of the same failure, plus the time before the first
// collection round and any unreadable archive lines.
func (m *WebMonitor) coverageGaps(network string, from, to time.Time, corrupt int, keep func(webMonitorGap) bool) ([]WebCoverageGap, error) {
	var recorded []webMonitorGap
	n, err := scanWebMonitorLines(m.store, network, "gaps.jsonl", func(g webMonitorGap) { recorded = append(recorded, g) })
	if err != nil {
		return nil, err
	}
	corrupt += n
	cursor, err := m.readCursor(network)
	if err != nil {
		return nil, err
	}
	recorded = append(append(recorded, cursor.Open...), cursor.RemoteOpen...)
	first := cursor.FirstCollected
	if first.IsZero() || first.After(to) {
		first = to
	}
	if from.Before(first) {
		recorded = append(recorded, webMonitorGap{From: from, To: first, Source: "collector", Reason: "not_collected"})
	}
	sort.SliceStable(recorded, func(i, j int) bool { return recorded[i].From.Before(recorded[j].From) })
	type key struct{ node, source, reason string }
	open := map[key]int{}
	out := []WebCoverageGap{}
	for _, g := range recorded {
		if g.To.Before(from) || g.From.After(to) || !keep(g) {
			continue
		}
		k := key{g.Node, g.Source, g.Reason}
		if i, ok := open[k]; ok && !g.From.After(out[i].To.Add(time.Second)) {
			if g.To.After(out[i].To) {
				out[i].To = g.To
			}
			continue
		}
		open[k] = len(out)
		out = append(out, WebCoverageGap{From: g.From, To: g.To, Reason: strings.TrimSpace(strings.Join([]string{g.Node, g.Source, g.Reason}, " "))})
	}
	if corrupt > 0 {
		out = append(out, WebCoverageGap{From: from, To: to, Reason: "archive corrupt_record " + strconv.Itoa(corrupt)})
	}
	return out, nil
}

func webMonitorDays(from, to time.Time) []time.Time {
	var days []time.Time
	for day := from.UTC().Truncate(24 * time.Hour); !day.After(to); day = day.Add(24 * time.Hour) {
		days = append(days, day)
	}
	return days
}

func downsampleWebMetric(samples []WebMetricSample, from, to time.Time) []WebMetricSample {
	span := to.Sub(from) / webMonitorMaxPoints
	if span <= 0 {
		return samples
	}
	out := []WebMetricSample{}
	var bucket time.Time
	var sum float64
	var count int
	flush := func() {
		if count > 0 {
			v := sum / float64(count)
			out = append(out, WebMetricSample{Time: bucket, Value: &v})
		}
	}
	for _, s := range samples {
		b := from.Add(s.Time.Sub(from) / span * span)
		if count > 0 && !b.Equal(bucket) {
			flush()
			sum, count = 0, 0
		}
		bucket = b
		sum += *s.Value
		count++
	}
	flush()
	return out
}

// scanWebMonitorLines streams one archive record. A trailing line without its
// newline is still being written and is skipped; an unreadable line is counted
// and reported rather than failing the whole read.
func scanWebMonitorLines[T any](store *session.ObservationStore, network, record string, fn func(T)) (int, error) {
	f, err := store.Open(network, record)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	reader := bufio.NewReader(f)
	corrupt := 0
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			return corrupt, nil
		}
		if err != nil {
			return corrupt, err
		}
		var item T
		if json.Unmarshal(line, &item) != nil {
			corrupt++
			continue
		}
		fn(item)
	}
}
