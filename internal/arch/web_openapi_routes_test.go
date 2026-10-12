package arch

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// webRoute is one method and path; segments in braces match any segment.
type webRoute struct{ method, path string }

var webRouteLiteral = regexp.MustCompile(`(?:HandleFunc|Handle|route)\(\s*"(GET|POST|PUT|PATCH|DELETE) (/api/[^"\s]*)"`)

// webServedRoutes reads every /api route the dashboard registers. Routes are
// registered with literal patterns, so the sources are the complete list.
func webServedRoutes(t *testing.T) []webRoute {
	files, err := filepath.Glob("../dashboard/*.go")
	if err != nil {
		t.Fatal(err)
	}
	seen, out := map[webRoute]bool{}, []webRoute{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range webRouteLiteral.FindAllStringSubmatch(string(raw), -1) {
			r := webRoute{m[1], m[2]}
			if !seen[r] {
				seen[r] = true
				out = append(out, r)
			}
		}
	}
	return out
}

func webDocumentedRoutes(t *testing.T) []webRoute {
	raw, err := os.ReadFile("../../docs/dev/web-ui.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	out := []webRoute{}
	for path, ops := range doc.Paths {
		for method := range ops {
			switch method {
			case "get", "post", "put", "patch", "delete":
				out = append(out, webRoute{strings.ToUpper(method), path})
			}
		}
	}
	return out
}

// webRouteCovers reports whether template (from either side) covers path:
// the same segments, where a {name} segment matches any one segment.
func webRouteCovers(template, path webRoute) bool {
	if template.method != path.method {
		return false
	}
	a, b := strings.Split(template.path, "/"), strings.Split(path.path, "/")
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] && !strings.HasPrefix(a[i], "{") && !strings.HasPrefix(b[i], "{") {
			return false
		}
	}
	return true
}

// TestWebOpenAPIListsEveryServedRoute keeps the HTTP contract and the server
// in step: every registered /api route is documented, and every documented
// route is served.
func TestWebOpenAPIListsEveryServedRoute(t *testing.T) {
	served, documented := webServedRoutes(t), webDocumentedRoutes(t)
	if len(served) < 40 {
		t.Fatalf("found only %d served routes; the route reader is broken", len(served))
	}
	missing, stale := []string{}, []string{}
	for _, s := range served {
		ok := false
		for _, d := range documented {
			ok = ok || webRouteCovers(d, s)
		}
		if !ok {
			missing = append(missing, s.method+" "+s.path)
		}
	}
	for _, d := range documented {
		ok := false
		for _, s := range served {
			ok = ok || webRouteCovers(d, s)
		}
		if !ok {
			stale = append(stale, d.method+" "+d.path)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	if len(missing) > 0 {
		t.Errorf("served but not in docs/dev/web-ui.openapi.yaml:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("in docs/dev/web-ui.openapi.yaml but not served:\n  %s", strings.Join(stale, "\n  "))
	}
}
