package statemachine

import "testing"

// bandWidth is how much room one machine's four bands need: the private band
// starts at +0x100 and the events from below at +0x180, so a base has to own at
// least 0x200.
const bandWidth = 0x200

// TestBasesDoNotOverlap is what this file is for.
//
// A child machine's messages pass through its controller's queue, so two
// machines numbering from bases that overlap would have one machine answering
// the other's message — and the concrete type would match, because both sides
// switch on What. The reference keeps the same distance for the same reason
// (IpClient gives DHCPv4 the 1000s and DHCPv6 the 2000s).
func TestBasesDoNotOverlap(t *testing.T) {
	bases := map[string]What{
		"BaseChain": BaseChain,
		"BaseTest":  BaseTest,
	}
	for name, base := range bases {
		if base <= 0 {
			t.Errorf("%s is %d, and a base has to be positive so a message's band can be read off it", name, base)
		}
		for other, ob := range bases {
			if other == name {
				continue
			}
			if d := base - ob; d > -bandWidth && d < bandWidth {
				t.Errorf("%s (%#x) and %s (%#x) are within %#x of each other, so their bands collide",
					name, base, other, ob, bandWidth)
			}
		}
	}
}
