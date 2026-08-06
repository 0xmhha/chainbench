// Package dashboard is the HTTP backend for the chainbench dashboard
// (requirement #19). It streams the obs event bus to browsers over SSE and
// exposes stored run state as JSON, so a UI can show the three phases
// (setup/verify/test) live. The realtime contract is SSE (one-way event
// stream); the served page is the built Svelte SPA (decision D5), with the
// interim build-free page kept at /legacy as a no-JS fallback
// (docs/CHAINBENCH_GO_REDESIGN.md §8.2).
package dashboard

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/0xmhha/chainbench/internal/core/obs"
)

//go:embed index.html
var indexHTML []byte

// Server serves the dashboard API and page over one http.Handler.
type Server struct {
	bus   *obs.Bus
	store obs.Store
	mux   *http.ServeMux
}

// NewServer wires the routes. store may be nil (runs API returns []).
func NewServer(bus *obs.Bus, store obs.Store) *Server {
	s := &Server{bus: bus, store: store, mux: http.NewServeMux()}
	// The built Svelte SPA (decision D5) is served at the root; the more specific
	// API/stream routes below take precedence over this catch-all. The interim
	// build-free page remains available at /legacy as a no-JS fallback.
	s.mux.Handle("GET /", spaHandler())
	s.mux.HandleFunc("GET /legacy", s.handleLegacy)
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /events", s.handleEvents)
	s.mux.HandleFunc("GET /api/runs", s.handleRuns)
	s.mux.HandleFunc("POST /api/events", s.handlePublish)
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

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

	sub := s.bus.Subscribe()
	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-sub:
			if !ok {
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
	runs := []obs.RunRecord{}
	if s.store != nil {
		runs = s.store.ListRuns()
	}
	_ = json.NewEncoder(w).Encode(runs)
}

// handlePublish accepts an obs.Event and publishes it to the bus, letting
// external processes (a CLI run, a remote agent) feed the dashboard.
func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	var e obs.Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "bad event: "+err.Error(), http.StatusBadRequest)
		return
	}
	s.bus.Publish(e)
	w.WriteHeader(http.StatusAccepted)
}
