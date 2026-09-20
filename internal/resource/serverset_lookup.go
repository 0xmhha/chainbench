package resource

import (
	"fmt"
	"sort"
)

// Picking a server out of the set, and filling in what an entry left unsaid.
//
// An entry inherits the set's defaults rather than repeating them, so a set
// where every host shares one SSH login says that once.

func (p Ports) HasMetrics() bool { return p.RPCStep >= metricsRPCStep }

// label names a server for messages: its name, else its index, else its
// position in the file.
func (s Server) label(i int) string {
	switch {
	case s.Name != "":
		return s.Name
	case s.Index != 0:
		return fmt.Sprintf("#%d", s.Index)
	default:
		return fmt.Sprintf("at position %d", i+1)
	}
}

// IsRemote reports whether this server's nodes run over SSH.
func (s Server) IsRemote() bool { return s.Kind == KindRemote }

// Server returns the server with the given index, fully resolved against the
// file defaults. It errors, listing what is available, if none matches.
func (c *Set) Server(index int) (Server, error) {
	for _, s := range c.Servers {
		if s.Index == index && index != 0 {
			return c.resolve(s), nil
		}
	}
	return Server{}, fmt.Errorf("serverset: no server with index %d in %s (available: %v)", index, c.path, c.indexes())
}

// ByName returns the server with the given name, fully resolved.
func (c *Set) ByName(name string) (Server, error) {
	for _, s := range c.Servers {
		if s.Name == name && name != "" {
			return c.resolve(s), nil
		}
	}
	return Server{}, fmt.Errorf("serverset: no server named %q in %s (available: %v)", name, c.path, c.names())
}

// Select resolves a server by name when one is given, otherwise by index, and
// otherwise the only server in a single-server set. It is what a command
// with an optional --server flag calls.
func (c *Set) Select(name string, index int) (Server, error) {
	switch {
	case name != "":
		return c.ByName(name)
	case index != 0:
		return c.Server(index)
	case len(c.Servers) == 1:
		return c.resolve(c.Servers[0]), nil
	default:
		return Server{}, fmt.Errorf("serverset: %s has %d servers — name one with --server (available: %v)",
			c.path, len(c.Servers), c.names())
	}
}

// resolve fills a server's omitted fields from the file defaults, and any still
// unset from the built-in defaults, so callers never see a zero port plan.
func (c *Set) resolve(s Server) Server {
	if s.Kind == "" {
		s.Kind = KindLocal
	}
	if s.Slots == 0 {
		s.Slots = c.Defaults.Slots
	}
	if s.Slots == 0 {
		s.Slots = 1
	}
	s.Ports = s.Ports.inherit(c.Defaults.Ports).inherit(BuiltinPorts())
	s.SSH = s.SSH.inherit(c.Defaults.SSH)
	return s
}

// inherit fills p's zero fields from other.
func (p Ports) inherit(other Ports) Ports {
	if p.P2PBase == 0 {
		p.P2PBase = other.P2PBase
	}
	if p.P2PStep == 0 {
		p.P2PStep = other.P2PStep
	}
	if p.RPCBase == 0 {
		p.RPCBase = other.RPCBase
	}
	if p.RPCStep == 0 {
		p.RPCStep = other.RPCStep
	}
	return p
}

// inherit fills s's zero fields from other.
func (s SSH) inherit(other SSH) SSH {
	if s.User == "" {
		s.User = other.User
	}
	if s.Port == 0 {
		s.Port = other.Port
	}
	if s.Password == "" {
		s.Password = other.Password
	}
	if s.KeyFile == "" {
		s.KeyFile = other.KeyFile
	}
	return s
}

// indexes lists the configured indexes in ascending order for error messages.
func (c *Set) indexes() []int {
	out := make([]int, 0, len(c.Servers))
	for _, s := range c.Servers {
		if s.Index != 0 {
			out = append(out, s.Index)
		}
	}
	sort.Ints(out)
	return out
}

// names lists the configured names in order for error messages.
func (c *Set) names() []string {
	out := make([]string, 0, len(c.Servers))
	for _, s := range c.Servers {
		if s.Name != "" {
			out = append(out, s.Name)
		}
	}
	sort.Strings(out)
	return out
}

// Credentials builds the SSH credentials for a remote server. The server set
// file is the single source: a server named there is reached exactly as the
// file says, and the environment is never consulted — an exported variable
// left over from another environment must not silently redirect a login.
// Secrets can still stay out of the file itself: password_file and
// key_passphrase_file reference a separate file (0600, one line), which is
// also the shape a secret manager renders to disk. It errors if the server is
// local (there is nothing to authenticate to) or if no user or auth resolves.
