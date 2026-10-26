package app

import (
	"sync"
	"time"
)

// webObservationBuffer is how many events a subscriber may fall behind before
// the collector starts dropping them for it.
const webObservationBuffer = 256

// webLogEventLines caps the log lines one event carries for a node per round;
// the rest stay in the archive for the logs API.
const webLogEventLines = 50

// WebObservationEvent is one live change the collector saw, for the events
// stream. It is best effort: a reader that misses one rereads the metrics and
// logs APIs, so these carry no stream cursor.
type WebObservationEvent struct {
	// Type is node, metrics, gap or logs.
	Type       string    `json:"type"`
	NetworkID  string    `json:"networkId"`
	ObservedAt time.Time `json:"observedAt"`
	// Node is a node whose reading changed (type node).
	Node *WebNodeReading `json:"node,omitempty"`
	// Samples are one round's samples of the network (type metrics).
	Samples []WebMetricPoint `json:"samples,omitempty"`
	// Gap opened or closed (type gap); GapState says which.
	Gap      *WebCoverageGap `json:"gap,omitempty"`
	GapState string          `json:"gapState,omitempty"`
	// NodeID, Lines and LinesDropped are one node's new log lines (type logs),
	// the newest webLogEventLines of them, with how many earlier ones were left
	// for the logs API.
	NodeID       string        `json:"nodeId,omitempty"`
	Lines        []WebLogEntry `json:"lines,omitempty"`
	LinesDropped int           `json:"linesDropped,omitempty"`
}

// WebNodeReading is a node's latest reading as the events stream carries it.
type WebNodeReading struct {
	ID                string    `json:"id"`
	State             string    `json:"state"`
	ObservationReason string    `json:"observationReason,omitempty"`
	ObservedPID       int       `json:"observedPid,omitempty"`
	ObservedAt        time.Time `json:"observedAt"`
}

// WebMetricPoint is one sample with the series it belongs to.
type WebMetricPoint struct {
	NodeID    string    `json:"nodeId"`
	Name      string    `json:"name"`
	Metric    string    `json:"metric"`
	Canonical bool      `json:"canonical"`
	Source    string    `json:"source"`
	Unit      string    `json:"unit,omitempty"`
	Time      time.Time `json:"time"`
	Value     float64   `json:"value"`
}

type webObservationHub struct {
	mu          sync.Mutex
	subscribers map[chan WebObservationEvent]struct{}
	dropped     uint64
}

// SubscribeObservations returns the live events and a function that ends the
// subscription. A subscriber that does not keep up loses events rather than
// slowing collection; ObservationDrops counts them.
func (m *WebMonitor) SubscribeObservations() (<-chan WebObservationEvent, func()) {
	m.events.mu.Lock()
	defer m.events.mu.Unlock()
	if m.events.subscribers == nil {
		m.events.subscribers = map[chan WebObservationEvent]struct{}{}
	}
	channel := make(chan WebObservationEvent, webObservationBuffer)
	m.events.subscribers[channel] = struct{}{}
	var once sync.Once
	return channel, func() {
		once.Do(func() {
			m.events.mu.Lock()
			defer m.events.mu.Unlock()
			delete(m.events.subscribers, channel)
			close(channel)
		})
	}
}

// ObservationDrops is how many events subscribers have missed in total.
func (m *WebMonitor) ObservationDrops() uint64 {
	m.events.mu.Lock()
	defer m.events.mu.Unlock()
	return m.events.dropped
}

func (m *WebMonitor) publish(event WebObservationEvent) {
	m.events.mu.Lock()
	defer m.events.mu.Unlock()
	for subscriber := range m.events.subscribers {
		select {
		case subscriber <- event:
		default:
			m.events.dropped++
		}
	}
}

// publishRound announces one network's round: the samples, the gaps that
// opened or closed, and each node's new log lines.
func (m *WebMonitor) publishRound(network string, now time.Time, samples []webMonitorSample, before, open, ended []webMonitorGap, lines map[string][]webMonitorLogLine) {
	if len(samples) > 0 {
		points := make([]WebMetricPoint, 0, len(samples))
		for _, s := range samples {
			metric, canonical := webMetricConcept(s.Name)
			points = append(points, WebMetricPoint{NodeID: s.Node, Name: s.Name, Metric: metric, Canonical: canonical, Source: s.Source, Unit: s.Unit, Time: s.Time, Value: s.Value})
		}
		m.publish(WebObservationEvent{Type: "metrics", NetworkID: network, ObservedAt: now, Samples: points})
	}
	key := func(g webMonitorGap) string { return g.Node + "\x00" + g.Source + "\x00" + g.Reason }
	known := map[string]bool{}
	for _, g := range before {
		known[key(g)] = true
	}
	for _, g := range open {
		if !known[key(g)] {
			gap := webCoverageGap(g)
			m.publish(WebObservationEvent{Type: "gap", NetworkID: network, ObservedAt: now, Gap: &gap, GapState: "opened"})
		}
	}
	for _, g := range ended {
		gap := webCoverageGap(g)
		m.publish(WebObservationEvent{Type: "gap", NetworkID: network, ObservedAt: now, Gap: &gap, GapState: "closed"})
	}
	for label, got := range lines {
		if len(got) == 0 {
			continue
		}
		dropped := max(0, len(got)-webLogEventLines)
		entries := make([]WebLogEntry, 0, len(got)-dropped)
		for _, line := range got[dropped:] {
			entries = append(entries, WebLogEntry{Time: line.Time, Text: line.Text, Level: parseLogLevel(line.Text)})
		}
		m.publish(WebObservationEvent{Type: "logs", NetworkID: network, ObservedAt: now, NodeID: label, Lines: entries, LinesDropped: dropped})
	}
}
