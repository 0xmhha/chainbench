package chainsetup

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The protocol is two claims that are easy to make and easy to let rot: every
// message has a name, and a message's value says which way it goes. Both are
// only true while somebody checks, so these read the source rather than a
// second list kept by hand.

// band is one range of the machine's base and what a message in it must look
// like.
type band struct {
	name     string
	lo, hi   statemachine.What // offsets from the base, hi exclusive
	exported bool
	prefix   string
}

// bands is the layout protocol.go documents. A message's value decides which
// band it is in, and the band decides what its name has to be.
var bands = []band{
	{"a Cmd the outside sends down", 0x000, 0x040, true, "Cmd"},
	{"a Cmd this machine sends up", 0x040, 0x100, true, "Cmd"},
	{"an Event this machine leaves itself", 0x100, 0x180, false, "event"},
	{"an Event something below posts up", 0x180, 0x200, true, "Event"},
}

// TestEveryMessageHasAName is the "missing name" check.
//
// The reference builds this list by reflection over its constants so that a log
// prints names. Go cannot, so the list is written out and this holds it to the
// declarations — in both directions, because an entry for a message that no
// longer exists is as misleading as a message with no entry.
func TestEveryMessageHasAName(t *testing.T) {
	declared := declaredWhats(t)
	named := map[string]bool{}
	for _, name := range whatNames {
		named[name] = true
	}

	var missing, stale []string
	for _, name := range declared {
		if !named[name] {
			missing = append(missing, name)
		}
	}
	for name := range named {
		if !contains(declared, name) {
			stale = append(stale, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	for _, name := range missing {
		t.Errorf("%s is a message and whatNames does not name it — a log would print its number", name)
	}
	for _, name := range stale {
		t.Errorf("whatNames names %s, which protocol.go no longer declares — remove the entry", name)
	}
}

// TestEveryMessageHasItsOwnValue: two messages sharing a value are one message
// as far as any log or range check is concerned.
func TestEveryMessageHasItsOwnValue(t *testing.T) {
	if got, want := len(whatNames), len(declaredWhats(t)); got != want {
		t.Fatalf("whatNames holds %d entries for %d messages, so at least two share a value", got, want)
	}
}

// TestAMessagesValueSaysWhichWayItGoes holds the bands to the names.
//
// This is the property the numbering exists for: a reader who meets 0x1103 in a
// log knows it is the machine talking to itself before knowing which message it
// is, and the compiler knows the same thing from the leading letter.
func TestAMessagesValueSaysWhichWayItGoes(t *testing.T) {
	for w, name := range whatNames {
		off := w - statemachine.BaseChain
		if off < 0 || off >= 0x200 {
			t.Errorf("%s is %#x, which is %#x past BaseChain and outside this machine's range", name, int(w), int(off))
			continue
		}
		var in *band
		for i := range bands {
			if off >= bands[i].lo && off < bands[i].hi {
				in = &bands[i]
				break
			}
		}
		if in == nil {
			t.Errorf("%s at offset %#x falls in no band", name, int(off))
			continue
		}
		if exported := name[0] >= 'A' && name[0] <= 'Z'; exported != in.exported {
			t.Errorf("%s is %s, so it should%s be exported", name, in.name, ifElse(in.exported, "", " not"))
		}
		if !strings.HasPrefix(name, in.prefix) {
			t.Errorf("%s is %s, so its name should start with %q", name, in.name, in.prefix)
		}
	}
}

// TestWhatName covers both halves of the lookup, including the message from
// somewhere else that has no name here.
func TestWhatName(t *testing.T) {
	if got := WhatName(CmdCompose); got != "CmdCompose" {
		t.Errorf("WhatName(CmdCompose) is %q", got)
	}
	if got := WhatName(statemachine.BaseTest + 7); got != "What(0x8007)" {
		t.Errorf("a message from another machine came back as %q, want its number", got)
	}
}

// TestEveryMessageTypeCarriesADeclaredWhat.
//
// A message body whose What is not one of this machine's is a message nothing
// will ever route, and the compiler is happy with it: What returns an int.
func TestEveryMessageTypeCarriesADeclaredWhat(t *testing.T) {
	// Every type in this package that implements statemachine.Message. The list
	// is checked against the source below, so adding a message without adding
	// it here fails.
	types := map[string]statemachine.Message{
		"Compose":         Compose{},
		"RunStep":         RunStep{},
		"Stop":            Stop{},
		"ClearError":      ClearError{},
		"PostCompose":     PostCompose{},
		"OnQuit":          OnQuit{},
		"NodeDied":        NodeDied{},
		"stageDone":       stageDone{},
		"stageFailed":     stageFailed{},
		"workspaceOpened": workspaceOpened{},
		"nodeTableBuilt":  nodeTableBuilt{},
	}
	for name, msg := range types {
		if _, ok := whatNames[msg.What()]; !ok {
			t.Errorf("%s carries %#x, which is not one of this machine's messages", name, int(msg.What()))
		}
		exportedType := name[0] >= 'A' && name[0] <= 'Z'
		exportedWhat := whatNames[msg.What()][0] >= 'A' && whatNames[msg.What()][0] <= 'Z'
		if exportedType != exportedWhat {
			t.Errorf("%s is%s exported and its message %s is%s — a body and its What are visible to the same callers or to different ones",
				name, ifElse(exportedType, "", " not"), whatNames[msg.What()], ifElse(exportedWhat, "", " not"))
		}
	}

	for _, name := range messageTypesInSource(t) {
		if _, ok := types[name]; !ok {
			t.Errorf("%s implements Message and this test does not know it — add it here too", name)
		}
	}
}

// declaredWhats reads protocol.go for the constants typed statemachine.What,
// including the ones an iota group carries the type down to.
func declaredWhats(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "protocol.go", nil, 0)
	if err != nil {
		t.Fatalf("parse protocol.go: %v", err)
	}
	var out []string
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		typed := false
		for _, s := range gd.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// Only the first line of an iota group carries the type; the rest
			// inherit it, which is exactly what makes them easy to overlook.
			if sel, ok := vs.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "What" {
				typed = true
			}
			if !typed {
				continue
			}
			for _, id := range vs.Names {
				out = append(out, id.Name)
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no messages found, so the parse is wrong rather than the protocol empty")
	}
	sort.Strings(out)
	return out
}

// messageTypesInSource reads the package for the types that declare a What
// method, which is what makes a type a statemachine.Message.
func messageTypesInSource(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "protocol.go", nil, 0)
	if err != nil {
		t.Fatalf("parse protocol.go: %v", err)
	}
	var out []string
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "What" || fd.Recv == nil || len(fd.Recv.List) != 1 {
			continue
		}
		if id, ok := fd.Recv.List[0].Type.(*ast.Ident); ok {
			out = append(out, id.Name)
		}
	}
	sort.Strings(out)
	return out
}

// contains reports whether the sorted list holds s.
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// ifElse picks one of two strings, so a message can say "should" or "should
// not" without a second Errorf.
func ifElse(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}
