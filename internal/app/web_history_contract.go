package app

import (
	"bytes"
	"encoding/json"
	"time"
)

// WebRun indexes a captured engine result or a durable job. Empty metadata is
// explicitly explained in Summary; it is never inferred from a directory name.
type WebRun struct {
	ID          string `json:"id"`
	JobID       string `json:"jobId,omitempty"`
	WorkspaceID string `json:"workspaceId"`
	Chain       string `json:"chain"`
	ActorID     string `json:"actorId,omitempty"`
	State       string `json:"state"`
	StartedAt   string `json:"startedAt,omitempty"`
	FinishedAt  string `json:"finishedAt,omitempty"`
	// Coverage is the archive coverage of the run's window, in detail reads.
	Coverage     *WebCoverage      `json:"coverage,omitempty"`
	SessionRefs  []string          `json:"sessionRefs"`
	Fingerprints map[string]string `json:"fingerprints"`
	Summary      map[string]any    `json:"summary"`
	ArtifactRefs []string          `json:"artifactRefs"`
	// Tests are the session's tests with their result files read, in detail
	// reads of engine-session runs only.
	Tests []WebRunTest `json:"tests,omitempty"`
}

type WebHistoryQuery struct {
	Search, Chain, WorkspaceID, ActorID, State, CaseID, Cursor string
	From, To                                                   time.Time
	Limit                                                      int
}

type WebRunPage struct {
	Items      []WebRun `json:"items"`
	NextCursor *string  `json:"nextCursor"`
}

type WebComparison struct {
	RunIDs      []string         `json:"runIds"`
	Comparable  bool             `json:"comparable"`
	Limitations []string         `json:"limitations"`
	Results     []map[string]any `json:"results"`
	// Durations is each run's recorded length in seconds; null when the run
	// has no recorded end.
	Durations map[string]*float64 `json:"durations"`
	// Metrics compares the node metrics archived in each run's window.
	Metrics []WebMetricComparison `json:"metrics"`
}

// WebMetricComparison lines up one metric across runs. A run without samples
// has none listed rather than zeros.
type WebMetricComparison struct {
	Name   string                      `json:"name"`
	Unit   string                      `json:"unit,omitempty"`
	Source string                      `json:"source"`
	Values map[string]WebMetricSummary `json:"values"`
}

type WebMetricSummary struct {
	Samples int      `json:"samples"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Last    *float64 `json:"last,omitempty"`
}

type webHistoryCapture struct {
	Run   WebRun            `json:"run"`
	Files map[string]string `json:"files"`
}
type webHistoryAudit struct {
	ActorID   string    `json:"actorId"`
	RunID     string    `json:"runId"`
	Operation string    `json:"operation"`
	Time      time.Time `json:"time"`
}
type webHistoryState struct {
	Captures map[string]webHistoryCapture `json:"captures"`
	Deleted  map[string]bool              `json:"deleted"`
	Audit    []webHistoryAudit            `json:"audit"`
}

func detachedHistory(c webHistoryCapture) webHistoryCapture {
	b, _ := json.Marshal(c)
	var out webHistoryCapture
	_ = decodeHistoryJSON(b, &out)
	return out
}

func decodeHistoryJSON(b []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	return decoder.Decode(out)
}
