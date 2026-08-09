// Package testkit is the chainbench test HELPER module (requirement #16),
// deliberately separate from the test CODE under tests/. It provides: the Case
// type and registry test authors register into, the T handle a case uses to
// drive a live NodeSet (RPC clients, assertions, waits), and the Report the
// runner produces. Test cases are runtime scenarios against a running network,
// not compile-time go-test units — so they register via init() and are executed
// by internal/core/pipeline/testrun (docs/CHAINBENCH_GO_REDESIGN.md §9).
//
// Legacy: this Go-func case model is being retired in favor of the declarative
// DSL path (internal/testspec parsed and run by internal/engine, reached via
// `chainbench run`). The result model (Report/Result/Status) is reused by the
// new path; the Case/registry authoring surface is what goes away once the
// suites under tests/ are ported. See docs/dev/legacy-retirement-plan.md.
package testkit

// CaseFunc is a single test scenario. It drives the network through t and
// reports failure via t's assertions.
type CaseFunc func(t *T)

// Case is one registered test. Gating fields let the runner skip cases that do
// not apply to the target chain or lack a required capability (requirement #12,
// #16), replacing the bash chain_compat / requires_capabilities frontmatter.
type Case struct {
	// Name uniquely identifies the case (kebab-case), e.g. "validator-set".
	Name string
	// Category groups cases, e.g. "consensus", "tx", "fault".
	Category string
	// ChainCompat lists the chain ids the case applies to. Empty = all chains.
	ChainCompat []string
	// RequiresCaps lists capabilities the target NodeSet must expose (e.g.
	// "consensus"). All must be present or the case is skipped.
	RequiresCaps []string
	// Fn is the scenario body.
	Fn CaseFunc
}

// AppliesTo reports whether the case runs on the given chain (per ChainCompat).
func (c Case) AppliesTo(chain string) bool {
	if len(c.ChainCompat) == 0 {
		return true
	}
	for _, ch := range c.ChainCompat {
		if ch == chain {
			return true
		}
	}
	return false
}
