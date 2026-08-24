// Package obs is chainbench's observability core (requirement #17): structured
// logging, a runtime event bus, and a state/result store. It is deliberately
// SEPARATE from the test helper (pkg/testkit): the setup and verify phases also
// emit runtime records, so observation is not a test-only concern. testkit
// consumes obs to build reports; it does not own storage
// (docs/CHAINBENCH_GO_REDESIGN.md §8).
//
// The event bus feeds the dashboard (requirement #19) via chainbench-dashboard; its wire
// format is deferred to the dashboard phase (G8).
package obs

import "time"

// Phase is one of the three pipeline phases an event belongs to.
type Phase string

const (
	PhaseSetup  Phase = "setup"
	PhaseVerify Phase = "verify"
	PhaseTest   Phase = "test"
)

// Kind is a coarse event category, kept open (a string) so chains and drivers
// can emit their own kinds without a central enum.
type Kind string

const (
	KindInfo     Kind = "info"
	KindProgress Kind = "progress"
	KindResult   Kind = "result"
	KindError    Kind = "error"
)

// Event is one runtime observation. Fields carries structured, event-specific
// data (node index, block number, durations, ...). Time is stamped by the Bus
// at publish if left zero.
type Event struct {
	Time    time.Time      `json:"time"`
	Phase   Phase          `json:"phase"`
	Kind    Kind           `json:"kind"`
	Network string         `json:"network,omitempty"`
	Node    int            `json:"node,omitempty"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}
