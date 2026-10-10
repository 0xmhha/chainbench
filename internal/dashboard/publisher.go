package dashboard

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
)

// WithPublisherToken grants only event ingestion, separately from browser cookies.
func WithPublisherToken(token string) Option {
	return func(s *Server) { s.publisherToken = token }
}

func (s *Server) servePublisher(w http.ResponseWriter, r *http.Request) bool {
	if s.publisherToken == "" || r.URL.Path != "/api/events" || r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		return false
	}
	sanitize := func(text string) string {
		text = strings.ReplaceAll(text, s.publisherToken, "[REDACTED]")
		if s.publisherRedact != nil {
			return s.publisherRedact(text)
		}
		return app.RedactWebText(text)
	}
	response := &secureResponse{ResponseWriter: w, status: http.StatusOK, redact: sanitize}
	w.Header().Set("Cache-Control", "no-store")
	if s.publisherAudit != nil {
		defer func() { _ = s.publisherAudit("publisher", "web.publisher", response.status) }()
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), []byte(s.publisherToken)) != 1 {
		http.Error(response, "publisher authentication failed", http.StatusUnauthorized)
		return true
	}
	if s.publisherAudit != nil {
		if err := s.publisherAudit("publisher", "web.publisher.accepted", 0); err != nil {
			http.Error(response, "audit unavailable", http.StatusInternalServerError)
			return true
		}
	}
	// Sanitize before publication so no subscriber receives the producer token.
	s.handlePublishSanitized(response, r, sanitize)
	return true
}
