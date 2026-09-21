package chainsetup

import (
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// What a composition is given from outside.
//
// It lives here rather than beside the verbs because it is not the verbs'. The
// workspace takes it too — a baseline reads the clock, a lock names the command
// — so a file holding only verbs would be one that everything below it imports.

// Deps is what a composition needs from its caller: a clock for step stamps, an
// environment, a command line for the workspace lock's owner note, and a
// reporter for operational side notes. All may be zero — the defaults are
// time.Now, the process environment, an empty owner, and silence.
type Deps struct {
	Clock   func() time.Time
	Env     func(string) string
	Command string
	Report  func(format string, args ...any)
	// Driver overrides the transport every machine of a workspace controls
	// its nodes through; nil uses each machine's own process. Injected for
	// tests and for surfaces that route the same verb over another transport.
	Driver func() (process.Driver, error)
}

// Now is the clock the doc above promises, with its default applied. Callers
// outside this package need the same default — a nil Clock is ordinary, and
// every caller that reached for d.Clock() directly was one nil away from a
// panic.
func (d Deps) Now() time.Time {
	if d.Clock == nil {
		return time.Now()
	}
	return d.Clock()
}

func (d Deps) command() string { return d.Command }

func (d Deps) logf(format string, args ...any) {
	if d.Report != nil {
		d.Report(format, args...)
	}
}
