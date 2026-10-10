package dashboard

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
)

// DeploymentAuthenticator resolves a trusted account identity; request bodies cannot select users.
type DeploymentAuthenticator func(*http.Request) (app.DeploymentActor, error)

// WithDeployments enables shared configuration and private SSH overlays using the account provider.
func WithDeployments(store *app.DeploymentStore, authenticate DeploymentAuthenticator) Option {
	return func(s *Server) {
		s.legacySecurity = legacyWebSecurity(s, store, authenticate)
		route := func(pattern string, fn func(http.ResponseWriter, *http.Request, app.DeploymentActor)) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				if authenticate == nil {
					deploymentError(w, app.ErrDeploymentForbidden)
					return
				}
				actor, err := authenticate(r)
				if err != nil || actor.ID == "" {
					w.Header().Set("WWW-Authenticate", `Basic realm="chainbench", charset="UTF-8"`)
					http.Error(w, "login required", http.StatusUnauthorized)
					return
				}
				// Basic credentials are explicitly sent by the UI, never ambient cookies.
				// Reject cross-origin mutation even if a browser cached Basic credentials.
				if r.Method != "GET" {
					origin := r.Header.Get("Origin")
					scheme := "http"
					if r.TLS != nil {
						scheme = "https"
					}
					if origin != "" && origin != scheme+"://"+r.Host {
						deploymentError(w, app.ErrDeploymentForbidden)
						return
					}
				}
				fn(w, r, actor)
			}
			if pattern == "POST /api/v1/documents/validate" {
				previous := s.validateDocument
				s.validateDocument = func(w http.ResponseWriter, r *http.Request) {
					b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
					if err != nil {
						http.Error(w, "invalid document input", http.StatusBadRequest)
						return
					}
					r.Body = io.NopCloser(strings.NewReader(string(b)))
					var kind struct {
						Kind string `json:"kind"`
					}
					_ = json.Unmarshal(b, &kind)
					if previous != nil && kind.Kind == "chain-preset" {
						previous(w, r)
						return
					}
					handler(w, r)
				}
			} else {
				s.mux.HandleFunc(pattern, handler)
			}
		}
		route("GET /api/v1/contracts/deployment", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			deploymentJSON(w, 200, app.DeploymentContract())
		})
		route("POST /api/v1/documents/validate", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.DeploymentDocumentInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			err := store.ValidateDocument(in)
			issues := []map[string]string{}
			if err != nil {
				issues = append(issues, map[string]string{"path": "/content", "code": "invalid", "message": err.Error()})
			}
			deploymentJSON(w, 200, map[string]any{"valid": err == nil, "errors": issues, "contractVersion": "2", "warnings": []string{"SSH access requires a personal credential and actual target permission check."}})
		})
		route("GET /api/v1/deployment-account", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) { deploymentJSON(w, 200, a) })
		route("GET /api/v1/documents", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			deploymentJSON(w, 200, map[string]any{"items": store.Documents(r.URL.Query().Get("kind")), "nextCursor": nil})
		})
		route("POST /api/v1/documents/import", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.DocumentImportInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			v, err := store.PreviewImport(a, in)
			deploymentResponse(w, v, err, http.StatusOK)
		})
		route("POST /api/v1/documents/import/commit", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.DocumentImportCommit
			if !deploymentDecode(w, r, &in) {
				return
			}
			v, err := store.CommitImport(a, in)
			deploymentResponse(w, map[string]any{"items": v, "nextCursor": nil}, err, http.StatusCreated)
		})
		route("POST /api/v1/documents", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.DeploymentDocumentInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			d, err := store.SaveDocument(a, "", 0, in)
			deploymentResponse(w, d, err, 201)
		})
		route("GET /api/v1/documents/{id}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			d, err := deploymentReadDocument(store, r)
			deploymentResponse(w, d, err, 200)
		})
		route("PATCH /api/v1/documents/{id}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			rev, ok := deploymentRevision(w, r)
			if !ok {
				return
			}
			var in app.DeploymentDocumentInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			d, err := store.SaveDocument(a, r.PathValue("id"), rev, in)
			deploymentResponse(w, d, err, 200)
		})
		route("GET /api/v1/documents/{id}/export", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			d, err := deploymentReadDocument(store, r)
			if err != nil {
				deploymentError(w, err)
				return
			}
			b, err := app.ExportDeploymentDocument(d.DeploymentDocumentInput, r.URL.Query().Get("format"))
			if err != nil {
				deploymentError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(b)
		})
		route("GET /api/v1/documents/{id}/bundle", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			b, err := store.ExportDocumentBundle(a, r.PathValue("id"))
			if err != nil {
				deploymentError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="chainbench-document.bundle.json"`)
			_, _ = w.Write(b)
		})
		route("GET /api/v1/workspaces", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			deploymentJSON(w, 200, map[string]any{"items": store.Workspaces(), "nextCursor": nil})
		})
		route("POST /api/v1/workspaces", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.DeploymentWorkspaceInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			v, err := store.SaveWorkspace(a, "", 0, in)
			deploymentResponse(w, v, err, 201)
		})
		route("GET /api/v1/workspaces/{id}/export", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			b, err := store.ExportWorkspaceBundle(a, r.PathValue("id"))
			if err != nil {
				deploymentError(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="chainbench-workspace.bundle.json"`)
			_, _ = w.Write(b)
		})
		route("GET /api/v1/workspaces/{id}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			v, err := store.Workspace(r.PathValue("id"))
			deploymentResponse(w, v, err, 200)
		})
		route("PATCH /api/v1/workspaces/{id}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			rev, ok := deploymentRevision(w, r)
			if !ok {
				return
			}
			var in app.DeploymentWorkspaceInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			v, err := store.SaveWorkspace(a, r.PathValue("id"), rev, in)
			deploymentResponse(w, v, err, 200)
		})
		route("GET /api/v1/credentials", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			deploymentJSON(w, 200, map[string]any{"items": store.Credentials(a), "nextCursor": nil})
		})
		route("POST /api/v1/credentials", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.DeploymentCredentialInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			v, err := store.SaveCredential(a, in)
			deploymentResponse(w, v, err, 201)
		})
		route("GET /api/v1/credentials/{id}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			v, err := store.Credential(a, r.PathValue("id"))
			deploymentResponse(w, v, err, 200)
		})
		route("DELETE /api/v1/credentials/{id}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			if err := store.RevokeCredential(a, r.PathValue("id")); err != nil {
				deploymentError(w, err)
				return
			}
			if s.jobs != nil {
				if err := s.jobs.RevokeCredential(a.ID, r.PathValue("id")); err != nil {
					deploymentError(w, err)
					return
				}
			}
			w.WriteHeader(http.StatusNoContent)
		})
		route("POST /api/v1/credentials/{id}/check", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in struct {
				WorkspaceID string `json:"workspaceId"`
				ServerRef   string `json:"serverRef"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			v, err := store.CheckCredential(r.Context(), a, in.WorkspaceID, in.ServerRef, r.PathValue("id"))
			deploymentResponse(w, v, err, 200)
		})
		route("GET /api/v1/workspaces/{id}/credential-bindings", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			if _, err := store.Workspace(r.PathValue("id")); err != nil {
				deploymentError(w, err)
				return
			}
			deploymentJSON(w, 200, store.Bindings(a, r.PathValue("id")))
		})
		route("PUT /api/v1/workspaces/{id}/credential-bindings", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in struct {
				ServerRef    string `json:"serverRef"`
				CredentialID string `json:"credentialId"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			if err := store.BindCredential(a, r.PathValue("id"), in.ServerRef, in.CredentialID); err != nil {
				deploymentError(w, err)
				return
			}
			deploymentJSON(w, 200, store.Bindings(a, r.PathValue("id")))
		})
	}
}
func deploymentDecode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		http.Error(w, "invalid deployment input", 400)
		return false
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		http.Error(w, "expected one input object", 400)
		return false
	}
	return true
}
func deploymentRevision(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.Header.Get("If-Match")
	if raw == "" {
		http.Error(w, "If-Match required", http.StatusPreconditionRequired)
		return 0, false
	}
	rev, err := strconv.Atoi(strings.Trim(raw, `"`))
	if err != nil || rev < 1 {
		http.Error(w, "invalid revision", 400)
		return 0, false
	}
	return rev, true
}
func deploymentJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func deploymentResponse(w http.ResponseWriter, v any, err error, status int) {
	if err != nil {
		deploymentError(w, err)
		return
	}
	switch d := v.(type) {
	case app.DeploymentDocument:
		w.Header().Set("ETag", strconv.Quote(strconv.Itoa(d.Revision)))
	case app.DeploymentWorkspace:
		w.Header().Set("ETag", strconv.Quote(strconv.Itoa(d.Revision)))
	}
	deploymentJSON(w, status, v)
}
func deploymentError(w http.ResponseWriter, err error) {
	status := http.StatusUnprocessableEntity
	switch {
	case errors.Is(err, app.ErrDeploymentNotFound):
		status = 404
	case errors.Is(err, app.ErrDeploymentForbidden):
		status = 403
	case errors.Is(err, app.ErrDeploymentConflict):
		status = http.StatusConflict
	case webInternalError(err):
		status = http.StatusInternalServerError // Storage and OS failures are not input problems.
	}
	http.Error(w, err.Error(), status)
}

func deploymentReadDocument(store *app.DeploymentStore, r *http.Request) (app.DeploymentDocument, error) {
	revision := 0
	if raw := r.URL.Query().Get("revision"); raw != "" {
		var err error
		revision, err = strconv.Atoi(raw)
		if err != nil || revision < 0 {
			return app.DeploymentDocument{}, errors.New("invalid document revision")
		}
	}
	return store.DocumentRevision(r.PathValue("id"), revision)
}

// webInternalError recognises filesystem and system-call failures, whose text
// names server paths; they are reported as server errors with generic text.
func webInternalError(err error) bool {
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	var sysErr *os.SyscallError
	return errors.As(err, &pathErr) || errors.As(err, &linkErr) || errors.As(err, &sysErr)
}
