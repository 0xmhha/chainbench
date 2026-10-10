package app

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const webJobObservationBuffer = 128

type WebJobSnapshot struct {
	Version    uint64       `json:"version"`
	Cursor     string       `json:"cursor"`
	Jobs       []WebJob     `json:"jobs"`
	Networks   []WebNetwork `json:"networks"`
	ObservedAt time.Time    `json:"observedAt"`
}

type WebJobChange struct {
	Type       string    `json:"type"`
	Version    uint64    `json:"version"`
	Cursor     string    `json:"cursor"`
	Job        *WebJob   `json:"job,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	ObservedAt time.Time `json:"observedAt"`
}

type webJobFeed struct {
	instance    string
	journal     []WebJobChange
	subscribers map[chan WebJobChange]struct{}
	dropped     uint64
}

func newWebJobFeed() *webJobFeed {
	return &webJobFeed{instance: deploymentID(), subscribers: map[chan WebJobChange]struct{}{}}
}

func (s *WebJobs) streamCursor(version uint64) string {
	return s.feed.instance + ":" + strconv.FormatUint(version, 10)
}

func detachedWebJobChange(event WebJobChange) WebJobChange {
	if event.Job != nil {
		job := detachedWebJob(*event.Job)
		event.Job = &job
	}
	return event
}

// recordJobChange runs under the same lock as the durable commit. Failed writes
// never become stream updates, and a slow observer never delays the executor.
func (s *WebJobs) recordJobChange(id string) {
	if s.feed == nil {
		return
	}
	version := uint64(len(s.state.Audit))
	event := WebJobChange{Type: "version_changed", Version: version, Cursor: s.streamCursor(version), ObservedAt: time.Now().UTC()}
	if job, exists := s.state.Jobs[id]; exists {
		copy := detachedWebJob(job)
		event.Job = &copy
		event.Type = "job_changed"
	}
	s.feed.journal = append(s.feed.journal, event)
	if len(s.feed.journal) > webJobObservationBuffer {
		s.feed.journal = append([]WebJobChange(nil), s.feed.journal[len(s.feed.journal)-webJobObservationBuffer:]...)
	}
	for subscriber := range s.feed.subscribers {
		select {
		case subscriber <- detachedWebJobChange(event):
		default:
			s.feed.dropped++
		}
	}
}

// StreamPosition identifies a process-local replay generation and the durable
// committed job version. A restart retains the version but starts a new cursor.
func (s *WebJobs) StreamPosition() (uint64, string, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	version := uint64(len(s.state.Audit))
	return version, s.streamCursor(version), s.feed.dropped
}

// Snapshot pairs a detached job view with owned cached network records. Retry
// if a concurrent commit crossed the read; cached PIDs never become live proof.
func (s *WebJobs) Snapshot(ctx context.Context) (WebJobSnapshot, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return WebJobSnapshot{}, err
		}
		s.mu.Lock()
		version := uint64(len(s.state.Audit))
		snapshot := WebJobSnapshot{Version: version, Cursor: s.streamCursor(version), Jobs: []WebJob{}, Networks: []WebNetwork{}, ObservedAt: time.Now().UTC()}
		for _, job := range s.state.Jobs {
			snapshot.Jobs = append(snapshot.Jobs, detachedWebJob(job))
		}
		s.mu.Unlock()
		sort.Slice(snapshot.Jobs, func(i, j int) bool {
			if snapshot.Jobs[i].CreatedAt.Equal(snapshot.Jobs[j].CreatedAt) {
				return snapshot.Jobs[i].ID < snapshot.Jobs[j].ID
			}
			return snapshot.Jobs[i].CreatedAt.After(snapshot.Jobs[j].CreatedAt)
		})
		networks, err := s.Networks(ctx)
		if err != nil {
			return WebJobSnapshot{}, err
		}
		// Detach cached slices before stripping control suggestions. Observing
		// state must not mutate the engine's records or another reader's view.
		networks = append([]WebNetwork(nil), networks...)
		for i := range networks {
			networks[i].Nodes = append([]WebNode{}, networks[i].Nodes...)
			for n := range networks[i].Nodes {
				networks[i].Nodes[n].SupportedControls = []string{}
			}
		}
		if networks == nil {
			networks = []WebNetwork{}
		}
		snapshot.Networks = networks
		s.mu.Lock()
		consistent := version == uint64(len(s.state.Audit))
		s.mu.Unlock()
		if consistent {
			return snapshot, nil
		}
	}
	return WebJobSnapshot{}, ErrDeploymentConflict
}

// SubscribeChanges replays a bounded committed suffix or explicitly asks the
// client for a new snapshot. Subscription and commit share one lock, so no
// event can fall between the replay boundary and live delivery.
func (s *WebJobs) SubscribeChanges(cursor string) (<-chan WebJobChange, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.feed.subscribers) >= 64 {
		return nil, nil, ErrDeploymentConflict
	}
	channel := make(chan WebJobChange, webJobObservationBuffer)
	version := uint64(len(s.state.Audit))
	instance, raw, ok := strings.Cut(cursor, ":")
	requested, err := strconv.ParseUint(raw, 10, 64)
	reason := ""
	switch {
	case cursor == "":
		reason = "initial_snapshot_required"
	case !ok || err != nil || raw != strconv.FormatUint(requested, 10):
		reason = "cursor_invalid"
	case instance != s.feed.instance:
		reason = "server_restarted"
	case requested > version:
		reason = "cursor_invalid"
	case requested < version && (len(s.feed.journal) == 0 || requested+1 < s.feed.journal[0].Version):
		reason = "cursor_expired"
	}
	if reason != "" {
		channel <- WebJobChange{Type: "resync_required", Version: version, Cursor: s.streamCursor(version), Reason: reason, ObservedAt: time.Now().UTC()}
	} else {
		for _, event := range s.feed.journal {
			if event.Version > requested {
				channel <- detachedWebJobChange(event)
			}
		}
	}
	s.feed.subscribers[channel] = struct{}{}
	var once sync.Once
	return channel, func() {
		once.Do(func() { s.mu.Lock(); defer s.mu.Unlock(); delete(s.feed.subscribers, channel); close(channel) })
	}, nil
}
