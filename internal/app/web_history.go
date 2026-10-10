package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

type historyStorage interface {
	Read() ([]byte, error)
	Write([]byte) error
}

// WebHistory owns redacted evidence copies, independently of live node paths.
// One server owns this store. Tombstones and audit survive deletion and restart.
type WebHistory struct {
	mu              sync.Mutex
	files           historyStorage
	state           webHistoryState
	artifactRoot    string
	webArtifactRoot string
	jobs            *WebJobs
	redact          func(string) string
	observations    *WebMonitor
}

func OpenWebHistory(root, artifactRoot string, jobs *WebJobs, redact func(string) string) (*WebHistory, error) {
	files, err := session.OpenHistoryStore(root)
	if err != nil {
		return nil, err
	}
	if redact == nil {
		redact = RedactWebText
	}
	s := &WebHistory{files: files, artifactRoot: artifactRoot, webArtifactRoot: webOwnedSessions(root), jobs: jobs, redact: redact, state: webHistoryState{Captures: map[string]webHistoryCapture{}, Deleted: map[string]bool{}, Audit: []webHistoryAudit{}}}
	b, err := files.Read()
	if err == nil {
		if err = decodeHistoryJSON(b, &s.state); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if s.state.Captures == nil || s.state.Deleted == nil {
		return nil, errors.New("invalid history snapshot")
	}
	for id, capture := range s.state.Captures {
		if capture.Run.ID != id || s.state.Deleted[id] || capture.Files == nil {
			return nil, errors.New("invalid captured result")
		}
	}
	return s, nil
}

func (s *WebHistory) commit(next webHistoryState) error {
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err = s.files.Write(b); err != nil {
		return err
	}
	s.state = next
	return nil
}
func (s *WebHistory) clone() webHistoryState {
	b, _ := json.Marshal(s.state)
	var next webHistoryState
	_ = decodeHistoryJSON(b, &next)
	return next
}
func historySessionID(ref string) string {
	digest := sha256.Sum256([]byte("session:" + ref))
	return "session-" + hex.EncodeToString(digest[:16])
}

// Sync captures new engine sessions and current job state. Old captures remain
// readable even if their source tree is later moved or removed.
func (s *WebHistory) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.clone()
	jobRuns, refs := s.jobHistory()
	for _, run := range jobRuns {
		if !next.Deleted[run.ID] {
			b, _ := json.Marshal(run)
			next.Captures[run.ID] = webHistoryCapture{run, map[string]string{"job.json": string(b)}}
		}
	}
	sources := []struct{ root, prefix string }{{s.webArtifactRoot, "web:"}}
	if s.artifactRoot != "" && filepath.Clean(s.artifactRoot) != filepath.Clean(s.webArtifactRoot) {
		sources = append(sources, struct{ root, prefix string }{s.artifactRoot, ""})
	}
	for _, source := range sources {
		ids, err := session.List(source.root)
		if err != nil {
			return err
		}
		for _, ref := range ids {
			if err = ctx.Err(); err != nil {
				return err
			}
			sourceRef := source.prefix + ref
			id := historySessionID(sourceRef)
			previous, found := next.Captures[id]
			if next.Deleted[id] || found && (source.prefix == "" || previous.Run.JobID != "" && terminalWebJob(previous.Run.State)) {
				continue
			}
			captured, err := session.Capture(source.root, ref)
			if err != nil {
				return errors.New("session evidence could not be safely captured")
			}
			capture := s.sessionHistory(id, sourceRef, captured, refs[sourceRef])
			next.Captures[id] = capture
		}
	}
	before, _ := json.Marshal(s.state)
	after, _ := json.Marshal(next)
	if string(before) == string(after) {
		return nil
	}
	return s.commit(next)
}

func (s *WebHistory) List(q WebHistoryQuery) (WebRunPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := WebRunPage{Items: []WebRun{}}
	if q.Limit == 0 {
		q.Limit = 50
	}
	if q.Limit < 1 || q.Limit > 200 || (!q.From.IsZero() && !q.To.IsZero() && q.From.After(q.To)) {
		return out, errors.New("invalid history query")
	}
	rows := []WebRun{}
	for _, capture := range s.state.Captures {
		run := capture.Run
		if historyMatches(run, q) {
			rows = append(rows, s.withObservations(detachedHistory(capture).Run))
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].StartedAt == rows[j].StartedAt {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].StartedAt > rows[j].StartedAt
	})
	start := 0
	if q.Cursor != "" {
		found := false
		for i, row := range rows {
			if row.ID == q.Cursor {
				start, found = i+1, true
				break
			}
		}
		if !found {
			return out, errors.New("cursor does not belong to this filtered result")
		}
	}
	end := min(start+q.Limit, len(rows))
	out.Items = append(out.Items, rows[start:end]...)
	if end < len(rows) {
		cursor := rows[end-1].ID
		out.NextCursor = &cursor
	}
	return out, nil
}

func historyMatches(run WebRun, q WebHistoryQuery) bool {
	if (q.Chain != "" && run.Chain != q.Chain) || (q.WorkspaceID != "" && run.WorkspaceID != q.WorkspaceID) || (q.ActorID != "" && run.ActorID != q.ActorID) || (q.State != "" && run.State != q.State) {
		return false
	}
	started, err := time.Parse(time.RFC3339Nano, run.StartedAt)
	if (!q.From.IsZero() && (err != nil || started.Before(q.From))) || (!q.To.IsZero() && (err != nil || started.After(q.To))) {
		return false
	}
	b, _ := json.Marshal(run)
	if q.Search != "" && !strings.Contains(strings.ToLower(string(b)), strings.ToLower(q.Search)) {
		return false
	}
	if q.CaseID != "" {
		for _, test := range historyTests(run) {
			if test.ID == q.CaseID {
				return true
			}
		}
		return false
	}
	return true
}

func (s *WebHistory) Get(id string) (WebRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	capture, found := s.state.Captures[id]
	if !found {
		return WebRun{}, ErrDeploymentNotFound
	}
	return s.withObservations(detachedHistory(capture).Run), nil
}

func (s *WebHistory) Export(id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	capture, found := s.state.Captures[id]
	if !found {
		return nil, ErrDeploymentNotFound
	}
	// Redact again at read time, including credentials registered after capture.
	b, err := json.Marshal(capture)
	if err != nil {
		return nil, err
	}
	return []byte(s.redact(string(b))), nil
}

func (s *WebHistory) Delete(a DeploymentActor, id string) error {
	if a.ID == "" || a.Role != "admin" {
		return ErrDeploymentForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	capture, found := s.state.Captures[id]
	if !found {
		return ErrDeploymentNotFound
	}
	if !terminalWebJob(capture.Run.State) {
		return ErrDeploymentConflict
	}
	if s.jobs != nil && capture.Run.JobID != "" {
		job, err := s.jobs.Get(capture.Run.JobID)
		if err != nil || !terminalWebJob(job.State) {
			return ErrDeploymentConflict
		}
	}
	if err := s.deleteObservations(id, capture.Run); err != nil {
		failed := s.clone()
		failed.Audit = append(failed.Audit, webHistoryAudit{a.ID, id, "history.delete_failed", time.Now().UTC()})
		return errors.Join(err, s.commit(failed))
	}
	next := s.clone()
	delete(next.Captures, id)
	next.Deleted[id] = true
	next.Audit = append(next.Audit, webHistoryAudit{a.ID, id, "history.deleted", time.Now().UTC()})
	// One atomic replacement removes the owned evidence and stores the tombstone
	// and audit together. Failure leaves every prior artifact and index intact.
	return s.commit(next)
}
