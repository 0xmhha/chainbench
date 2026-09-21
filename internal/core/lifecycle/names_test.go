package lifecycle

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"testing"
)

// TestEveryDeclaredStateHasAName reads this package's own source and holds the
// constants and the name table together.
//
// [Status.String] is what turns a state into something a person can read, and
// it reads a map somebody types by hand. A constant added without its line in
// that map still compiles, still flows through the machine, and prints as
// Status(0x1903) at the one moment it matters — while somebody is reading why a
// run stopped. The two other checks here cannot catch it: one walks the
// transition table and one walks the names, so a constant missing from BOTH is
// invisible to them.
//
// Parsing the source is what closes that. The declaration is the only place a
// state is certainly written down, so it is the side to compare from.
func TestEveryDeclaredStateHasAName(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "status.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, sp := range gen.Specs {
			vs, ok := sp.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, id := range vs.Names {
				// Only the exported states. The block geometry (blockSize,
				// failureSlot) and the area bases are not states a run is ever
				// in, so they have no name to print.
				if !id.IsExported() {
					continue
				}
				if !declaresAName(id.Name) {
					missing = append(missing, id.Name)
				}
			}
		}
	}
	sort.Strings(missing)
	for _, n := range missing {
		t.Errorf("%s is declared and names has no entry for it, so it would print as hex", n)
	}
}

// declaresAName reports whether the name table has an entry whose text is this
// identifier. The table maps a value to its own identifier spelled out, so the
// comparison is against the text rather than against the value — a constant
// whose value collides with another's would otherwise look named.
func declaresAName(ident string) bool {
	for _, n := range names {
		if n == ident {
			return true
		}
	}
	return false
}

// TestStringIsTheIdentifier pins what String returns, because the point of the
// name table is that a message names the thing a reader can grep for.
func TestStringIsTheIdentifier(t *testing.T) {
	for _, c := range []struct {
		s    Status
		want string
	}{
		{ChainOpenWorkspace, "ChainOpenWorkspace"},
		{ChainLaunchNodesFailPortBusy, "ChainLaunchNodesFailPortBusy"},
		{ChainReady, "ChainReady"},
		{FailLoop, "FailLoop"},
	} {
		if got := c.s.String(); got != c.want {
			t.Errorf("%#x printed as %q, want %q", uint32(c.s), got, c.want)
		}
	}
	// An undeclared value says so in the shape the table is written in, so the
	// number a reader sees is the number they can look up.
	if got := Status(0x1234).String(); got != "Status(0x1234)" {
		t.Errorf("an undeclared value printed as %q", got)
	}
}

// TestNoTwoStatesShareAValue is the check the near-miss asked for.
//
// Writing the operational area put its shared failures at
// areaChainOp + failureSlot, which is the STOP block's failure range because
// stop is that area's first block. Two names, one value. Nothing would have
// said so: the name table is keyed BY value, so the second entry silently
// replaces the first, and every other check here reads that table and sees one
// consistent state.
//
// The declarations are the side to compare from, so this parses them the way
// TestEveryDeclaredStateHasAName does and evaluates what the name table cannot.
func TestNoTwoStatesShareAValue(t *testing.T) {
	byValue := map[Status][]string{}
	for s, n := range names {
		byValue[s] = append(byValue[s], n)
	}
	// names is value-keyed, so a collision is invisible in it. Count the
	// declarations instead: more declared names than distinct values means two
	// share one.
	declared := 0
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "status.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, sp := range gen.Specs {
			vs, ok := sp.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, id := range vs.Names {
				if id.IsExported() {
					declared++
				}
			}
		}
	}
	if declared != len(names) {
		t.Errorf("%d exported states are declared and the name table holds %d — "+
			"a value declared twice keeps one name and loses the other", declared, len(names))
	}
}
