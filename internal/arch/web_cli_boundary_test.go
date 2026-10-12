package arch_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Web owns review and presentation. Target transport, process identification
// and execution planning must remain in the CLI's shared modules.
func TestWebDoesNotImplementTargetTransportOrPlanning(t *testing.T) {
	dir := "../app"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || !strings.HasPrefix(name, "web_") && !strings.HasPrefix(name, "deployment_") {
			continue
		}
		set := token.NewFileSet()
		f, err := parser.ParseFile(set, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		imports := map[string]string{}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			alias := filepath.Base(path)
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			imports[alias] = path
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			qual, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			path, method := imports[qual.Name], sel.Sel.Name
			forbidden := strings.HasSuffix(path, "/core/remote") && (method == "SSHExec" || method == "SSHDriver" || method == "ShellQuote") || strings.HasSuffix(path, "/core/process") && (method == "ProcessTableOf" || method == "ObserveLaunch" || method == "ObserveVacancy") || strings.HasSuffix(path, "/testengine") && strings.HasPrefix(method, "Plan")
			if forbidden {
				t.Errorf("%s: Web must delegate %s.%s to chainbench", set.Position(call.Pos()), qual.Name, method)
			}
			return true
		})
	}
}
