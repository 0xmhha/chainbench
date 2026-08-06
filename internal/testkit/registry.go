package testkit

import "sync"

var (
	regMu    sync.Mutex
	registry []Case
)

// Register adds a test case to the global registry. Test files call this from
// init(). Duplicate names are allowed to register but the runner reports on
// each; keep names unique by convention.
func Register(c Case) {
	regMu.Lock()
	defer regMu.Unlock()
	registry = append(registry, c)
}

// Cases returns a copy of the registered cases.
func Cases() []Case {
	regMu.Lock()
	defer regMu.Unlock()
	out := make([]Case, len(registry))
	copy(out, registry)
	return out
}
