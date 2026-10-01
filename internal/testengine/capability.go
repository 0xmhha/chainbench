package testengine

import (
	"strings"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// attachCapability is the sole capability an attached network advertises: RPC
// reachability. Attach makes no producer/consensus/ws assumptions.
const attachCapability = "rpc"

// applicableTo reports whether a spec applies to chain: an empty or absent
// applicableChains applies to every chain; otherwise chain must appear in the
// comma/space-separated list.
func applicableTo(chain string) func(dsl.Spec) bool {
	return func(s dsl.Spec) bool {
		list := strings.FieldsFunc(s.ApplicableChains, func(r rune) bool { return r == ',' || r == ' ' })
		if len(list) == 0 {
			return true
		}
		for _, c := range list {
			if c == chain {
				return true
			}
		}
		return false
	}
}

// satisfies reports whether every required capability is present in provided.
func satisfies(required, provided []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]bool, len(provided))
	for _, c := range provided {
		set[c] = true
	}
	for _, r := range required {
		if !set[r] {
			return false
		}
	}
	return true
}

// Applicability is what the gate decided about one spec on this target.
type Applicability struct {
	// Runs is whether the spec applies here at all.
	Runs bool
	// Foreseen is whether a spec that does not run said it would not. A spec
	// that declares no skipsOn foresees every skip, which is the behaviour
	// everything had before the field existed. A spec whose only unmet
	// requirement is a target: capability also foresees it — see
	// unmetAreAllTarget.
	Foreseen bool
	// Stale is a spec that declared it would skip on this chain and did not.
	// The declaration outlived what it described, and a reader who trusts it
	// is told this test asks nothing here when it does.
	Stale bool
}

// applicableWithCaps composes chain applicability with capability gating: a spec
// applies only when its chain matches (see applicableTo) and the target network
// provides every capability the spec requires.
//
// A spec that requires a capability the target lacks is still skipped rather
// than failed — that is what makes one corpus runnable against several chains.
// What changed is that a spec may now say where it expects to be skipped, and
// then the gate holds it to that: see dsl.Spec.SkipsOn for why a silent skip is
// worth turning into a decision.
func applicableWithCaps(chain string, provided []string) func(dsl.Spec) Applicability {
	chainOK := applicableTo(chain)
	return func(s dsl.Spec) Applicability {
		runs := chainOK(s) && satisfies(s.Requires, provided)
		if s.SkipsOn == nil {
			return Applicability{Runs: runs, Foreseen: true}
		}
		declared := false
		for _, c := range s.SkipsOn {
			if c == chain {
				declared = true
				break
			}
		}
		// skipsOn names CHAINS, so it cannot say "skipped because this target
		// is not remote" — and registry.CapTarget already states the intent
		// for that case: "asking for it turns a run on the wrong target into a
		// SKIP with a reason". requires is where that was declared, so the
		// declaration exists; it is just not spelled in skipsOn and never can
		// be. Only target: is treated this way. A missing contract:, fork: or
		// engine: capability is a fact about how the network was composed, and
		// a spec vanishing over one of those is the surprise the guard exists
		// for.
		foreseen := declared || (!runs && chainOK(s) && unmetAreAllTarget(s.Requires, provided))
		return Applicability{Runs: runs, Foreseen: foreseen, Stale: runs && declared}
	}
}

// unmetAreAllTarget reports whether every requirement the target does not meet
// is a target: capability. False when nothing is unmet, so a caller cannot read
// "all unmet are target" as "this ran".
func unmetAreAllTarget(required, provided []string) bool {
	set := make(map[string]bool, len(provided))
	for _, c := range provided {
		set[c] = true
	}
	unmet := 0
	for _, r := range required {
		if set[r] {
			continue
		}
		if !strings.HasPrefix(r, registry.CapTarget) {
			return false
		}
		unmet++
	}
	return unmet > 0
}
