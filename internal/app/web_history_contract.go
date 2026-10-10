package app

import (
	"bytes"
	"encoding/json"
	"time"
)

// WebRun indexes a captured engine result or a durable job. Empty metadata is
// explicitly explained in Summary; it is never inferred from a directory name.
type WebRun struct {
	ID           string            `json:"id"`
	JobID        string            `json:"jobId,omitempty"`
	WorkspaceID  string            `json:"workspaceId"`
	Chain        string            `json:"chain"`
	ActorID      string            `json:"actorId,omitempty"`
	State        string            `json:"state"`
	StartedAt    string            `json:"startedAt,omitempty"`
	SessionRefs  []string          `json:"sessionRefs"`
	Fingerprints map[string]string `json:"fingerprints"`
	Summary      map[string]any    `json:"summary"`
	ArtifactRefs []string          `json:"artifactRefs"`
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
