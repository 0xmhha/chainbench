package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

// webObservationWindow is the archived interval a history run owns on one
// network. A zero To means the run is still open.
type webObservationWindow struct {
	Network  string
	From, To time.Time
}

func (w webObservationWindow) covers(t time.Time) bool {
	return !t.Before(w.From) && (w.To.IsZero() || !t.After(w.To))
}

// UseRuns lets metric and log queries select a history run's window.
func (m *WebMonitor) UseRuns(resolve func(runID string) (webObservationWindow, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs = resolve
}

// queryWindow resolves an explicit window or a run's window on this network.
func (m *WebMonitor) queryWindow(network string, q WebMonitorQuery) (time.Time, time.Time, error) {
	if q.RunID == "" {
		return m.window(q)
	}
	if !q.From.IsZero() || !q.To.IsZero() {
		return time.Time{}, time.Time{}, ErrWebMonitorQuery
	}
	m.mu.Lock()
	resolve := m.runs
	m.mu.Unlock()
	if resolve == nil {
		return time.Time{}, time.Time{}, ErrDeploymentNotFound
	}
	w, err := resolve(q.RunID)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if w.Network != network {
		return time.Time{}, time.Time{}, ErrDeploymentNotFound
	}
	// A run's own window is used as recorded, even beyond the explicit limit.
	return w.From.UTC(), w.To.UTC(), nil
}

// deleteWindow removes archived samples and log lines inside a run's window,
// except instants another run or a running job still covers, and records the
// removed intervals as gaps so they never read as missing collection. Every
// rewrite is idempotent, so a failed deletion can simply be retried.
func (m *WebMonitor) deleteWindow(runID string, w webObservationWindow, protected []webObservationWindow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	to := w.To
	if to.IsZero() {
		return ErrDeploymentConflict // An open run still owns new observations.
	}
	// Record the deletion first: if a rewrite then fails, the removed part is
	// still reported, and a retry only duplicates gaps that coverage merges.
	var gaps []webMonitorGap
	for _, interval := range webUnprotected(w, protected) {
		gaps = append(gaps, webMonitorGap{From: interval.From, To: interval.To, Source: "archive", Reason: "deleted_by_admin " + runID})
	}
	if err := appendWebMonitorLines(m.store, w.Network, "gaps.jsonl", gaps); err != nil {
		return err
	}
	logs, err := m.store.Logs(w.Network)
	if err != nil {
		return err
	}
	var records []string
	for _, day := range webMonitorDays(w.From, to) {
		records = append(records, webMonitorDay("samples", day)+".jsonl")
		suffix := "-" + day.UTC().Format("20060102") + ".jsonl"
		for _, record := range logs {
			if strings.HasSuffix(record, suffix) {
				records = append(records, record)
			}
		}
	}
	owned := func(t time.Time) bool {
		if !w.covers(t) {
			return false
		}
		for _, p := range protected {
			if p.Network == w.Network && p.covers(t) {
				return false
			}
		}
		return true
	}
	for _, record := range records {
		if err = m.dropArchived(w.Network, record, owned); err != nil {
			return err
		}
	}
	return nil
}

// dropArchived rewrites one record without the lines whose time is owned.
// Lines that cannot be read are kept: deletion never guesses.
func (m *WebMonitor) dropArchived(network, record string, owned func(time.Time) bool) error {
	f, err := m.store.Open(network, record)
	if err != nil {
		return err
	}
	var kept bytes.Buffer
	changed := false
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			kept.Write(line) // A trailing partial line is still being written.
			break
		}
		if err != nil {
			_ = f.Close()
			return err
		}
		var stamped struct {
			Time time.Time `json:"t"`
		}
		if json.Unmarshal(line, &stamped) == nil && owned(stamped.Time) {
			changed = true
			continue
		}
		kept.Write(line)
	}
	_ = f.Close()
	if !changed {
		return nil
	}
	return m.store.Replace(network, record, kept.Bytes())
}

// webUnprotected returns the parts of w no protected window covers.
func webUnprotected(w webObservationWindow, protected []webObservationWindow) []webObservationWindow {
	out := []webObservationWindow{w}
	for _, p := range protected {
		if p.Network != w.Network {
			continue
		}
		var next []webObservationWindow
		for _, part := range out {
			if !p.To.IsZero() && p.To.Before(part.From) || p.From.After(part.To) {
				next = append(next, part)
				continue
			}
			if p.From.After(part.From) {
				next = append(next, webObservationWindow{Network: w.Network, From: part.From, To: p.From})
			}
			if !p.To.IsZero() && p.To.Before(part.To) {
				next = append(next, webObservationWindow{Network: w.Network, From: p.To, To: part.To})
			}
		}
		out = next
	}
	return out
}
