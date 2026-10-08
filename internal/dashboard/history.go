package dashboard

import (
	"net/http"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

// WithWebHistory exposes persistent captured results with the current account
// identity. Legacy /api/sessions response shapes and source files stay intact.
func WithWebHistory(history *app.WebHistory, authenticate DeploymentAuthenticator) Option {
	return func(s *Server) {
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
				if err = history.Sync(r.Context()); err != nil {
					http.Error(w, "history capture unavailable", http.StatusInternalServerError)
					return
				}
				fn(w, r, a)
			})
		}
		route("GET /api/v1/history", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			q, err := historyQuery(r)
			if err != nil {
				http.Error(w, "invalid history query", 400)
				return
			}
			page, err := history.List(q)
			if err != nil {
				http.Error(w, "invalid history cursor or filters", 400)
				return
			}
			deploymentJSON(w, 200, page)
		})
		route("GET /api/v1/history/{runId}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			run, err := history.Get(r.PathValue("runId"))
			deploymentResponse(w, run, err, 200)
		})
		route("POST /api/v1/history/compare", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in struct {
				RunIDs []string `json:"runIds"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			comparison, err := history.Compare(in.RunIDs)
			deploymentResponse(w, comparison, err, 200)
		})
		route("GET /api/v1/history/{runId}/export", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			b, err := history.Export(r.PathValue("runId"))
			if err != nil {
				deploymentError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="chainbench-result.json"`)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			_, _ = w.Write(b)
		})
		route("DELETE /api/v1/history/{runId}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			if err := history.Delete(a, r.PathValue("runId")); err != nil {
				deploymentError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func historyQuery(r *http.Request) (app.WebHistoryQuery, error) {
	v := r.URL.Query()
	q := app.WebHistoryQuery{Search: v.Get("search"), Chain: v.Get("chain"), WorkspaceID: v.Get("workspaceId"), ActorID: v.Get("actorId"), State: v.Get("state"), CaseID: v.Get("caseId"), Cursor: v.Get("cursor"), Limit: 50}
	var err error
	if v.Get("limit") != "" {
		q.Limit, err = strconv.Atoi(v.Get("limit"))
		if err != nil {
			return q, err
		}
	}
	if v.Get("from") != "" {
		q.From, err = time.Parse(time.RFC3339Nano, v.Get("from"))
		if err != nil {
			return q, err
		}
	}
	if v.Get("to") != "" {
		q.To, err = time.Parse(time.RFC3339Nano, v.Get("to"))
	}
	return q, err
}
