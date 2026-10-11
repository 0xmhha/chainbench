package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// WebAuditRecord is one entry of the audit trail. Each store keeps its own
// append-only records; this is the shape an administrator reads them in.
// Result is an HTTP status for a request, or the outcome a store recorded.
type WebAuditRecord struct {
	ID        string    `json:"id"`
	ActorID   string    `json:"actorId"`
	Operation string    `json:"operation"`
	Target    string    `json:"target"`
	RequestID string    `json:"requestId,omitempty"`
	JobID     string    `json:"jobId,omitempty"`
	Result    string    `json:"result"`
	Time      time.Time `json:"time"`
}

// WebAuditQuery selects a time range and a page, newest first.
type WebAuditQuery struct {
	From, To time.Time
	Cursor   string
	Limit    int
}

type WebAuditPage struct {
	Items      []WebAuditRecord `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

// WebAuditSource reads one store's records.
type WebAuditSource func() ([]WebAuditRecord, error)

// ReadWebAudit merges every store's records for an administrator. History
// deletion removes runs, never these records.
func ReadWebAudit(a DeploymentActor, q WebAuditQuery, sources ...WebAuditSource) (WebAuditPage, error) {
	if a.ID == "" || a.Role != "admin" {
		return WebAuditPage{}, ErrDeploymentForbidden
	}
	limit := q.Limit
	if limit == 0 {
		limit = 25
	}
	if limit < 1 || limit > 100 {
		return WebAuditPage{}, errors.New("limit must be between 1 and 100")
	}
	offset := 0
	if q.Cursor != "" {
		n, err := strconv.Atoi(q.Cursor)
		if err != nil || n < 0 {
			return WebAuditPage{}, errors.New("invalid cursor")
		}
		offset = n
	}
	all := []WebAuditRecord{}
	for _, source := range sources {
		if source == nil {
			continue
		}
		records, err := source()
		if err != nil {
			return WebAuditPage{}, err
		}
		for _, r := range records {
			if !q.From.IsZero() && r.Time.Before(q.From) || !q.To.IsZero() && !r.Time.Before(q.To) {
				continue
			}
			all = append(all, r)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Time.After(all[j].Time) })
	page := WebAuditPage{Items: []WebAuditRecord{}}
	if offset < len(all) {
		end := min(offset+limit, len(all))
		page.Items = all[offset:end]
		if end < len(all) {
			next := strconv.Itoa(end)
			page.NextCursor = &next
		}
	}
	return page, nil
}

// webAuditID names a record by where it is kept and its position there, so
// the same record keeps its id across reads.
func webAuditID(source string, index int, r WebAuditRecord) string {
	return manifestHash(fmt.Appendf(nil, "%s\x00%d\x00%s\x00%s\x00%s", source, index, r.ActorID, r.Operation, r.Time.Format(time.RFC3339Nano)))[:24]
}

// webAuditLines reads a JSON-lines audit file, skipping nothing: a line that
// does not parse is an error, not a silent gap.
func webAuditLines(source string, raw []byte, convert func(map[string]any) WebAuditRecord) ([]WebAuditRecord, error) {
	out := []WebAuditRecord{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for i := 0; sc.Scan(); i++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("%s audit line %d: %w", source, i+1, err)
		}
		r := convert(m)
		r.ID = webAuditID(source, i, r)
		out = append(out, r)
	}
	return out, sc.Err()
}

func webAuditString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func webAuditTime(m map[string]any, keys ...string) time.Time {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func webAuditStatus(m map[string]any) string {
	switch v := m["status"].(type) {
	case float64:
		if v == 0 {
			return "accepted"
		}
		return strconv.Itoa(int(v))
	}
	return ""
}

// AuditRecords reads the request trail the security layer writes.
func (a *WebAuth) AuditRecords() ([]WebAuditRecord, error) {
	a.mu.Lock()
	raw, err := a.storage.ReadAudit()
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return webAuditLines("security", raw, func(m map[string]any) WebAuditRecord {
		return WebAuditRecord{ActorID: webAuditString(m, "actorId"), Operation: webAuditString(m, "operation"), Target: webAuditString(m, "target"), RequestID: webAuditString(m, "requestId"), Result: webAuditStatus(m), Time: webAuditTime(m, "time")}
	})
}

// AuditRecords reads the manifest and asset trail.
func (s *ManifestStore) AuditRecords() ([]WebAuditRecord, error) {
	s.mu.Lock()
	raw, err := s.files.ReadAudit()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return webAuditLines("manifest", raw, func(m map[string]any) WebAuditRecord {
		return WebAuditRecord{ActorID: webAuditString(m, "actorId"), Operation: webAuditString(m, "operation"), Target: webAuditString(m, "target"), Result: webAuditStatus(m), Time: webAuditTime(m, "at", "time")}
	})
}

// AuditRecords reads shared document and credential changes; each is kept
// only once it is committed.
func (s *DeploymentStore) AuditRecords() ([]WebAuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]WebAuditRecord, 0, len(s.state.Audit))
	for i, a := range s.state.Audit {
		r := WebAuditRecord{ActorID: a.Actor, Operation: a.Operation, Target: a.Target, Result: "committed", Time: a.Time}
		r.ID = webAuditID("deployment", i, r)
		out = append(out, r)
	}
	return out, nil
}

// AuditRecords reads job transitions, cancellations included.
func (s *WebJobs) AuditRecords() ([]WebAuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]WebAuditRecord, 0, len(s.state.Audit))
	for i, a := range s.state.Audit {
		r := WebAuditRecord{ActorID: a.ActorID, Operation: a.Operation, Target: "jobs/" + a.JobID, JobID: a.JobID, Result: "recorded", Time: a.Time}
		r.ID = webAuditID("jobs", i, r)
		out = append(out, r)
	}
	return out, nil
}

// AuditRecords reads history deletions and failed deletions.
func (s *WebHistory) AuditRecords() ([]WebAuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]WebAuditRecord, 0, len(s.state.Audit))
	for i, a := range s.state.Audit {
		result := "deleted"
		if a.Operation == "history.delete_failed" {
			result = "failed"
		}
		r := WebAuditRecord{ActorID: a.ActorID, Operation: a.Operation, Target: "history/" + a.RunID, Result: result, Time: a.Time}
		r.ID = webAuditID("history", i, r)
		out = append(out, r)
	}
	return out, nil
}
