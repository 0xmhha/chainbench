package testspec

import "sort"

// Unresolved returns the references a spec makes that nothing satisfies:
// action and assertion names not registered in reg (prefixed "action:" /
// "assert:"), read sources that name no reader (prefixed "source:"), and
// binding references no earlier step saved (prefixed "ref:"). Results are
// sorted and de-duplicated.
//
// A spec names its steps, assertions, and saved values by string, so every one
// of these would otherwise fail only at run time — against a live chain, after a
// network has been brought up. Unresolved surfaces them offline instead, which
// is what `chainbench validate` reports.
//
// Binding references are checked in execution order (pre-actions, then steps,
// then assertions, then post-actions), so a reference to a value saved *later*
// is reported too — the interpreter would not have it bound yet.
func Unresolved(s Spec, reg Registry) []string {
	seen := map[string]bool{}
	bound := map[string]bool{}

	// checkAction validates one action entry and records what it saves.
	checkAction := func(entry map[string]any) {
		name := actionName(entry)
		if name == "" {
			seen["action:(empty)"] = true
			return
		}
		if _, ok := reg.Action(name); !ok {
			seen["action:"+name] = true
		}
		args := argsOf(entry[name])
		checkRefs(args, bound, seen)
		// A read action names its source by string too, so an unknown one would
		// only surface once a network is up. Catch it here with the rest.
		if name == ActionRead {
			source, _ := args["source"].(string)
			if source == "" {
				seen["source:(missing)"] = true
			} else if _, ok := reg.Reader(source); !ok {
				seen["source:"+source] = true
			}
		}
		if save := saveName(args); save != "" {
			bound[save] = true
		}
		// newAccount binds a second name (the generated private key) via "saveKey".
		if sk, _ := args["saveKey"].(string); sk != "" {
			bound[sk] = true
		}
	}

	for _, a := range s.PreActions {
		checkAction(a)
	}
	for _, st := range s.Steps {
		checkAction(st)
	}
	for _, as := range s.Assertions {
		name, _ := as["assert"].(string)
		if name == "" {
			seen["assert:(missing)"] = true
		} else if _, ok := reg.Assertion(name); !ok {
			seen["assert:"+name] = true
		}
		checkRefs(as, bound, seen)
	}
	for _, po := range s.PostActions {
		checkAction(po)
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// checkRefs records every binding reference in v that is not yet bound.
func checkRefs(v any, bound, seen map[string]bool) {
	for _, name := range refNames(v) {
		if !bound[name] {
			seen["ref:"+name] = true
		}
	}
}
