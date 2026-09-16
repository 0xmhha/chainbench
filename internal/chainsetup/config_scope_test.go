package chainsetup

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestConfigOverridesFor_FoldsEveryScopeMostGeneralFirst pins what a node's
// config is built from.
//
// Config used to take "all" and "node<N>" only, while launch already took a
// role. The asymmetry meant "every endpoint archives" had to be written once per
// endpoint, and a node added later silently missed it. The two now take the same
// three forms and fold them in the same order, so a value set for one node beats
// one set for its role, which beats one set for every node.
func TestConfigOverridesFor_FoldsEveryScopeMostGeneralFirst(t *testing.T) {
	w := &Workspace{state: State{ConfigSet: map[string][]string{
		"all":   {"metricsHost=0.0.0.0"},
		"bp":    {"syncMode=full"},
		"en":    {"syncMode=snap"},
		"pn":    {"httpHost=127.0.0.1"},
		"node3": {"syncMode=archive"},
	}}}

	cases := map[string]struct {
		role  node.Role
		index int
		want  string // comma-joined, in application order
	}{
		"producer gets all+bp":             {node.RoleBP, 1, "metricsHost=0.0.0.0,syncMode=full"},
		"endpoint gets all+en":             {node.RoleEN, 2, "metricsHost=0.0.0.0,syncMode=snap"},
		"proxy gets all+pn":                {node.RolePN, 4, "metricsHost=0.0.0.0,httpHost=127.0.0.1"},
		"node3 gets all+en+node3":          {node.RoleEN, 3, "metricsHost=0.0.0.0,syncMode=snap,syncMode=archive"},
		"an unreadable role gets all only": {node.Role("validator"), 5, "metricsHost=0.0.0.0"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := strings.Join(w.configOverridesFor(tc.role, tc.index), ",")
			if got != tc.want {
				t.Errorf("configOverridesFor(%q,%d) = %q, want %q", tc.role, tc.index, got, tc.want)
			}
		})
	}
}

// TestRecordConfigSet_RefusesAScopeNothingWouldRead is the hole this change
// closed.
//
// The scope was not checked at all. A typo stored the overrides under a key no
// node ever looks up, the step reported success, and the node came up with a
// config that silently lacked them — the failure showed later as behaviour, not
// as an error.
func TestRecordConfigSet_RefusesAScopeNothingWouldRead(t *testing.T) {
	for _, scope := range []string{"all", "bp", "en", "pn", "node1", "node12"} {
		w := &Workspace{}
		if err := w.recordConfigSet(scope, []string{"syncMode=full"}); err != nil {
			t.Errorf("scope %q must be accepted: %v", scope, err)
		}
	}
	for _, scope := range []string{"validator", "endpoint", "boot", "nodes", "node0", "", "bp1"} {
		w := &Workspace{}
		if err := w.recordConfigSet(scope, []string{"syncMode=full"}); err == nil {
			t.Errorf("scope %q must be refused: nothing would ever read it", scope)
		}
	}
	// A knob is still checked where it is set, not at render.
	w := &Workspace{}
	if err := w.recordConfigSet("all", []string{"nosuchknob=1"}); err == nil {
		t.Error("an unknown config knob must be refused")
	}
}

// TestSortedScopes_OrdersMostGeneralFirst: recording walks the scopes in the
// order they will be applied, so a reader of chain-record.json sees them in the
// order that decides the outcome, and the walk does not depend on map order.
func TestSortedScopes_OrdersMostGeneralFirst(t *testing.T) {
	got := strings.Join(sortedScopes(map[string][]string{
		"node10": {"a"}, "pn": {"b"}, "all": {"c"}, "node2": {"d"}, "bp": {"e"}, "en": {"f"},
	}), ",")
	if want := "all,bp,en,pn,node10,node2"; got != want {
		t.Errorf("sortedScopes = %q, want %q", got, want)
	}
}
