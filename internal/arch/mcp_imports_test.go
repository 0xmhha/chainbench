package arch_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// mcpImportAllowed is the ratchet for the surface rule (architecture-v2 §2):
// MCP reaches features through the app layer. An internal/mcp file may import
// app, its own package tree, and non-internal dependencies; every other
// internal import is a tool still wired straight to core, listed here with
// the migration that removes it. The list may only shrink: an entry whose
// import disappeared fails the test until removed.
var mcpImportAllowed = map[string]string{
	"internal/accounts":              "account tools; migrates with the account verbs' module move",
	"internal/core/collector":        "network status collection; V6 follow-up",
	"internal/core/logs":             "log reading; V6 follow-up",
	"internal/core/machine":          "target kind rendering in net tools; goes with the V5.4 display cleanups",
	"internal/core/netreg":           "network registry reads; V6 follow-up",
	"internal/core/node":             "node set types; V6 follow-up",
	"internal/core/obs":              "dashboard event bus; V6 follow-up",
	"internal/core/pipeline/testrun": "legacy run pipeline; retires with T7.11",
	"internal/core/registry":         "chain plugin lookup; V6 follow-up",
	"internal/core/remote":           "remote exec tool; V6 follow-up",
	"internal/core/rpc":              "direct RPC tools; V6 follow-up",
	"internal/testkit":               "test fixtures for remote tools; V6 follow-up",
}

// TestMCPGoesThroughApp pins the asymmetric surface rule: CLI calls core
// directly, MCP goes through app. Every internal import in internal/mcp that
// is not app (or mcp itself) must be on the shrink-only ratchet above.
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
			if rel == "internal/app" || strings.HasPrefix(rel, "internal/mcp") {
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
