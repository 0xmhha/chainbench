// Holding the plan and the launched network to each other.
//
// Everything else this package checks asks whether the DECLARATION is
// coherent. Nothing asked whether the network that came up is the one the
// declaration described, and the last word belongs to the command line: an
// override that names no layer wins over everything a document says, so a
// merge can be perfect and the nodes still run something else.
//
// A test against the wrong network does not fail — it answers a question
// nobody asked. So a mismatch here stops the run.

package testengine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// LaunchMismatch is one thing the plan asked for that the launched network
// does not show.
type LaunchMismatch struct {
	// Where is the node this is about, or "" for a fact about the network.
	Where string
	Want  string
	Got   string
}

func (m LaunchMismatch) String() string {
	where := "the network"
	if m.Where != "" {
		where = m.Where
	}
	return fmt.Sprintf("%s: asked for %s, launched with %s", where, m.Want, m.Got)
}

// VerifyLaunched compares the plan with what the workspace recorded once the
// network was up.
//
// It compares only facts both sides state outright. A port or a data directory
// is derived by the composer and has nothing in the plan to disagree with; the
// launch knobs are the opposite, and are the reason this exists — a knob the
// declaration asked for that never reached an argv is invisible everywhere
// else.
//
// A handoff plans no layout of its own, so there is nothing here to hold it to.
func VerifyLaunched(plan ComposePlan, st chainsetup.State) []LaunchMismatch {
	if plan.Handoff != nil {
		return nil
	}
	var out []LaunchMismatch
	if st.Chain != plan.Chain {
		out = append(out, LaunchMismatch{Want: "chain " + plan.Chain, Got: "chain " + st.Chain})
	}
	if plan.Keys.Dir != "" && st.KeysDir != plan.Keys.Dir {
		out = append(out, LaunchMismatch{
			Want: "keys " + plan.Keys.Dir + askedBy(plan, FieldKeysDir),
			Got:  "keys " + st.KeysDir,
		})
	}
	out = append(out, roleCountMismatches(plan, st)...)
	out = append(out, launchKnobMismatches(plan, st)...)
	return out
}

// roleCountMismatches reports a node table that is not the shape the plan said.
//
// A plan whose bp count fills from the server set states a floor, not a number,
// so only a table smaller than the floor is a mismatch.
func roleCountMismatches(plan ComposePlan, st chainsetup.State) []LaunchMismatch {
	got := map[node.Role]int{}
	for _, n := range st.Nodes {
		got[node.Role(n.Role)]++
	}
	var out []LaunchMismatch
	add := func(role node.Role, want int, f PlanField) {
		have := got[role]
		if want == have {
			return
		}
		if role == node.RoleBP && plan.Nodes.AutoSize && have >= want {
			return
		}
		out = append(out, LaunchMismatch{
			Want: fmt.Sprintf("%d %s%s", want, role, askedBy(plan, f)),
			Got:  fmt.Sprintf("%d", have),
		})
	}
	add(node.RoleBP, plan.Nodes.BP, FieldNodesBP)
	add(node.RoleEN, plan.Nodes.EN, FieldNodesEN)
	add(node.RolePN, plan.Nodes.PN, FieldNodesPN)
	return out
}

// launchKnobMismatches reports a launch knob the plan listed that is missing
// from the argv of a node its scope covers.
//
// Only presence is checked, and only for knobs written as a flag name. A value
// is rendered by the dialect (one chain's --mine is another's absent flag), so
// comparing values here would be comparing this package's idea of a spelling
// against the one that actually assembles argv — two copies of one fact, which
// is the thing this track keeps removing.
//
// The message names who asked for the knob. Whoever reads it has to go change
// something, and the declaration and the command line are different places to
// go.
func launchKnobMismatches(plan ComposePlan, st chainsetup.State) []LaunchMismatch {
	var out []LaunchMismatch
	for _, scope := range sortedScopes(plan.Launch) {
		for _, knob := range plan.Launch[scope] {
			name := knobName(knob.Knob)
			if name == "" {
				continue
			}
			for _, n := range st.Nodes {
				if !scopeCovers(scope, n) || len(n.Args) == 0 {
					continue
				}
				if argvNames(n.Args)[name] {
					continue
				}
				out = append(out, LaunchMismatch{
					Where: n.Label,
					Want:  fmt.Sprintf("launch %s (scope %s, asked by the %s)", knob.Knob, scope, knob.From),
					Got:   "an argv without it",
				})
			}
		}
	}
	return out
}

// askedBy is the phrase naming who chose a value, or "" when the plan does not
// record a source for that field.
//
// A mismatch is read by someone who has to go change something, so it says
// where to go. It stays silent rather than guessing: a wrong address sends them
// to edit a file that says nothing about the value.
func askedBy(plan ComposePlan, f PlanField) string {
	src, ok := plan.From[f]
	if !ok {
		return ""
	}
	return ", asked by the " + string(src)
}

// scopeCovers reports whether a launch scope applies to a node.
//
// node.ScopeFor owns that rule — the scopes one node answers to, most general
// first — and asking it is how this stays right when the vocabulary grows.
func scopeCovers(scope string, n node.Record) bool {
	for _, s := range node.ScopeFor(node.Role(n.Role), n.Index) {
		if s == scope {
			return true
		}
	}
	return false
}

// knobName is the flag name a knob asks for, without its value. A knob is
// "key=value" or a bare "key"; the key is a dot path (chain.bootnodes), and
// only its last segment ever reaches argv as a flag name.
func knobName(knob string) string {
	key, _, _ := strings.Cut(knob, "=")
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if i := strings.LastIndex(key, "."); i >= 0 {
		key = key[i+1:]
	}
	return key
}

// argvNames is the set of flag names an argv carries, without their values and
// without the leading dashes. A dotted flag (--rpc.allow-unprotected-txs) is
// also recorded by its last segment, because that is how a knob names it.
func argvNames(args []string) map[string]bool {
	out := map[string]bool{}
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimLeft(a, "-"), "=")
		out[name] = true
		if i := strings.LastIndex(name, "."); i >= 0 {
			out[name[i+1:]] = true
		}
	}
	return out
}

// sortedScopes orders scopes most general first, whatever the scope holds.
func sortedScopes[T any](m map[string][]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := node.ScopeRank(out[i]), node.ScopeRank(out[j])
		if ri != rj {
			return ri < rj
		}
		return out[i] < out[j]
	})
	return out
}
