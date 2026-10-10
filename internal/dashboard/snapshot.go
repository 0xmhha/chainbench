package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

func (s *Server) handleJobSnapshot(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	for name, values := range query {
		if (name != "workspaceId" && name != "jobId") || len(values) != 1 {
			http.Error(w, "invalid snapshot scope", 400)
			return
		}
	}
	snapshot, err := s.jobs.Snapshot(r.Context())
	if err != nil {
		deploymentError(w, err)
		return
	}
	jobs := []app.WebJob{}
	found := query.Get("jobId") == ""
	for _, job := range snapshot.Jobs {
		if query.Get("jobId") != "" && job.ID != query.Get("jobId") {
			continue
		}
		if query.Get("workspaceId") != "" && job.WorkspaceID != query.Get("workspaceId") {
			continue
		}
		found = true
		jobs = append(jobs, job)
	}
	if !found {
		deploymentError(w, app.ErrDeploymentNotFound)
		return
	}
	snapshot.Jobs = jobs
	workspaceID := query.Get("workspaceId")
	if query.Get("jobId") != "" {
		workspaceID = jobs[0].WorkspaceID
	}
	networks := []app.WebNetwork{}
	for _, network := range snapshot.Networks {
		if workspaceID == "" || network.WorkspaceID == workspaceID {
			networks = append(networks, network)
		}
	}
	snapshot.Networks = networks
	drops := uint64(0)
	if s.bus != nil {
		drops = s.bus.Dropped()
	}
	_, _, jobDrops := s.jobs.StreamPosition()
	deploymentJSON(w, 200, struct {
		app.WebJobSnapshot
		BusDeliveryDropsTotal uint64 `json:"busDeliveryDropsTotal"`
		JobDeliveryDropsTotal uint64 `json:"jobDeliveryDropsTotal"`
	}{snapshot, drops, jobDrops})
}

func (s *Server) handleJobEvents(w http.ResponseWriter, r *http.Request, authenticate DeploymentAuthenticator) {
	query := r.URL.Query()
	for name, values := range query {
		if (name != "cursor" && name != "workspaceId" && name != "jobId") || len(values) != 1 {
			http.Error(w, "invalid stream scope", 400)
			return
		}
	}
	workspaceID, jobID := query.Get("workspaceId"), query.Get("jobId")
	if jobID != "" {
		job, err := s.jobs.Get(jobID)
		if err != nil {
			deploymentError(w, err)
			return
		}
		if workspaceID != "" && workspaceID != job.WorkspaceID {
			deploymentError(w, app.ErrDeploymentNotFound)
			return
		}
		workspaceID = job.WorkspaceID
	}
	cursor := r.Header.Get("Last-Event-ID")
	if cursor == "" {
		cursor = r.URL.Query().Get("cursor")
	}
	changes, unsubscribe, err := s.jobs.SubscribeChanges(cursor)
	if err != nil {
		deploymentError(w, err)
		return
	}
	defer unsubscribe()
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	authorized := func() bool { actor, err := authenticate(r); return err == nil && actor.ID != "" }
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-changes:
			if !ok || !authorized() {
				return
			}
			// Keep the global revision sequence contiguous without exposing
			// unrelated jobs to a scoped observer.
			if event.Job != nil && ((workspaceID != "" && event.Job.WorkspaceID != workspaceID) || (jobID != "" && event.Job.ID != jobID)) {
				event.Job = nil
				event.Type = "version_changed"
			}
			raw, err := json.Marshal(event)
			if err != nil {
				return
			}
			if _, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.Cursor, event.Type, raw); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if !authorized() {
				return
			}
			version, position, drops := s.jobs.StreamPosition()
			raw, _ := json.Marshal(map[string]any{"version": version, "cursor": position, "jobDeliveryDropsTotal": drops, "observedAt": time.Now().UTC()})
			// Heartbeats must not advance Last-Event-ID past changes the client missed.
			if _, err = fmt.Fprintf(w, "event: observation\ndata: %s\n\n", raw); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
