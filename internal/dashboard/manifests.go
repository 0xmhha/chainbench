package dashboard

import (
	"net/http"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// WithManifests enables the existing plugins and declarative external imports.
// Binary assets are provisioned by the server administrator, never arbitrary browser paths.
func WithManifests(store *app.ManifestStore, authenticate DeploymentAuthenticator, assets []app.ManifestBinary, keys string) Option {
	return func(s *Server) {
		route := func(pattern string, fn func(http.ResponseWriter, *http.Request, app.DeploymentActor)) {
			s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				if r.Method != "GET" {
					recorder := &manifestResponse{ResponseWriter: w, status: http.StatusOK}
					w = recorder
					actorID := "unauthenticated"
					if authenticate != nil {
						if actor, err := authenticate(r); err == nil {
							actorID = actor.ID
						}
					}
					target := r.PathValue("id")
					if err := store.AuditManifest(actorID, pattern, target, 0); err != nil {
						http.Error(w, "audit storage unavailable", http.StatusInternalServerError)
						return
					}
					defer func() { _ = store.AuditManifest(actorID, pattern, target, recorder.status) }()
				}
				if authenticate == nil {
					http.Error(w, "login required", http.StatusUnauthorized)
					return
				}
				a, err := authenticate(r)
				if err != nil || a.ID == "" {
					http.Error(w, "login required", http.StatusUnauthorized)
					return
				}
				if r.Method != "GET" {
					scheme := "http"
					if r.TLS != nil {
						scheme = "https"
					}
					if origin := r.Header.Get("Origin"); origin != "" && origin != scheme+"://"+r.Host {
						deploymentError(w, app.ErrDeploymentForbidden)
						return
					}
				}
				fn(w, r, a)
			})
		}
		route("GET /api/v1/contracts/manifest", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			deploymentJSON(w, 200, map[string]any{"contractVersion": "2", "families": registry.FamilyNames(), "dialects": nodeconfig.DialectNames(), "protocols": registry.Names()})
		})
		route("GET /api/v1/manifests", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			items, err := store.List()
			deploymentResponse(w, map[string]any{"items": items}, err, 200)
		})
		route("GET /api/v1/manifests/{id}", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			item, err := store.Get(r.PathValue("id"))
			deploymentResponse(w, item, err, 200)
		})
		route("GET /api/v1/manifest-assets", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			items := []map[string]string{}
			for _, asset := range assets {
				items = append(items, map[string]string{"id": asset.ID, "chain": asset.Chain, "sha256": asset.SHA256, "commit": asset.Commit})
			}
			deploymentJSON(w, 200, map[string]any{"items": items})
		})
		route("POST /api/v1/manifests/validate", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.ManifestInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			_, err := app.ValidateManifest(in)
			issues := []string{}
			if err != nil {
				issues = append(issues, err.Error())
			}
			deploymentJSON(w, 200, map[string]any{"valid": err == nil, "errors": issues, "contractVersion": "2"})
		})
		route("POST /api/v1/manifests", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in app.ManifestInput
			if !deploymentDecode(w, r, &in) {
				return
			}
			item, err := store.Save(a, in)
			deploymentResponse(w, item, err, 201)
		})
		route("POST /api/v1/manifests/{id}/setup", func(w http.ResponseWriter, r *http.Request, a app.DeploymentActor) {
			var in struct {
				AssetID string `json:"assetId"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			for _, asset := range assets {
				if asset.ID == in.AssetID {
					out, err := store.SetupManifest(r.Context(), a, r.PathValue("id"), asset, keys)
					deploymentResponse(w, out, err, 200)
					return
				}
			}
			deploymentError(w, app.ErrDeploymentNotFound)
		})
	}
}

// manifestResponse captures the status for the secret-free audit receipt.
type manifestResponse struct {
	http.ResponseWriter
	status int
}

func (w *manifestResponse) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
