package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
)

// spaFiles is the built Svelte SPA (decision D5). The sources live in web/ and
// are compiled with `npm --prefix web run build`, which writes the static bundle
// to pkg/dashboard/spa/ (committed) so it embeds into the binary. The SPA is
// served under /app/ and speaks the same contract as the interim page — the SSE
// stream at /events and the run list at /api/runs.
//
//go:embed all:spa
var spaFiles embed.FS

// spaHandler serves the built SPA under /app/. It is nil-safe to embed: the spa/
// directory is always present in the tree (a placeholder ships when unbuilt), so
// the embed never fails the build.
func spaHandler() http.Handler {
	sub, err := fs.Sub(spaFiles, "spa")
	if err != nil {
		// spa/ is embedded at compile time; a failure here is a build/packaging
		// error, not a runtime condition.
		panic(err)
	}
	return http.StripPrefix("/app/", http.FileServer(http.FS(sub)))
}
