// Package app is the use-case layer between the two user surfaces (CLI, MCP)
// and the orchestration/core packages (worklist T7.5,
// structure-and-atomic-cli-proposal §2.2).
//
// One use case = one function. Inputs and outputs are plain structs that both
// cobra flag binding and MCP JSON-schema binding can target, so the two
// surfaces call the same function and cannot drift apart. Nothing here knows
// cobra or MCP types, and nothing here formats output — rendering (tabwriter,
// JSON, MCP text) is the surface's job.
package app

import "time"

// Deps are the collaborators the use cases share, injected once at the surface
// boundary. No package-level state.
type Deps struct {
	// Clock supplies timestamps (workspace step stamps); nil uses time.Now.
	Clock func() time.Time
	// Env reads environment variables (remote-target credentials at resolve
	// time); nil uses os.Getenv. Injected so tests never depend on the process
	// environment.
	Env func(string) string
}
