package dashboard

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// handleSessions lists the session IDs available under the artifact root.
func (s *Server) handleSessions(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ids := []string{}
	if s.artifactRoot != "" {
		found, err := session.List(s.artifactRoot)
		if err != nil {
			http.Error(w, "list sessions: "+err.Error(), http.StatusInternalServerError)
			return
		}
		ids = append(ids, found...)
	}
	_ = json.NewEncoder(w).Encode(ids)
}

// handleSession serves one session's session.json verdict.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validSessionID(id) {
		http.Error(w, "bad session id", http.StatusBadRequest)
		return
	}
	if s.artifactRoot == "" {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(session.SessionFilePath(s.artifactRoot, id))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// handleSessionChainstate returns the chainstate samples persisted across the
// session's environments as a JSON array, so the dashboard can replay a run.
func (s *Server) handleSessionChainstate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validSessionID(id) {
		http.Error(w, "bad session id", http.StatusBadRequest)
		return
	}
	samples := []json.RawMessage{}
	if s.artifactRoot != "" {
		paths, err := session.ChainstatePaths(s.artifactRoot, id)
		if err != nil {
			http.Error(w, "chainstate: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, p := range paths {
			samples = append(samples, readJSONL(p)...)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(samples)
}

// readJSONL reads a jsonl file into raw JSON messages, skipping blank lines. A
// missing or unreadable file yields no samples (best-effort, never fatal).
func readJSONL(path string) []json.RawMessage {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	var out []json.RawMessage
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		out = append(out, json.RawMessage(line))
	}
	return out
}

// validSessionID rejects empty ids and anything that could escape the artifact
// root; a session id is a single path segment.
func validSessionID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	return !strings.ContainsAny(id, `/\`)
}
