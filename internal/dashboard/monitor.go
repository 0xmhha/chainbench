package dashboard

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

// WithWebMonitor serves archived node metrics and time-linked logs to every
// signed-in role. The collector itself runs outside request handling.
func WithWebMonitor(monitor *app.WebMonitor, authenticate DeploymentAuthenticator) Option {
	return func(s *Server) {
		route := func(pattern string, fn func(http.ResponseWriter, *http.Request, app.DeploymentActor, app.WebMonitorQuery)) {
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
				q, err := monitorQuery(r)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				fn(w, r, a, q)
			})
		}
		route("GET /api/v1/networks/{networkId}/metrics", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor, q app.WebMonitorQuery) {
			metrics, err := monitor.Metrics(a, r.PathValue("networkId"), q)
			monitorResponse(w, metrics, err)
		})
		route("GET /api/v1/nodes/{nodeId}/logs", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor, q app.WebMonitorQuery) {
			logs, err := monitor.Logs(a, r.PathValue("nodeId"), q)
			monitorResponse(w, logs, err)
		})
	}
}

func monitorQuery(r *http.Request) (app.WebMonitorQuery, error) {
	values := r.URL.Query()
	q := app.WebMonitorQuery{NodeID: values.Get("nodeId"), Metric: values.Get("metric"), RunID: values.Get("runId"), Cursor: values.Get("cursor")}
	for name, dst := range map[string]*time.Time{"from": &q.From, "to": &q.To} {
		if raw := values.Get(name); raw != "" {
			t, err := time.Parse(time.RFC3339Nano, raw)
			if err != nil {
				return q, errors.New(name + " must be an RFC 3339 time")
			}
			*dst = t
		}
	}
	if !q.From.IsZero() && !q.To.IsZero() && q.To.Before(q.From) {
		return q, errors.New("to precedes from")
	}
	if raw := values.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return q, errors.New("limit must be between 1 and 100")
		}
		q.Limit = n
	}
	return q, nil
}

func monitorResponse(w http.ResponseWriter, v any, err error) {
	if errors.Is(err, app.ErrWebMonitorQuery) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	deploymentResponse(w, v, err, http.StatusOK)
}
