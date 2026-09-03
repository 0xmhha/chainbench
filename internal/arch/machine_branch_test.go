package arch_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// machineBranchAllowed is the ratchet for the no-branching rule
// (architecture-v2 §4): a consumer of core/machine must not choose behavior
// by the machine's kind — local is just the loopback machine, and the
// difference is the machine module's internal concern.
//
// Every entry is either a permanent exemption (it CONSTRUCTS a Spec, which
// requires stating its kind, or it owns its own remote notion) or a deferral
// to the worklist task that dissolves it. The list may only shrink: an entry
// whose file no longer branches fails the test until it is removed here.
var machineBranchAllowed = map[string]string{
	"internal/resource":   "the module owns the server set: it constructs Specs from its entries and wires the dial. Server.IsRemote is the set's own field logic, not machine.Kind (P1.2 merged the surface and the format; R3 merged the machine spec in)",
	"internal/chainsetup": "deferred: display and keys-path branches dissolve with V5 follow-ups and V6.3",
}

// TestMachineConsumersDoNotBranchOnKind walks every non-test Go file and
// flags (a) qualified uses of resource.Kind* constants and (b) IsRemote calls
// in files that import core/machine, unless the file (or its package dir) is
// in the ratchet above.
func TestMachineConsumersDoNotBranchOnKind(t *testing.T) {
	root := "../.."
	var offenders []string
	seen := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "bin" || name == "node_modules" || name == "data" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if !strings.HasPrefix(rel, "internal/") && !strings.HasPrefix(rel, "cmd/") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		found := 0
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			// IsRemote is flagged regardless of imports: the branch a method
			// call expresses does not need the machine package in scope.
			if sel.Sel.Name == "IsRemote" {
				found++
				if !allowed(rel) {
					offenders = append(offenders, rel+": calls IsRemote")
				}
			}
			return true
		})
		// An allowlisted file counts as seen only while it still branches, so
		// an entry whose branches were removed fails below until deleted.
		if found > 0 && allowed(rel) {
			seen[allowKey(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range offenders {
		t.Errorf("machine consumer branches on kind: %s (architecture-v2 §4: resolve and use the handles; the kind is the machine module's concern)", o)
	}
	for key, why := range machineBranchAllowed {
		if !seen[key] {
			t.Errorf("ratchet entry %q (%s) matched no file — shrink machineBranchAllowed", key, why)
		}
	}
}

func allowed(rel string) bool { return allowKey(rel) != "" }

// allowKey returns the ratchet key covering rel: the exact file, or its
// directory when a whole package is exempt.
func allowKey(rel string) string {
	if _, ok := machineBranchAllowed[rel]; ok {
		return rel
	}
	dir := filepath.Dir(rel)
	if _, ok := machineBranchAllowed[dir]; ok {
		return dir
	}
	return ""
}
