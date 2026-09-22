package nodeconfig

import (
	"sort"
	"strings"
	"testing"
)

// TestEveryDialectSpellsTheNetworkIDFlagTheSameWay is what lets the uniformity
// check read one flag name out of any node's argv.
//
// The check is handed assembled command lines and no dialect, because a network
// of mixed builds has one dialect per node and the thing being checked is that
// they agree. That is only sound while every dialect writes the same flag. If a
// generation ever spells it differently this fails, and the check has to be
// given each node's dialect rather than guessing.
func TestEveryDialectSpellsTheNetworkIDFlagTheSameWay(t *testing.T) {
	names := make([]string, 0, len(dialects))
	for name := range dialects {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("no dialects registered, so the parse is wrong rather than the table empty")
	}
	for _, name := range names {
		d := dialects[name]()
		got, ok := d.Spelling(KeyNetworkID)
		if !ok {
			t.Errorf("dialect %s has no spelling for the network id", name)
			continue
		}
		if got != flagNetworkID {
			t.Errorf("dialect %s spells the network id %q, and the uniformity check looks for %q", name, got, flagNetworkID)
		}
	}
}

func TestValidateUniformNetworkID(t *testing.T) {
	argv := func(id string) []string {
		return []string{"--datadir", "/d", "--networkid", id, "--http"}
	}
	cases := []struct {
		name string
		in   map[string][]string
		want string // substring the error must name; "" means it must pass
	}{
		{
			name: "one network, one number",
			in:   map[string][]string{"node1": argv("8285"), "node2": argv("8285")},
		},
		{
			name: "a successor left on its own build's number",
			in:   map[string][]string{"node1": argv("8284"), "node5": argv("8285")},
			want: "will not peer",
		},
		{
			name: "a node assembled without the flag takes its build's default",
			in:   map[string][]string{"node1": argv("8285"), "node2": {"--datadir", "/d"}},
			want: "own default",
		},
		{
			name: "a flag with no value is not a value",
			in:   map[string][]string{"node1": {"--networkid"}},
			want: "own default",
		},
		{
			name: "nothing to check is the wrong question",
			in:   map[string][]string{},
			want: "no argv",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateUniformNetworkID(c.in)
			switch {
			case c.want == "" && err != nil:
				t.Errorf("should pass, got %v", err)
			case c.want != "" && err == nil:
				t.Errorf("should fail naming %q, passed", c.want)
			case c.want != "" && !strings.Contains(err.Error(), c.want):
				t.Errorf("error does not name %q: %v", c.want, err)
			}
		})
	}
}

// TestValidateUniformNetworkID_NamesTheSamePairEveryTime: the message has to be
// stable, because a check whose wording moves between runs reads as two
// different failures to whoever is bisecting.
func TestValidateUniformNetworkID_NamesTheSamePairEveryTime(t *testing.T) {
	in := map[string][]string{
		"node3": {"--networkid", "1"},
		"node1": {"--networkid", "2"},
		"node2": {"--networkid", "3"},
	}
	first := ValidateUniformNetworkID(in)
	if first == nil {
		t.Fatal("three different ids should fail")
	}
	for i := 0; i < 20; i++ {
		if got := ValidateUniformNetworkID(in); got.Error() != first.Error() {
			t.Fatalf("message changed between runs:\n  %v\n  %v", first, got)
		}
	}
	if !strings.Contains(first.Error(), "node1") || !strings.Contains(first.Error(), "node2") {
		t.Errorf("the message should name the lowest pair that disagrees, got %v", first)
	}
}
