package dashboard

import (
	"net/http"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

// WithWebJobs binds durable engine jobs to the same server-owned account identity.
func WithWebJobs(jobs *app.WebJobs, authenticate DeploymentAuthenticator) Option {
	return func(s *Server) {
		s.jobs = jobs
		route := func(pattern string, fn func(http.ResponseWriter, *http.Request, app.DeploymentActor)) {
			s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				if authenticate == nil {
					http.Error(w, "login required", http.StatusUnauthorized)
					return
				}
				a, err := authenticate(r)
				if err != nil || a.ID == "" {
					http.Error(w, "login required", http.StatusUnauthorized)
					return
				}
				fn(w, r, a)
			})
		}
		route("GET /api/v1/networks", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			networks, err := jobs.Networks(r.Context())
			deploymentResponse(w, map[string]any{"items": networks, "nextCursor": nil}, err, 200)
		})
		route("POST /api/v1/plans", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.WebPlanInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			plan, err := jobs.Plan(r.Context(), a, in)
			deploymentResponse(w, plan, err, http.StatusCreated)
		})
		route("GET /api/v1/plans/{planId}/conflicts", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			conflicts, err := jobs.PlanConflicts(a, r.PathValue("planId"))
			deploymentResponse(w, map[string]any{"items": conflicts, "observedAt": time.Now().UTC()}, err, http.StatusOK)
		})
		route("POST /api/v1/jobs", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in struct {
				PlanID string `json:"planId"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			job, err := jobs.Start(r.Context(), a, in.PlanID, r.Header.Get("Idempotency-Key"))
			deploymentResponse(w, job, err, http.StatusAccepted)
		})
		route("GET /api/v1/jobs", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			query := r.URL.Query()
			limit := 50
			if raw := query.Get("limit"); raw != "" {
				v, err := strconv.Atoi(raw)
				if err != nil || v < 1 || v > 200 {
					http.Error(w, "invalid limit", 400)
					return
				}
				limit = v
			}
			cursor := query.Get("cursor")
			items := []app.WebJob{}
			next := ""
			past := cursor == ""
			found := past
			for _, job := range jobs.List() {
				if !past {
					if job.ID == cursor {
						past = true
						found = true
					}
					continue
				}
				if (query.Get("workspaceId") != "" && job.WorkspaceID != query.Get("workspaceId")) || (query.Get("state") != "" && job.State != query.Get("state")) || (query.Get("actorId") != "" && job.ActorID != query.Get("actorId")) {
					continue
				}
				if len(items) == limit {
					next = items[len(items)-1].ID
					break
				}
				items = append(items, job)
			}
			if !found {
				http.Error(w, "invalid cursor", 400)
				return
			}
			var nextCursor any
			if next != "" {
				nextCursor = next
			}
			deploymentJSON(w, 200, map[string]any{"items": items, "nextCursor": nextCursor})
		})
		route("GET /api/v1/jobs/{jobId}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			job, err := jobs.Get(r.PathValue("jobId"))
			deploymentResponse(w, job, err, 200)
		})
		route("POST /api/v1/jobs/{jobId}/cancel", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in struct {
				Message string `json:"message"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			job, err := jobs.Cancel(a, r.PathValue("jobId"))
			deploymentResponse(w, job, err, http.StatusAccepted)
		})
	}
}
