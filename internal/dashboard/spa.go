package dashboard

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
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
	packed := &sync.Map{}
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
			if strings.HasPrefix(r.URL.Path, spaAssetPrefix) {
				w.Header().Set("Cache-Control", spaAssetCache)
				if serveGzippedAsset(w, r, sub, packed) {
					return
				}
			}
			files.ServeHTTP(w, r)
		}
	})
}

// spaAssetPrefix is where the build puts hashed files. A hash names its bytes, so
// a browser can keep them for good.
const (
	spaAssetPrefix = "/assets/"
	spaAssetCache  = "public, max-age=31536000, immutable"
)

// serveGzippedAsset answers a text asset compressed when the client accepts it,
// compressing each file once. It reports false to leave the request to the plain
// file server.
func serveGzippedAsset(w http.ResponseWriter, r *http.Request, assets fs.FS, packed *sync.Map) bool {
	kind := mime.TypeByExtension(path.Ext(r.URL.Path))
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || !strings.HasPrefix(kind, "text/") {
		return false
	}
	body, ok := packed.Load(r.URL.Path)
	if !ok {
		raw, err := fs.ReadFile(assets, strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil {
			return false
		}
		var out bytes.Buffer
		zw := gzip.NewWriter(&out)
		if _, err = zw.Write(raw); err != nil {
			return false
		}
		if err = zw.Close(); err != nil {
			return false
		}
		body, _ = packed.LoadOrStore(r.URL.Path, out.Bytes())
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	_, _ = w.Write(body.([]byte))
	return true
}
