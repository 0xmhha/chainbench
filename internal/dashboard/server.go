// Package dashboard is the HTTP backend for the chainbench dashboard
// (requirement #19). It streams the obs event bus to browsers over SSE and
// exposes run state as JSON, so a UI can show the three phases
// (setup/verify/test) live. With an artifact root it also serves completed-run
// session artifacts from disk (session verdict and chainstate history) under
// /api/sessions. The realtime contract is SSE (one-way event stream); the served
// page is the built Svelte SPA (decision D5), with the interim build-free page
// kept at /legacy as a no-JS fallback.
package dashboard

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

//go:embed index.html
var indexHTML []byte

// Server serves the dashboard API and page over one http.Handler.
type Server struct {
	bus              *collector.Bus
	store            collector.Store
	artifactRoot     string
	jobs             *app.WebJobs
	mux              *http.ServeMux
	validateDocument http.HandlerFunc
	security         http.HandlerFunc
	legacySecurity   http.HandlerFunc
	authorizeStream  func(*http.Request) bool
	publisherToken   string
	publisherAudit   func(string, string, int) error
	publisherRedact  func(string) string
}

// Option configures a Server.
type Option func(*Server)

// WithArtifactRoot points the session-artifact API at the directory holding
// `.chainbench` session directories, so completed runs can be queried from disk.
func WithArtifactRoot(root string) Option {
	return func(s *Server) { s.artifactRoot = root }
}

// NewServer wires the routes. store may be nil (runs API returns []). Without
// WithArtifactRoot the session-artifact API returns empty results.
func NewServer(bus *collector.Bus, store collector.Store, opts ...Option) *Server {
	s := &Server{bus: bus, store: store, mux: http.NewServeMux()}
	for _, opt := range opts {
		opt(s)
	}
	if s.validateDocument != nil {
		s.mux.HandleFunc("POST /api/v1/documents/validate", s.validateDocument)
	}
	// The built Svelte SPA (decision D5) is served at the root; the more specific
	// API/stream routes below take precedence over this catch-all. The interim
	// build-free page remains available at /legacy as a no-JS fallback.
	s.mux.Handle("GET /", spaHandler())
	s.mux.HandleFunc("GET /legacy", s.handleLegacy)
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /events", s.handleEvents)
	s.mux.HandleFunc("GET /api/runs", s.handleRuns)
	s.mux.HandleFunc("POST /api/events", s.handlePublish)
	s.mux.HandleFunc("GET /api/sessions", s.handleSessions)
	s.mux.HandleFunc("GET /api/sessions/{id}", s.handleSession)
	s.mux.HandleFunc("GET /api/sessions/{id}/chainstate", s.handleSessionChainstate)
	// A run lives under the directory of the process that produced it, so its
	// id is two segments. The one-segment routes stay for a root written by an
	// earlier build, whose runs sit directly under it.
	s.mux.HandleFunc("GET /api/sessions/{proc}/{id}", s.handleSession)
	s.mux.HandleFunc("GET /api/sessions/{proc}/{id}/chainstate", s.handleSessionChainstate)
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.servePublisher(w, r) {
		return
	}
	if s.security != nil {
		s.security(w, r)
		return
	}
	if s.legacySecurity != nil {
		s.legacySecurity(w, r)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// handleLegacy serves the interim build-free page as a no-JS fallback at /legacy.
func (s *Server) handleLegacy(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

// handleEvents streams bus events as Server-Sent Events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush() // send headers so the client knows it is connected

	sub, unsubscribe := s.bus.SubscribeWithCancel()
	defer unsubscribe()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if s.authorizeStream != nil && !s.authorizeStream(r) {
				return
			}
			metadata, _ := json.Marshal(map[string]any{"busDeliveryDropsTotal": s.bus.Dropped(), "observedAt": time.Now().UTC()})
			if _, err := fmt.Fprintf(w, "event: observation\ndata: %s\n\n", metadata); err != nil {
				return
			}
			flusher.Flush()
		case e, ok := <-sub:
			if !ok {
				return
			}
			if s.authorizeStream != nil && !s.authorizeStream(r) {
				return
			}
			data, err := json.Marshal(e)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleRuns returns the stored run records as JSON.
func (s *Server) handleRuns(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	runs := []collector.RunRecord{}
	if s.store != nil {
		runs = s.store.ListRuns()
	}
	_ = json.NewEncoder(w).Encode(runs)
}

// handlePublish accepts an collector.Event and publishes it to the bus, letting
// external processes (a CLI run, a remote agent) feed the dashboard.
func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	s.handlePublishSanitized(w, r, func(text string) string { return text })
}

func (s *Server) handlePublishSanitized(w http.ResponseWriter, r *http.Request, sanitize func(string) string) {
	var e collector.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&e); err != nil {
		http.Error(w, "bad event: "+err.Error(), http.StatusBadRequest)
		return
	}
	raw, err := json.Marshal(e)
	if err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal([]byte(sanitize(string(raw))), &e); err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}
	s.bus.Publish(e)
	w.WriteHeader(http.StatusAccepted)
}
