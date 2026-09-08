package main

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
)

// find walks a command tree to the command at path, or nil.
func find(root *cobra.Command, path ...string) *cobra.Command {
	c := root
	for _, want := range path {
		var next *cobra.Command
		for _, sub := range c.Commands() {
			if sub.Name() == want {
				next = sub
				break
			}
		}
		if next == nil {
			return nil
		}
		c = next
	}
	return c
}

// TestQuery_IsTheSameRegistration is S7's gate: `query keyring list` and
// `keyring list` are one registration rendered twice, not two spellings that
// happen to agree today.
//
// What it catches is a projection built by CONSTRUCTING the command a second
// time. That is the plausible mistake — a query group assembled by calling the
// same New* functions again looks right in help output and is a genuinely
// separate registration, with its own flag variables, free to drift.
//
// Setting a flag through the projection and reading it back through the
// canonical spelling is what tells the two apart. A shallow copy of the flag
// would NOT fail this: pflag.Flag holds its Value as an interface over the
// bound variable, so copying the struct still shares the variable. Measured
// rather than assumed — the first version of this comment claimed copying could
// not share, and a probe showed it does.
func TestQuery_IsTheSameRegistration(t *testing.T) {
	root := newRootCmd()
	canonical := find(root, "keyring", "list")
	projected := find(root, "query", "keyring", "list")
	if canonical == nil || projected == nil {
		t.Fatalf("keyring list = %v, query keyring list = %v", canonical, projected)
	}
	if projected == canonical {
		t.Fatal("the projection is the same object, which breaks its own command path")
	}
	if projected.Short != canonical.Short {
		t.Errorf("descriptions differ:\n  %q\n  %q", canonical.Short, projected.Short)
	}

	// The flags are shared, not copied. Set one through the projection and the
	// canonical command sees it — that is what makes them one registration.
	if projected.Flags().Lookup("json") == nil {
		t.Fatal("the projection did not carry the command's flags")
	}
	if err := projected.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set --json on the projection: %v", err)
	}
	got, err := canonical.Flags().GetBool("json")
	if err != nil || !got {
		t.Errorf("the canonical command did not see the projection's flag (got %v, %v) — they are two flag sets, so they can drift", got, err)
	}
}

// TestQuery_HoldsExactlyWhatDeclaredItself is the other half of the gate: a
// command that did not declare itself read-only must not appear, and one that
// did must not be missing.
//
// Both directions matter. A leak puts something that writes inside the group an
// operator was told is safe to explore. A gap makes the projection a partial
// list, which is worse than no list — the reader concludes the missing command
// is not a query.
func TestQuery_HoldsExactlyWhatDeclaredItself(t *testing.T) {
	root := newRootCmd()
	declared := ReadOnlyPaths(root)
	// The projection's own paths, with the "query " prefix removed.
	var projected []string
	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		here := append(append([]string(nil), path...), c.Name())
		if c.Runnable() {
			projected = append(projected, strings.Join(here[1:], " "))
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "help" {
				walk(sub, here)
			}
		}
	}
	q := find(root, "query")
	if q == nil {
		t.Fatal("there is no query group")
	}
	walk(q, nil)
	sort.Strings(projected)

	// declared holds "query keyring list" too, since the projection carries the
	// annotation; drop that half before comparing.
	var canonical []string
	for _, p := range declared {
		if !strings.HasPrefix(p, "query ") {
			canonical = append(canonical, p)
		}
	}
	if len(canonical) == 0 {
		t.Fatal("no command declared itself read-only, so this test asserts nothing")
	}
	if strings.Join(canonical, "\n") != strings.Join(projected, "\n") {
		t.Errorf("the projection does not match the declarations\n declared: %v\nprojected: %v", canonical, projected)
	}
	t.Logf("%d commands declare themselves read-only, and query holds exactly those", len(canonical))
}

// TestReadOnly_DoesNotCoverAWriteOrASecret is a sanity check on the
// declarations themselves.
//
// The property is declared rather than inferred, and a declaration can be
// wrong. No walk of the code can prove one right — `node rpc` takes the method
// as an argument — but a verb that plainly writes, or a command that exists to
// print a secret, is a mistake worth catching before it reaches a group an
// operator is told is safe.
func TestReadOnly_DoesNotCoverAWriteOrASecret(t *testing.T) {
	writes := []string{
		"new", "add", "import", "set", "send", "fund", "deploy", "start", "stop",
		"restart", "rm", "clean", "init", "up", "run", "resume", "swap", "export",
		"keys", "genesis", "config", "build", "place", "provision", "hardfork",
	}
	for _, path := range ReadOnlyPaths(newRootCmd()) {
		if strings.HasPrefix(path, "query ") {
			continue
		}
		leaf := path[strings.LastIndex(path, " ")+1:]
		for _, w := range writes {
			if leaf == w {
				t.Errorf("%q declares itself read-only, but %q is a verb that writes — check the declaration", path, w)
			}
		}
	}
}

// TestReadOnly_MarksTheCommandItIsGiven keeps the helper honest: a command it
// touched must answer yes, and one it did not must answer no.
func TestReadOnly_MarksTheCommandItIsGiven(t *testing.T) {
	plain := &cobra.Command{Use: "x"}
	if surface.IsReadOnly(plain) {
		t.Error("an unmarked command claimed to be read-only")
	}
	if !surface.IsReadOnly(surface.ReadOnly(&cobra.Command{Use: "y"})) {
		t.Error("a marked command did not answer")
	}
}
