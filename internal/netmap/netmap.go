// Package netmap is the module that knows where things run: it manages the
// server set, allocates hosts and ports to nodes, composes enode addresses
// from placements, and — through Opener — is the one place a server name is
// bound to a live connection.
//
// The pure allocation core (roles, labels, ports, peering) lives in
// core/netmap; the server-set format lives in serverset. This package binds
// them to the low level, so no other module wires a dial itself. That rule
// exists because wiring diverged when every consumer did it alone: one
// passed the server set, another passed nil, and the same server accepted
// one login and refused the other (found live, 2026-08-25).
package netmap

import (
	"github.com/0xmhha/chainbench/internal/core/machine"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/serverset"
)

// Opener binds a machine spec to the server set and the docker translation,
// and opens it into capability handles. It is the single dial-wiring point:
// every consumer that reaches a server goes through here, so the server-set
// lookup, the --docker address translation, and the translation report cannot
// diverge between modules.
//
// The zero value opens with the default server-set location, no docker
// translation, and no reporting.
type Opener struct {
	// ServerSet is the server-set file consulted for srv:// names; empty uses
	// the default location.
	ServerSet string
	// Docker treats the servers as local docker containers: dials are
	// translated through the localmap next to the server set. The option is
	// the power switch — with it, a missing localmap is an error; without it,
	// a leftover localmap changes nothing.
	Docker bool
	// Env supplies environment lookups for the directly-named host form
	// (user@host:/path), which has no server set to consult. Nil reads none.
	Env func(string) string
	// Report hears operational side notes as they happen — today, the dial
	// translations docker mode applies. Nil discards them.
	Report func(format string, args ...any)
}

// Open resolves spec into live capability handles (files, driver). Local
// specs come back with local handles through the same path — the consumer
// never branches on where the machine is.
func (o Opener) Open(spec machine.Spec) (*machine.Access, error) {
	env := o.Env
	if env == nil {
		env = func(string) string { return "" }
	}
	var m remote.AddrMap
	if o.Docker {
		lm, err := serverset.LoadLocalMap(serverset.LocalMapNear(o.ServerSet))
		if err != nil {
			return nil, err
		}
		m = lm.AddrMap(func(from, to string) { o.report("docker: dialing %s as %s", from, to) })
	}
	return spec.ResolveWithMap(env, serverset.SetLookup(o.ServerSet), m)
}

// OpenPath parses a target path (plain path, srv://<server>/path,
// [user@]host:/path, ssh://…) and opens it.
func (o Opener) OpenPath(path string) (*machine.Access, error) {
	spec, err := machine.Parse(path)
	if err != nil {
		return nil, err
	}
	return o.Open(spec)
}

func (o Opener) report(format string, args ...any) {
	if o.Report != nil {
		o.Report(format, args...)
	}
}
