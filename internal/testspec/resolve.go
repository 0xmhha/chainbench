package testspec

import "sort"

// Unresolved returns the action and assertion names a spec references that are
// not registered in reg, prefixed "action:" or "assert:" (and sorted, de-duped).
// A spec names its steps and assertions by string, so an unknown name would fail
// only at run time; Unresolved surfaces those offline. An empty result means
// every reference resolves against reg.
func Unresolved(s Spec, reg Registry) []string {
	seen := map[string]bool{}
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
		}
	}

	checkAction := func(entry map[string]any) {
		name := actionName(entry)
		if name == "" {
			add("action:(empty)")
			return
		}
		if _, ok := reg.Action(name); !ok {
			add("action:" + name)
		}
	}
	for _, a := range s.PreActions {
		checkAction(a)
	}
	for _, st := range s.Steps {
		checkAction(st)
	}
	for _, po := range s.PostActions {
		checkAction(po)
	}
	for _, as := range s.Assertions {
		name, _ := as["assert"].(string)
		if name == "" {
			add("assert:(missing)")
			continue
		}
		if _, ok := reg.Assertion(name); !ok {
			add("assert:" + name)
		}
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
