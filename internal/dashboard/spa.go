package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// spaFiles is the built Svelte SPA (decision D5). The sources live in web/ and
// are compiled with `npm --prefix web run build`, which writes the static bundle
// to internal/dashboard/spa/ (committed) so it embeds into the binary. The SPA is
// served at the site root and reads the /api/v1 contract, with live updates on
// the one stream at /api/v1/events.
//
//go:embed all:spa
var spaFiles embed.FS

// spaPages are the fixed screens; spaDetail matches one job's or one node's
// detail screen.
var (
	spaPages  = []string{"/", "/chains", "/tests", "/monitoring", "/history", "/settings"}
	spaDetail = regexp.MustCompile(`^/(jobs/[0-9a-f]{32}|nodes/[0-9a-f]{32}/[A-Za-z0-9_.-]{1,64})$`)
)

// spaHandler serves the built SPA (its index at / and hashed assets at /assets/).
// The spa/ directory is embedded at compile time, so a Sub failure is a
// build/packaging error, not a runtime condition.
func spaHandler() http.Handler {
	sub, err := fs.Sub(spaFiles, "spa")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only known UI routes receive the shell. Missing API/assets must stay 404.
		switch path := r.URL.Path; {
		case slices.Contains(spaPages, path), spaDetail.MatchString(path):
			index, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.Error(w, "SPA unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write(index)
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			files.ServeHTTP(w, r)
		}
	})
}
