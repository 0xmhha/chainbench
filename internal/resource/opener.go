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
package resource

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/remote"
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
func (o Opener) Open(spec Spec) (*Access, error) {
	env := o.Env
	if env == nil {
		env = func(string) string { return "" }
	}
	m, err := o.AddrMap()
	if err != nil {
		return nil, err
	}
	return spec.ResolveWithPolicy(env, SetLookup(o.ServerSet), m, SetPolicy(o.ServerSet))
}

// HTTPEndpoint returns the URL a caller should dial to reach the HTTP service a
// node advertises at host:port.
//
// The caller passes the address it knows — the node's own, as the composition
// recorded it — and this decides how that address is actually reached: a docker
// container's address is translated through the localmap (the same translation
// SSH credentials get inside resolveOver), a remote server's address is not in
// the localmap and is dialled as given, and a local node is already its own
// address. So no caller above this layer needs to know a localmap exists, or
// pass a flag saying whether one applies — which is what let one dial site
// (the preflight liveness probe) miss the translation and read a live docker
// network as dead.
//
// Docker mode without a readable localmap is an error rather than an
// untranslated dial: the flag is the power switch, so a missing mapping file is
// a refusal that names the fix instead of a silent dial at an address nothing
// answers on.
func (o Opener) HTTPEndpoint(host string, port int) (string, error) {
	m, err := o.AddrMap()
	if err != nil {
		return "", err
	}
	if m != nil {
		host, port = m(host, port)
	}
	return fmt.Sprintf("http://%s:%d", host, port), nil
}

// AddrMap returns the dial-time address translation this opener applies — nil
// without docker mode. Prefer HTTPEndpoint for a node dial: it applies this map
// so the caller does not have to hold it. This stays exported for the resolve
// path, which threads the map into credentials.
func (o Opener) AddrMap() (remote.AddrMap, error) {
	if !o.Docker {
		return nil, nil
	}
	lm, err := LoadLocalMap(LocalMapNear(o.ServerSet))
	if err != nil {
		return nil, err
	}
	return lm.AddrMap(func(from, to string) { o.report("docker: dialing %s as %s", from, to) }), nil
}

// OpenPath parses a target path (plain path, srv://<server>/path,
// [user@]host:/path, ssh://…) and opens it.
func (o Opener) OpenPath(path string) (*Access, error) {
	spec, err := Parse(path)
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
