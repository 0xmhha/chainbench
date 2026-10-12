package dashboard

import (
	"context"
	"net/http"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

// WithChainSurfaces serves each chain's command and option catalog to every
// signed-in role. With assetId it also asks that registered binary for its
// help pages, which neither starts a node nor opens a database.
func WithChainSurfaces(engine *app.WebChainEngine, authenticate DeploymentAuthenticator) Option {
	return func(s *Server) {
		s.mux.HandleFunc("GET /api/v1/chains/{chainId}/surface", func(w http.ResponseWriter, r *http.Request) {
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
			ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
			defer cancel()
			surface, err := engine.ChainSurface(ctx, a, r.PathValue("chainId"), r.URL.Query().Get("assetId"))
			deploymentResponse(w, surface, err, http.StatusOK)
		})
	}
}
