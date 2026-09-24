package verb

import (
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"strings"
	"testing"
)

// TestRecordConfigSet_RefusesAScopeNothingWouldRead is the hole this change
// closed.
//
// The scope was not checked at all. A typo stored the overrides under a key no
// node ever looks up, the step reported success, and the node came up with a
// config that silently lacked them — the failure showed later as behaviour, not
// as an error.
func TestRecordConfigSet_RefusesAScopeNothingWouldRead(t *testing.T) {
	for _, scope := range []string{"all", "bp", "en", "pn", "node1", "node12"} {
		w := &chainsetup.Workspace{}
		if err := w.RecordConfigSet(scope, []string{"syncMode=full"}); err != nil {
			t.Errorf("scope %q must be accepted: %v", scope, err)
		}
	}
	for _, scope := range []string{"validator", "endpoint", "boot", "nodes", "node0", "", "bp1"} {
		w := &chainsetup.Workspace{}
		if err := w.RecordConfigSet(scope, []string{"syncMode=full"}); err == nil {
			t.Errorf("scope %q must be refused: nothing would ever read it", scope)
		}
	}
	// A knob is still checked where it is set, not at render.
	w := &chainsetup.Workspace{}
	if err := w.RecordConfigSet("all", []string{"nosuchknob=1"}); err == nil {
		t.Error("an unknown config knob must be refused")
	}
}

// TestSortedScopes_OrdersMostGeneralFirst: recording walks the scopes in the
// order they will be applied, so a reader of chain-record.json sees them in the
// order that decides the outcome, and the walk does not depend on map order.
func TestSortedScopes_OrdersMostGeneralFirst(t *testing.T) {
	got := strings.Join(chainsetup.SortedScopes(map[string][]string{
		"node10": {"a"}, "pn": {"b"}, "all": {"c"}, "node2": {"d"}, "bp": {"e"}, "en": {"f"},
	}), ",")
	if want := "all,bp,en,pn,node10,node2"; got != want {
		t.Errorf("sortedScopes = %q, want %q", got, want)
	}
}
