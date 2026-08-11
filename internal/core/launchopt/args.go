package launchopt

import "fmt"

// entry is one accumulated knob: its value, whether it is boolean, and the
// layer that set it last (the winner).
type entry struct {
	value   string
	boolean bool
	layer   Layer
}

// Args is the accumulating command line. Emission order is first-set order —
// deterministic and stable under overrides: a later layer replacing a value
// keeps the knob's original position, so overriding one value never reshuffles
// the argv.
type Args struct {
	dialect  Dialect
	order    []Key
	vals     map[Key]entry
	problems []error
}

// NewArgs starts an empty argv for one dialect.
func NewArgs(d Dialect) *Args {
	return &Args{dialect: d, vals: map[Key]entry{}}
}

// Set records a valued knob. An unsupported key is a classified problem —
// requested features are never silently skipped (the T3.2b discipline).
func (a *Args) Set(k Key, v string, l Layer) {
	name, ok := a.dialect.Spelling(k)
	if !ok {
		a.problems = append(a.problems,
			fmt.Errorf("launchopt: dialect %s does not support %q (layer %s)", a.dialect.ID, k, l))
		return
	}
	if a.dialect.IsBool(k) {
		a.problems = append(a.problems,
			fmt.Errorf("launchopt: %s (%q) is boolean; use Enable (layer %s)", name, k, l))
		return
	}
	a.put(k, entry{value: v, layer: l})
}

// SetIfSupported records a valued knob only when the dialect has it. This is
// the "harmless absence" branch of the tri-state rule: use it for knobs whose
// absence the generation covers by default, never to smuggle a feature past a
// dialect that lacks it.
func (a *Args) SetIfSupported(k Key, v string, l Layer) {
	if _, ok := a.dialect.Spelling(k); !ok {
		return
	}
	a.Set(k, v, l)
}

// Enable records a boolean knob.
func (a *Args) Enable(k Key, l Layer) {
	name, ok := a.dialect.Spelling(k)
	if !ok {
		a.problems = append(a.problems,
			fmt.Errorf("launchopt: dialect %s does not support %q (layer %s)", a.dialect.ID, k, l))
		return
	}
	if !a.dialect.IsBool(k) {
		a.problems = append(a.problems,
			fmt.Errorf("launchopt: %s (%q) takes a value; use Set (layer %s)", name, k, l))
		return
	}
	a.put(k, entry{boolean: true, layer: l})
}

// EnableIfSupported is Enable under the harmless-absence rule (see
// SetIfSupported).
func (a *Args) EnableIfSupported(k Key, l Layer) {
	if _, ok := a.dialect.Spelling(k); !ok {
		return
	}
	a.Enable(k, l)
}

func (a *Args) put(k Key, e entry) {
	if _, seen := a.vals[k]; !seen {
		a.order = append(a.order, k)
	}
	a.vals[k] = e
}

// Has reports whether the knob has been set.
func (a *Args) Has(k Key) bool {
	_, ok := a.vals[k]
	return ok
}

// Value returns the recorded value for a valued knob ("" when unset or
// boolean).
func (a *Args) Value(k Key) string { return a.vals[k].value }

// WonBy returns the layer that set the knob last ("" when unset). This is the
// provenance surface for `net status`.
func (a *Args) WonBy(k Key) Layer { return a.vals[k].layer }

// Problems returns every classified problem accumulated so far. The Builder
// joins them into one error, so a broken assembly reports all defects at once.
func (a *Args) Problems() []error { return a.problems }

// Argv renders the accumulated knobs in first-set order.
func (a *Args) Argv() []string {
	out := make([]string, 0, len(a.order)*2)
	for _, k := range a.order {
		e := a.vals[k]
		name, _ := a.dialect.Spelling(k)
		out = append(out, name)
		if !e.boolean {
			out = append(out, e.value)
		}
	}
	return out
}
