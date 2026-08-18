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

import (
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
)

// Deps are the collaborators the use cases share, injected once at the surface
// boundary. No package-level state.
type Deps struct {
	// Clock supplies timestamps (workspace step stamps, session age); nil uses
	// time.Now.
	Clock func() time.Time
	// Env reads environment variables (remote-target credentials at resolve
	// time); nil uses os.Getenv. Injected so tests never depend on the process
	// environment.
	Env func(string) string
	// Driver resolves the transport used to control node processes of an
	// already-launched network; nil uses the local driver. Injected so the use
	// cases can be tested without spawning processes, and so a surface that
	// targets a remote host routes the same use case over SSH.
	Driver func() (driver.Driver, error)
}

// now reports the current time through the injected clock.
func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Now()
	}
	return d.Clock()
}

// nodeDriver resolves the transport for node-process control, defaulting to
// this machine.
func (d Deps) nodeDriver() (driver.Driver, error) {
	if d.Driver == nil {
		return driver.NewLocalDriver(), nil
	}
	return d.Driver()
}
