// Package app is the use-case layer between the two user surfaces (CLI, MCP)
// and the orchestration/core packages.
//
// One use case = one function. Inputs and outputs are plain structs that both
// cobra flag binding and MCP JSON-schema binding can target, so the two
// surfaces call the same function and cannot drift apart. Nothing here knows
// cobra or MCP types, and nothing here formats output — rendering (tabwriter,
// JSON, MCP text) is the surface's job.
package app

import (
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
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
	// already-launched network; nil uses the local process. Injected so the use
	// cases can be tested without spawning processes, and so a surface that
	// targets a remote host routes the same use case over SSH.
	Driver func() (process.Driver, error)
	// Files resolves where a network's on-disk material lands; nil uses this
	// machine's filesystem.
	//
	// It is separate from Driver because the two can differ — a launch driven
	// over SSH still reads its keys from here — and because leaving it implicit
	// is how a remote provision came to write its genesis and configs to the
	// operator's own disk while shipping only the identities.
	Files func() (filestore.Store, error)
	// Command is what the operator typed, recorded in the workspace lock so a
	// run that finds the workspace busy can say what is using it. Injected
	// rather than read from os.Args: a use case must not depend on how the
	// process was started.
	Command string
	// Logf reports operational side notes a caller should see as they happen —
	// today, the dial-address translations --docker applies. Nil discards them.
	// It is not a result channel: anything a caller acts on belongs in the
	// use case's returned value.
	Logf func(format string, args ...any)
}

// command is what the operator typed, or a placeholder when nothing was
// injected (an MCP call, a test).
func (d Deps) command() string {
	if d.Command == "" {
		return "(command not recorded)"
	}
	return d.Command
}

// now reports the current time through the injected clock.
func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Now()
	}
	return d.Clock()
}
