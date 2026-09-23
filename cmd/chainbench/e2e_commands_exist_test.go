package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// A test behind a build tag that names a command which no longer exists is not
// coverage — it is the appearance of it.
//
// Three hardfork e2e tests invoked `upgrade run` long after the root command
// stopped registering an `upgrade` group. They compiled, because cobra takes
// args as strings, and they skipped, because they are gated on chain binaries
// being present. So nothing ever said the command was gone.
//
// This walks every test in this package — gated ones included, since the parse
// does not care about build tags — collects the command paths they invoke, and
// asks the real root command to find each one.
// invocationDebt holds command paths a test still names although the CLI does
// not have them, each with what is at stake in fixing it.
//
// It may only shrink. An entry is not permission — it is a note that coverage
// nobody can run is sitting here, and what would be lost by deleting it instead.
var invocationDebt = map[string]string{}

func TestEveryCommandATestInvokesExists(t *testing.T) {
	root := newRootCmd()
	paths, files := invokedCommands(t)
	if len(paths) == 0 {
		t.Fatal("no command invocations found, so the walk is wrong rather than the tests silent")
	}
	unpaid := map[string]bool{}
	for p := range invocationDebt {
		unpaid[p] = true
	}
	var missing []string
	for _, p := range paths {
		if _, _, err := root.Find(strings.Fields(p)); err != nil {
			if _, known := invocationDebt[p]; known {
				t.Logf("known debt: %s — %s", p, invocationDebt[p])
				delete(unpaid, p)
				continue
			}
			missing = append(missing, p+"  ("+strings.Join(files[p], ", ")+")")
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("these tests invoke commands the CLI does not have:\n  %s\n"+
			"Point them at a command that exists, or remove them — a test naming a gone command reads as coverage and is not.",
			strings.Join(missing, "\n  "))
	}
	// An entry for a path nothing names any more is debt that was paid without
	// the note being removed, which makes the list read as worse than it is.
	for p := range unpaid {
		t.Errorf("invocationDebt names %q, which no test invokes — remove the entry", p)
	}
	t.Logf("%d command invocations checked, %d known debt", len(paths), len(invocationDebt))
}

// invokedCommands finds the leading command words of each argv a test builds:
// cmd.SetArgs([]string{"chain", "up", "--flag", …}) and run(t, "chain", "up", …).
// Words are taken until the first flag or non-literal, which is where the
// command path ends.
func invokedCommands(t *testing.T) ([]string, map[string][]string) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string][]string{}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Clean(name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		// argv built into a variable before being handed over.
		argvLits := map[string][]ast.Expr{}
		ast.Inspect(f, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				return true
			}
			id, ok := as.Lhs[0].(*ast.Ident)
			if !ok {
				return true
			}
			lit, ok := as.Rhs[0].(*ast.CompositeLit)
			if !ok {
				return true
			}
			if arr, ok := lit.Type.(*ast.ArrayType); ok {
				if e, ok := arr.Elt.(*ast.Ident); ok && e.Name == "string" {
					argvLits[id.Name] = lit.Elts
				}
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			var args []ast.Expr
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				if fn.Sel.Name != "SetArgs" || len(call.Args) != 1 {
					return true
				}
				switch a := call.Args[0].(type) {
				case *ast.CompositeLit:
					args = a.Elts
				case *ast.Ident:
					// SetArgs(args) where args was built earlier. The first
					// version of this check read only inline literals and
					// missed a test that assigned its argv to a variable.
					lit, ok := argvLits[a.Name]
					if !ok {
						return true
					}
					args = lit
				default:
					return true
				}
			case *ast.Ident:
				if fn.Name != "run" || len(call.Args) < 2 {
					return true
				}
				args = call.Args[1:]
			default:
				return true
			}
			var words []string
			for _, a := range args {
				lit, ok := a.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					break
				}
				v, err := strconv.Unquote(lit.Value)
				if err != nil || !looksLikeCommand(v) {
					break
				}
				words = append(words, v)
			}
			if len(words) == 0 {
				return true
			}
			record(seen, strings.Join(words, " "), name)
			return true
		})
	}
	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, seen
}

// looksLikeCommand keeps the walk from reading any list of strings as an argv.
// A command word is lowercase letters, digits and hyphens, and short. Without
// this the broadened walk read a table of hex keys and a list of Solidity
// signatures as command paths.
func looksLikeCommand(v string) bool {
	if v == "" || len(v) > 24 {
		return false
	}
	for i, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		case r == '-' && i > 0:
		default:
			return false
		}
	}
	return true
}

// record notes that file invoked the command path p.
func record(seen map[string][]string, p, file string) {
	if !contains(seen[p], file) {
		seen[p] = append(seen[p], file)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
