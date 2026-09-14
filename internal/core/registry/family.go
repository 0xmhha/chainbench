package registry

import (
	"fmt"
	"sort"
)

// Consensus families register themselves the way chains do, and for the same
// reason.
//
// A chain names its family as a string in its manifest, so something has to
// turn that string into an implementation. That was a switch in the package
// that loads an external manifest, which meant two things: adding a family
// meant editing a function in a package that has nothing to do with it, and
// that package had to import every family to name them — the "branch on the
// target instead of asking it" shape this codebase removes everywhere else.
//
// A family registers itself, so the set is whatever was linked in, and the
// blank import that pulls in a chain pulls in the family it composes.

// families is the registered set, keyed by the id a manifest names.
var families = map[string]ConsensusFamily{}

// RegisterFamily adds a consensus family. Intended to be called from a family
// package's init(); panics on a duplicate or empty id so a wiring mistake fails
// at startup rather than resolving to whichever registration ran last.
func RegisterFamily(f ConsensusFamily) {
	if f == nil {
		panic("registry: nil consensus family")
	}
	id := f.ID()
	if id == "" {
		panic("registry: consensus family with empty id")
	}
	if _, dup := families[id]; dup {
		panic(fmt.Sprintf("registry: duplicate consensus family %q", id))
	}
	families[id] = f
}

// FamilyByName returns the family registered under id, or an error naming the
// set that is linked in. A family that is not there is not a typo in the
// manifest by definition: it may simply not be part of this build.
func FamilyByName(id string) (ConsensusFamily, error) {
	if f, ok := families[id]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("registry: unknown consensus family %q (linked in: %v)", id, FamilyNames())
}

// FamilyNames returns the sorted ids of the registered families.
func FamilyNames() []string {
	names := make([]string, 0, len(families))
	for n := range families {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
