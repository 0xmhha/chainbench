package arch_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// mcpImportAllowed is the ratchet for the surface rule (architecture-v2 §2):
// every surface reaches features through the app layer. An internal/mcp file
// may import app, its L5 feature binding registry, its own package tree, and non-internal dependencies; every
// other internal import would be a tool still wired straight to core, listed
// here with the migration that removes it.
//
// It is empty as of 2026-09-05 (U6): the last six entries — core/rpc,
// core/collector, resource, core/node, core/session, core/remote — went with
// the query migration they each named. The list may only shrink, and an entry
// whose import has disappeared fails the test until it is removed, so an empty
// map is the rule holding rather than the rule being unenforced.
var mcpImportAllowed = map[string]string{}

// TestMCPGoesThroughApp pins the application boundary. The L5 registry selects
// typed app functions; it is not a transport or engine escape hatch. Its own
// dependency guard and AST registration coverage keep that route through app.
func TestMCPGoesThroughApp(t *testing.T) {
	const modPrefix = "github.com/0xmhha/chainbench/"
	files, err := filepath.Glob("../mcp/*.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(p, modPrefix) {
				continue
			}
			rel := strings.TrimPrefix(p, modPrefix)
			if rel == "internal/app" || rel == "internal/feature" || strings.HasPrefix(rel, "internal/mcp") {
				continue
			}
			if _, ok := mcpImportAllowed[rel]; ok {
				seen[rel] = true
				continue
			}
			t.Errorf("%s imports %s directly — MCP goes through app (architecture-v2 §2); add an app wrapper instead", filepath.Base(path), rel)
		}
	}
	for rel, why := range mcpImportAllowed {
		if !seen[rel] {
			t.Errorf("ratchet entry %q (%s) matched no import — shrink mcpImportAllowed", rel, why)
		}
	}
}

func TestFeatureBindingsCannotReachPastApp(t *testing.T) {
	const prefix = "github.com/0xmhha/chainbench/internal/"
	files, err := filepath.Glob("../feature/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			name := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(name, prefix) && name != prefix+"app" {
				t.Errorf("feature binding %s bypasses app through %s", path, name)
			}
		}
	}
}
