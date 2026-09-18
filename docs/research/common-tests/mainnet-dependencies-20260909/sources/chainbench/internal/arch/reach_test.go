package arch

import (
	"sort"
	"testing"
)

// TestEveryExportedSymbolIsNamedBySomething is what survived A8.
//
// The task read "clean up exported symbols with zero external consumers —
// mostly unexport, some delete", on a count of 230, then 896, then 928. The
// count was never of dead code. It was of symbols no OTHER package names with a
// selector, and at least five different things produce that:
//
//   - a method, whose receiver is a value, so no selector can see the call at
//     all (424 of the 928, entirely unmeasurable this way);
//   - a type in an exported signature, which a caller receives and never
//     spells (internal/app alone returns dozens of named result types);
//   - a symbol its own package uses, exported for no caller;
//   - a member of a const group whose siblings are named, which is one
//     vocabulary rather than several symbols;
//   - a symbol only tests reach;
//   - and something nothing names.
//
// Only the last is a deletion, and separating the causes left three of them out
// of 1,227: a context-free Import wrapper nobody called (whose name also
// collided with operation.Import), a Transport interface that described what
// Driver already is and had no implementer, and an EnvNames helper written so a
// surface would not keep its own copy of the list, which no surface ever did.
//
// Separating the causes
// is what the task actually needed, so the remedy it proposed — unexport most
// of them — would have been wrong: internal/core/nodeconfig's 111 Key
// constants are a declared vocabulary of knobs, exported as a set, and some
// members ARE named from outside while the rest are simply knobs nobody has
// overridden yet.
//
// What is worth guarding is the invariant, not the count: nothing exported is
// unreachable from any source. A budget on the rest would fail the next time
// somebody adds a knob, and a ratchet that punishes correct work is one people
// learn to switch off.
func TestEveryExportedSymbolIsNamedBySomething(t *testing.T) {
	got, err := Reach(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}

	counts := map[Use]int{}
	var orphans []string
	for _, r := range got {
		counts[r.Use]++
		if r.Use == UseNowhere {
			orphans = append(orphans, r.Pkg+"."+r.Name+" ("+r.File+")")
		}
	}
	sort.Strings(orphans)
	for _, o := range orphans {
		t.Errorf("%s is exported and named by nothing — give it a caller, unexport it, or delete it", o)
	}

	if len(got) == 0 {
		t.Fatal("the walk found no exported symbols, so it is measuring nothing")
	}
	t.Logf("%d exported non-method symbols: %d reached from another package, %d only from tests, %d named in an exported signature, %d a live const group's members, %d used only inside their own package",
		len(got), counts[UseOutside], counts[UseTest], counts[UseSignature], counts[UseSet], counts[UseInside])
}
