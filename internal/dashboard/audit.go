package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

// newRequestID names one request in its response and in the audit trail.
func newRequestID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var auditIdentifier = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

// auditTarget names what a request acted on from the matched route and its
// own identifiers ("users/3f2a…"). A raw URL is never recorded because a
// client controls it; an identifier outside the server's id alphabet is
// recorded as redacted.
//
// pattern is the route the request matches. A request refused before
// routing has no values yet, so they are read from its path segment by
// segment against the pattern.
func auditTarget(r *http.Request, pattern string, redact func(string) string) string {
	_, path, ok := strings.Cut(pattern, " ")
	if !ok {
		path = pattern
	}
	path = strings.TrimPrefix(path, "/api/v1/")
	if path == "" || strings.HasPrefix(path, "/") {
		return ""
	}
	parts := strings.Split(path, "/")
	actual := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			value := r.PathValue(strings.Trim(part, "{}."))
			if value == "" && i < len(actual) {
				value = actual[i]
			}
			if !auditIdentifier.MatchString(value) {
				value = "redacted"
			}
			parts[i] = redact(value)
		}
	}
	return strings.Join(parts, "/")
}

// WithWebAudit serves the merged audit trail to administrators. Every store
// keeps its own records; nil stores are skipped.
func WithWebAudit(auth *app.WebAuth, deployments *app.DeploymentStore, jobs *app.WebJobs, history *app.WebHistory, manifests *app.ManifestStore) Option {
	return func(s *Server) {
		var sources []app.WebAuditSource
		if auth != nil {
			sources = append(sources, auth.AuditRecords)
		}
		if deployments != nil {
			sources = append(sources, deployments.AuditRecords)
		}
		if jobs != nil {
			sources = append(sources, jobs.AuditRecords)
		}
		if history != nil {
			sources = append(sources, history.AuditRecords)
		}
		if manifests != nil {
			sources = append(sources, manifests.AuditRecords)
		}
		authenticate := WebAuthenticator(auth)
		s.mux.HandleFunc("GET /api/v1/audit", func(w http.ResponseWriter, r *http.Request) {
			a, err := authenticate(r)
			if err != nil || a.ID == "" {
				http.Error(w, "login required", http.StatusUnauthorized)
				return
			}
			q := app.WebAuditQuery{Cursor: r.URL.Query().Get("cursor")}
			for key, at := range map[string]*time.Time{"from": &q.From, "to": &q.To} {
				if raw := r.URL.Query().Get(key); raw != "" {
					if *at, err = time.Parse(time.RFC3339, raw); err != nil {
						http.Error(w, key+" must be an RFC 3339 time", http.StatusBadRequest)
						return
					}
				}
			}
			if raw := r.URL.Query().Get("limit"); raw != "" {
				if q.Limit, err = strconv.Atoi(raw); err != nil {
					http.Error(w, "limit must be an integer", http.StatusBadRequest)
					return
				}
			}
			page, err := app.ReadWebAudit(a, q, sources...)
			deploymentResponse(w, page, err, http.StatusOK)
		})
	}
}
