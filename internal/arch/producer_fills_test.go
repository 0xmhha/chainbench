package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// producerFills holds the rule this ratchet exists for: when one function is the
// sole producer of a struct another function decides on, every field the decider
// can read has to be filled — or listed here with why it is not.
//
// Three defects in one week were this exact shape, and none of them looked like a
// bug in a diff. The field, the comparison and the condition were all written and
// all tested; nothing supplied the value, so the check was inert from the day it
// shipped.
//
//   - preflight.Want.GenesisDeclared — Have filled a genesis hash, Want left it
//     empty, and the comparison skips when either side is empty. Two environments
//     differing only in their overlay reused one network and the second test ran
//     against the first one's chain.
//   - nodemonitor.Facts.WantPeers — the classifier refuses a node with no peer,
//     and the observer never said how many were wanted. One validator of four sat
//     at height 0 while the gate called the network ready.
//   - nodemonitor.Facts.Forked — health answers whether the nodes are on one
//     chain and the classifier calls a split fatal. Nothing carried the answer
//     across, so a forked network gated as ready.
//
// A grep cannot make this judgement, which is why the rule is scoped to named
// pairs rather than swept over every struct: a request type bound from CLI flags
// by reflection has no assignment in the source at all, so "never assigned" is
// true of it and means nothing. These pairs are hand-written structs handed from
// one function to another, where the producer is the only writer.
//
// The list grows when a new decision struct appears. The exceptions may only
// shrink.
var producerFills = []struct {
	Type     string            // the struct, as the producer spells it
	Decl     string            // file declaring it
	Producer string            // file:function that is its only writer
	Unset    map[string]string // field -> why the producer leaves it empty
}{
	{
		Type:     "preflight.Want",
		Decl:     "../core/preflight/preflight.go",
		Producer: "../chainsetup/steps_preflight.go:WantOf",
		Unset: map[string]string{
			"Nodes": "per-node facts are pinned by the caller that has them (a spec's topology), not invented from a request that names only counts",
		},
	},
	{
		Type:     "nodemonitor.Facts",
		Decl:     "../nodemonitor/classify.go",
		Producer: "../testengine/nodegate.go:factsFromReport",
		Unset: map[string]string{
			"WantChainID":     "the classifier calls a mismatch FATAL, so the wanted value must be authoritative: reading it from the observation is circular, the workspace records the chain's name rather than its numeric id, and taking whichever node answered first would make a fatal verdict depend on iteration order",
			"WantParticipate": "Participating comes from the collector and this observer has no source for it; requiring it would hold every network until the wait budget ran out",
			"Participating":   "same missing source — the observer reads health, which does not report sealing",
			"Failure":         "a launch/bring-up failure mode is classified by process inspection, which this observer does not do; the gate's restart adapter owns that path",
		},
	},
}

// TestEveryFieldAProducerOwnsIsFilledOrExplained is the ratchet. For each pair it
// compares the struct's fields against the ones the producer assigns, and requires
// the difference to be accounted for in writing.
func TestEveryFieldAProducerOwnsIsFilledOrExplained(t *testing.T) {
	for _, rule := range producerFills {
		t.Run(rule.Type, func(t *testing.T) {
			short := rule.Type
			if i := strings.LastIndex(short, "."); i >= 0 {
				short = short[i+1:]
			}
			declared := structFields(t, rule.Decl, short)
			if len(declared) == 0 {
				t.Fatalf("%s declares no fields in %s — the rule names the wrong type or file", rule.Type, rule.Decl)
			}
			parts := strings.SplitN(rule.Producer, ":", 2)
			if len(parts) != 2 {
				t.Fatalf("Producer %q must be file:function", rule.Producer)
			}
			filled := literalFields(t, parts[0], parts[1], short)

			for _, f := range declared {
				if filled[f] {
					if _, excused := rule.Unset[f]; excused {
						t.Errorf("%s.%s is assigned by %s and also listed as unset — remove the entry, it teaches the next reader to skip the list", rule.Type, f, rule.Producer)
					}
					continue
				}
				if reason := rule.Unset[f]; reason == "" {
					t.Errorf("%s.%s is never filled by %s — a field a decision reads and nobody writes makes the decision inert; fill it or say here why it stays empty", rule.Type, f, rule.Producer)
				}
			}
			for f := range rule.Unset {
				if !contains(declared, f) {
					t.Errorf("Unset names %s.%s, which the struct does not declare — the field was renamed or removed", rule.Type, f)
				}
			}
			t.Logf("%d fields, %d filled by %s, %d explained", len(declared), len(filled), parts[1], len(rule.Unset))
		})
	}
}

func contains(all []string, want string) bool {
	for _, s := range all {
		if s == want {
			return true
		}
	}
	return false
}

// structFields returns the exported field names a struct declares.
func structFields(t *testing.T, path, name string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Clean(path), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != name {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, fld := range st.Fields.List {
			for _, id := range fld.Names {
				if id.IsExported() {
					out = append(out, id.Name)
				}
			}
		}
		return false
	})
	sort.Strings(out)
	return out
}

// literalFields returns the field names a function assigns in a composite literal
// of the named type. Keyed fields only: a positional literal of a decision struct
// would be unreadable and none exists.
func literalFields(t *testing.T, path, fn, typeName string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Clean(path), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, d := range f.Decls {
		decl, ok := d.(*ast.FuncDecl)
		if !ok || decl.Name.Name != fn {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			sel, ok := lit.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != typeName {
				return true
			}
			for _, el := range lit.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if id, ok := kv.Key.(*ast.Ident); ok {
					out[id.Name] = true
				}
			}
			return true
		})
	}
	if len(out) == 0 {
		t.Fatalf("%s:%s assigns no %s fields — the rule names the wrong function", path, fn, typeName)
	}
	return out
}
