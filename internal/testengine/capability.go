package testengine

import (
	"strings"

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

// applicableWithCaps composes chain applicability with capability gating: a spec
// applies only when its chain matches (see applicableTo) and the target network
// provides every capability the spec requires. A spec that requires a capability
// the target lacks is skipped, not failed.
func applicableWithCaps(chain string, provided []string) func(dsl.Spec) bool {
	chainOK := applicableTo(chain)
	return func(s dsl.Spec) bool {
		return chainOK(s) && satisfies(s.Requires, provided)
	}
}
