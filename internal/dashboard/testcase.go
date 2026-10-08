package dashboard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/0xmhha/chainbench/internal/app"
)

// WithTestCases exposes read-only validation/import operations through the
// existing account provider. Documents remain in the browser until explicitly
// saved by a document-management surface.
func WithTestCases(authenticate DeploymentAuthenticator) Option {
	return func(s *Server) {
		secure := func(fn http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				if authenticate == nil {
					http.Error(w, "login required", http.StatusUnauthorized)
					return
				}
				actor, err := authenticate(r)
				if err != nil || actor.ID == "" {
					http.Error(w, "login required", http.StatusUnauthorized)
					return
				}
				if r.Method != "GET" && actor.Role == "viewer" {
					http.Error(w, "read-only account", http.StatusForbidden)
					return
				}
				if origin := r.Header.Get("Origin"); origin != "" {
					scheme := "http"
					if r.TLS != nil {
						scheme = "https"
					}
					if origin != scheme+"://"+r.Host {
						http.Error(w, "origin rejected", http.StatusForbidden)
						return
					}
				}
				fn(w, r)
			}
		}
		s.mux.HandleFunc("GET /api/v1/vocabulary", secure(func(w http.ResponseWriter, r *http.Request) {
			v, _, err := app.DSLContract()
			deploymentResponse(w, v, err, 200)
		}))
		s.mux.HandleFunc("GET /api/v1/contracts/dsl", secure(func(w http.ResponseWriter, r *http.Request) {
			_, schema, err := app.DSLContract()
			deploymentResponse(w, schema, err, 200)
		}))
		s.mux.HandleFunc("POST /api/v1/test-cases/import", secure(func(w http.ResponseWriter, r *http.Request) {
			var in app.TestCaseInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			out, err := app.PrepareTestCase(in)
			if err != nil {
				deploymentJSON(w, 422, map[string]any{"valid": false, "errors": []map[string]string{{"path": "/content", "code": "invalid", "message": err.Error()}}})
				return
			}
			deploymentJSON(w, 200, out)
		}))
		previous := s.validateDocument
		s.validateDocument = func(w http.ResponseWriter, r *http.Request) {
			raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
			if err != nil {
				http.Error(w, "invalid document input", 400)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
			var head struct {
				Kind string `json:"kind"`
			}
			_ = json.Unmarshal(raw, &head)
			if head.Kind != "case" && previous != nil {
				previous(w, r)
				return
			}
			secure(func(w http.ResponseWriter, r *http.Request) {
				var in struct {
					Kind            string          `json:"kind"`
					Name            string          `json:"name"`
					ContractVersion string          `json:"contractVersion"`
					Content         json.RawMessage `json:"content"`
					AssetRefs       []string        `json:"assetRefs"`
				}
				if !deploymentDecode(w, r, &in) {
					return
				}
				issues := []map[string]string{}
				if in.Kind != "case" || in.ContractVersion != "2" || in.Name == "" {
					issues = append(issues, map[string]string{"path": "/", "code": "unsupported", "message": "expected named case document with contractVersion 2"})
				} else if _, err := app.PrepareTestCase(app.TestCaseInput{Content: in.Content}); err != nil {
					issues = append(issues, map[string]string{"path": "/content", "code": "invalid", "message": err.Error()})
				}
				deploymentJSON(w, 200, map[string]any{"valid": len(issues) == 0, "contractVersion": "2", "errors": issues, "warnings": []string{}})
			})(w, r)
		}
	}
}
