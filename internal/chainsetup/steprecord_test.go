package chainsetup

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

// The ledger holds two kinds of step and says so in two lists. These hold the
// lists and the recording to each other.
//
// The defect they prevent is quiet by construction. A name recorded into
// State.Steps that neither list declares is written and then read by nobody:
// resume walks UpStepNames, preflight asks for "start", and composeNeeds
// answers a nil slice for a key it does not know, so every prerequisite of a
// misspelled step is vacuously satisfied. Nothing fails; the step simply does
// not count. Checking the source for what is actually recorded is the only way
// to see it, because at run time the map takes whatever it is given.

// recordedStepNames returns every step name this package records as a literal,
// with the file and line that records it.
func recordedStepNames(t *testing.T) map[string][]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	fset := token.NewFileSet()
	found := map[string][]string{}
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, file)
	}
	// Step names the package holds as constants. A name written out at every
	// call is the duplication the rest of this package removes, so the check
	// has to see through the constant to the word — otherwise the ratchet
	// pushes back towards the literal it was never about.
	consts := map[string]string{}
	for _, file := range files {
		for _, d := range file.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
					continue
				}
				lit, ok := vs.Values[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if v, err := strconv.Unquote(lit.Value); err == nil {
					consts[vs.Names[0].Name] = v
				}
			}
		}
	}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "markStep" && sel.Sel.Name != "MarkStepFailed") {
				return true
			}
			var step string
			switch arg := call.Args[0].(type) {
			case *ast.BasicLit:
				if arg.Kind != token.STRING {
					return true
				}
				unquoted, err := strconv.Unquote(arg.Value)
				if err != nil {
					return true
				}
				step = unquoted
			case *ast.Ident:
				// A constant this package declares, resolved to its word. An
				// identifier from anywhere else is a name passed through, and
				// a caller that passes one takes it from a table already held
				// to a list.
				name, ok := consts[arg.Name]
				if !ok {
					return true
				}
				step = name
			default:
				return true
			}
			at := fset.Position(call.Args[0].Pos())
			found[step] = append(found[step], filepath.Base(at.Filename)+":"+strconv.Itoa(at.Line))
			return true
		})
	}
	if len(found) == 0 {
		t.Fatal("no recorded step names found — the check is looking in the wrong place")
	}
	return found
}

// TestRecordedStepsAreDeclared is the ratchet the two lists exist for: a name
// this package records has to be one the lists know, so the next person to add
// a markStep call says which kind they added.
func TestRecordedStepsAreDeclared(t *testing.T) {
	declared := map[string]string{}
	for _, name := range UpStepNames {
		declared[name] = "UpStepNames"
	}
	for _, name := range OpStepNames {
		if from, dup := declared[name]; dup {
			t.Errorf("%q is in both %s and OpStepNames — a name has one kind", name, from)
		}
		declared[name] = "OpStepNames"
	}

	var undeclared []string
	for step, where := range recordedStepNames(t) {
		if declared[step] == "" {
			undeclared = append(undeclared, step+" ("+strings.Join(where, ", ")+")")
		}
	}
	sort.Strings(undeclared)
	if len(undeclared) > 0 {
		t.Errorf("these step names are recorded but neither list declares them:\n  %s\n"+
			"Add a rung to UpStepNames or an operation to OpStepNames — a name in neither is written and read by nobody.",
			strings.Join(undeclared, "\n  "))
	}
}

// TestDeclaredStepsAreRecorded is the other direction: a list that names a step
// nothing records is a list going stale, which is how the reader it exists for
// ends up trusting it wrongly.
func TestDeclaredStepsAreRecorded(t *testing.T) {
	recorded := recordedStepNames(t)
	var ghosts []string
	for _, name := range append(append([]string{}, UpStepNames...), OpStepNames...) {
		if len(recorded[name]) == 0 {
			ghosts = append(ghosts, name)
		}
	}
	sort.Strings(ghosts)
	if len(ghosts) > 0 {
		t.Errorf("declared but recorded nowhere:\n  %s", strings.Join(ghosts, "\n  "))
	}
}
