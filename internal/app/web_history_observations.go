package app

import (
	"errors"
	"fmt"
)

// ErrHistoryObservationsIncomplete reports that a run's archived metrics and
// logs were only partly removed; the run stays listed so deletion can be retried.
var ErrHistoryObservationsIncomplete = errors.New("run observations were not fully deleted; the run is kept for retry")

// UseObservations links history runs to the metric and log archive: run
// detail shows the archived window and deletion removes the run-owned part.
func (s *WebHistory) UseObservations(m *WebMonitor) {
	s.mu.Lock()
	s.observations = m
	s.mu.Unlock()
	m.UseRuns(s.observationWindow)
}

func (s *WebHistory) observationWindow(runID string) (webObservationWindow, error) {
	s.mu.Lock()
	capture, found := s.state.Captures[runID]
	s.mu.Unlock()
	if !found {
		return webObservationWindow{}, ErrDeploymentNotFound
	}
	w, ok := s.windowOf(capture.Run)
	if !ok {
		return webObservationWindow{}, ErrDeploymentNotFound
	}
	if w.To.IsZero() {
		w.To = s.observations.now().UTC()
	}
	return w, nil
}

// windowOf derives a run's archived interval from its durable job: creation
// until the last finished phase, or open while the job has not ended.
func (s *WebHistory) windowOf(run WebRun) (webObservationWindow, bool) {
	if s.jobs == nil || run.JobID == "" {
		return webObservationWindow{}, false
	}
	job, err := s.jobs.Get(run.JobID)
	if err != nil || job.WorkspaceID == "" {
		return webObservationWindow{}, false
	}
	w := webObservationWindow{Network: job.WorkspaceID, From: job.CreatedAt.UTC()}
	if terminalWebJob(job.State) {
		for _, phase := range job.Phases {
			if phase.FinishedAt.After(w.To) {
				w.To = phase.FinishedAt.UTC()
			}
		}
		if !w.To.After(w.From) {
			// No recorded end: the interval cannot be bounded, so it is
			// neither shown as archived nor deleted.
			return webObservationWindow{}, false
		}
	}
	return w, true
}

// withObservations adds the archived window to a detached run copy.
func (s *WebHistory) withObservations(run WebRun) WebRun {
	if s.observations == nil {
		return run
	}
	w, ok := s.windowOf(run)
	if !ok {
		return run
	}
	summary := map[string]any{}
	for k, v := range run.Summary {
		summary[k] = v
	}
	window := map[string]any{"networkId": w.Network, "from": w.From}
	if !w.To.IsZero() {
		window["to"] = w.To
	}
	summary["observations"] = window
	missing := []string{}
	if list, ok := run.Summary["missingDimensions"].([]any); ok {
		for _, item := range list {
			if name, _ := item.(string); name != "" && name != "metrics" && name != "logs" {
				missing = append(missing, name)
			}
		}
	}
	summary["missingDimensions"] = missing
	run.Summary = summary
	return run
}

// deleteObservations removes the run-owned archive before the run itself. Other
// runs and running jobs on the same network protect what they cover.
func (s *WebHistory) deleteObservations(id string, run WebRun) error {
	if s.observations == nil {
		return nil
	}
	w, ok := s.windowOf(run)
	if !ok {
		return nil
	}
	var protected []webObservationWindow
	for other, capture := range s.state.Captures {
		if other == id || capture.Run.JobID == run.JobID {
			continue // Records of the same execution do not protect each other.
		}
		if p, ok := s.windowOf(capture.Run); ok && p.Network == w.Network {
			protected = append(protected, p)
		}
	}
	for _, job := range s.jobs.List() {
		if job.WorkspaceID == w.Network && !terminalWebJob(job.State) && job.ID != run.JobID {
			protected = append(protected, webObservationWindow{Network: w.Network, From: job.CreatedAt.UTC()})
		}
	}
	if err := s.observations.deleteWindow(id, w, protected); err != nil {
		return fmt.Errorf("%w: %v", ErrHistoryObservationsIncomplete, err)
	}
	return nil
}
