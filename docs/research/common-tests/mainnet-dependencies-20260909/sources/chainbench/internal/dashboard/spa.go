package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
)

// spaFiles is the built Svelte SPA (decision D5). The sources live in web/ and
// are compiled with `npm --prefix web run build`, which writes the static bundle
// to pkg/dashboard/spa/ (committed) so it embeds into the binary. The SPA is
// served at the site root and speaks the same contract as the legacy page — the
// SSE stream at /events and the run list at /api/runs.
//
//go:embed all:spa
var spaFiles embed.FS

// spaHandler serves the built SPA (its index at / and hashed assets at /assets/).
// The spa/ directory is embedded at compile time, so a Sub failure is a
// build/packaging error, not a runtime condition.
func spaHandler() http.Handler {
	sub, err := fs.Sub(spaFiles, "spa")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
