// Package target owns the single-path syntax that lets every layer name a
// location without branching on local versus remote: a bare path is this
// machine, user@host:/path is an SSH host.
//
// It lives at the primitive layer because the callers span the whole stack —
// the composition steps, the app use cases, and the keyring, which reads key
// material from wherever it already exists. Parsing a path must not require
// importing an orchestration package.
//
// Resolve turns a spec into the live pair a caller actually works through: a
// FileStore for the files and a Driver for the processes. Credentials never
// appear in the syntax — at resolve time a server-set entry reads them from
// the server set (the single source for a named server), and a directly named
// host reads the environment.

package machine

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Kind is where a network's data plane lives — this machine, or a remote
// SSH host.
type Kind string

const (
	// KindLocal materializes files on the local filesystem and runs processes
	// locally. The empty kind is treated as local.
	KindLocal Kind = "local"
	// KindRemote materializes files and runs processes on a remote host over
	// SSH.
	KindRemote Kind = "remote"
	// KindServer names a host indirectly, by its entry in the operator's
	// server set. Resolving one needs that server set (see ResolveWith).
	KindServer Kind = "server"
)

// Spec is the serializable descriptor of the compose target (persisted in
// workspace.json). It carries NO secrets: a server-set target reads its login
// from the server set at resolve time, and a directly named host reads the
// environment (CHAINBENCH_REMOTE_USER / _PASS / _KEY_FILE / _KEY_PASSPHRASE).
// DataRoot is the network's data-root path ON the target (a local path, or a
// path on the remote host).
type Spec struct {
	Kind     Kind   `json:"kind"`
	DataRoot string `json:"dataRoot"`
	Host     string `json:"host,omitempty"`
	User     string `json:"user,omitempty"`
	Port     int    `json:"port,omitempty"`
	// Server is the server-set entry name for a KindServer spec. It is the
	// whole address: host, port and credentials stay in the server set, so
	// neither a command line nor a persisted spec carries them.
	Server string `json:"server,omitempty"`
}

// IsRemote reports whether the target is on another host — named directly or
// through the server set.
func (s Spec) IsRemote() bool { return s.Kind == KindRemote || s.Kind == KindServer }

// Parse parses the single-path target syntax, so that every layer above
// names a location the same way whether it is here or on another machine:
//
//	/data/net1                        this machine
//	srv://bp1/data/net1               the server-set entry "bp1"
//	user@host:/data/net1              remote over SSH (port 22 / env)
//	ssh://user@host:2222/data/net1    remote with an explicit port
//
// Prefer srv:// for anything that gets typed or scripted. The other two remote
// forms put a host address in the command line and in shell history, which is
// exactly what keeping the server set out of the repository is meant to avoid;
// srv:// names an entry and lets the server set hold the address.
//
// Parsing does no I/O: an srv:// spec records the entry name, and looking that
// name up happens in ResolveWith. A bare local path stays bare — writing
// localhost:/path for a local file would make the common case look like an
// exception.
//
// Credentials never appear in the syntax — they come from the environment or
// the server set when the target is resolved.
func Parse(s string) (Spec, error) {
	if s == "" {
		return Spec{}, fmt.Errorf("target: empty target")
	}
	if rest, ok := strings.CutPrefix(s, "srv://"); ok {
		name, path, ok := strings.Cut(rest, "/")
		if !ok || name == "" || path == "" {
			return Spec{}, fmt.Errorf("target: bad target %q (want srv://<server>/path)", s)
		}
		return Spec{Kind: KindServer, Server: name, DataRoot: "/" + path}, nil
	}
	if strings.HasPrefix(s, "ssh://") {
		u, err := url.Parse(s)
		if err != nil {
			return Spec{}, fmt.Errorf("target: bad target %q: %w", s, err)
		}
		if u.Host == "" || u.Path == "" {
			return Spec{}, fmt.Errorf("target: target %q needs a host and a path", s)
		}
		port := 0
		if p := u.Port(); p != "" {
			n, err := strconv.Atoi(p)
			if err != nil {
				return Spec{}, fmt.Errorf("target: bad port in target %q", s)
			}
			port = n
		}
		return Spec{
			Kind: KindRemote, Host: u.Hostname(), User: u.User.Username(),
			Port: port, DataRoot: u.Path,
		}, nil
	}
	if spec, ok := parseHostColonPath(s); ok {
		return spec, nil
	}
	if strings.Contains(s, "@") {
		return Spec{}, fmt.Errorf("target: bad target %q (want [user@]host:/path)", s)
	}
	return Spec{Kind: KindLocal, DataRoot: s}, nil
}

// parseHostColonPath recognises the scp-style [user@]host:/path form.
//
// The colon is ambiguous — a local path may legally contain one — so the form
// is only accepted when it cannot be a local path: the host part must be
// non-empty and free of slashes, and the path part must be absolute. That
// leaves "./notes:draft/x" alone while still accepting "10.0.0.1:/keys/node1".
func parseHostColonPath(s string) (Spec, bool) {
	rest, user := s, ""
	if u, r, ok := strings.Cut(s, "@"); ok {
		user, rest = u, r
	}
	host, path, ok := strings.Cut(rest, ":")
	if !ok || host == "" || strings.Contains(host, "/") || !strings.HasPrefix(path, "/") {
		return Spec{}, false
	}
	if user == "" && strings.Contains(s, "@") {
		return Spec{}, false
	}
	return Spec{Kind: KindRemote, Host: host, User: user, DataRoot: path}, true
}

// Access is the resolved data plane: step functions materialize files through
// Files and run processes through Driver at DataRoot, without branching on local
// vs remote. This is the one seam that hides the location difference.
type Access struct {
	Spec     Spec
	DataRoot string
	Files    provision.FileStore
	Driver   driver.Driver
}

// Lookup turns a server-set entry name into SSH credentials.
//
// It is a function rather than a package dependency on purpose: parsing a path
// must not require reading the operator's server set, and this package must
// not know the server set's format. The caller that already loaded the server set
// supplies the lookup. It takes no environment on purpose — the server set is
// the single source of a named server's login.
type Lookup func(name string) (remote.Credentials, error)

// Resolve builds the live Access from its spec, without a server set. An
// srv:// spec is rejected with an error naming what is missing rather than
// being silently treated as something else — see ResolveWith.
func (s Spec) Resolve(env func(string) string) (*Access, error) {
	return s.ResolveWith(env, nil)
}

// ResolveWith builds the live Access, using inv to look up an srv:// entry from the server set.
//
// A local target uses the local filesystem and driver; the remote forms open an
// SSH-backed FileStore and driver. Credentials come from env for a directly
// named host and from the server set for an srv:// entry. The host-key policy
// comes from remote.ResolveHostKeyCallback (known_hosts, or the loud insecure
// opt-in) in both cases.
func (s Spec) ResolveWith(env func(string) string, inv Lookup) (*Access, error) {
	return s.ResolveWithMap(env, inv, nil)
}

// ResolveWithMap is ResolveWith with a dial-time address translation (--docker:
// local containers posing as servers). The map touches only the credentials the
// SSH transport dials; the spec, the data root, and everything composed onto
// the target keep the real addresses. Nil is no translation.
func (s Spec) ResolveWithMap(env func(string) string, inv Lookup, m remote.AddrMap) (*Access, error) {
	switch s.Kind {
	case "", KindLocal:
		return &Access{Spec: s, DataRoot: s.DataRoot, Files: provision.LocalFileStore{}, Driver: driver.NewLocalDriver()}, nil

	case KindServer:
		if inv == nil {
			return nil, fmt.Errorf("target: %q names a server-set entry, but no server set was provided", s.Server)
		}
		creds, err := inv(s.Server)
		if err != nil {
			return nil, err
		}
		return s.resolveOver(creds, env, m)

	case KindRemote:
		if s.Host == "" || s.DataRoot == "" {
			return nil, fmt.Errorf("target: remote target needs host and dataRoot")
		}
		creds, err := remote.CredentialsFromEnv(s.User, s.Host, s.Port, env)
		if err != nil {
			return nil, err
		}
		return s.resolveOver(creds, env, m)

	default:
		return nil, fmt.Errorf("target: unknown target kind %q", s.Kind)
	}
}

// mapCredentials applies the dial-time address translation to SSH credentials.
// The default port is resolved BEFORE mapping: a map keyed on 22 must match a
// dial that was going to use 22. Left at zero, the directly named host form
// slipped past the map and dialed the default port on the substitute address.
func mapCredentials(creds remote.Credentials, m remote.AddrMap) remote.Credentials {
	if m == nil {
		return creds
	}
	if creds.Port == 0 {
		creds.Port = remote.DefaultSSHPort
	}
	creds.Host, creds.Port = m(creds.Host, creds.Port)
	return creds
}

// resolveOver builds the SSH-backed target for already-resolved credentials.
// Both remote forms end here, so they cannot drift apart — and so the dial-time
// address translation cannot be applied to one form and missed on the other.
func (s Spec) resolveOver(creds remote.Credentials, env func(string) string, m remote.AddrMap) (*Access, error) {
	if s.DataRoot == "" {
		return nil, fmt.Errorf("target: remote target needs a path")
	}
	creds = mapCredentials(creds, m)
	hostKey, err := remote.ResolveHostKeyCallback(env)
	if err != nil {
		return nil, err
	}
	run := driver.SSHRunner(creds, hostKey)
	return &Access{
		Spec: s, DataRoot: s.DataRoot,
		Files: driver.NewRemoteFileStore(run), Driver: driver.NewRemoteDriver(run),
	}, nil
}

// Validate reports whether the spec is structurally complete for its kind —
// what resolving will need, checked before any dial. Consumers call this
// instead of inspecting the kind themselves: what a remote spec requires is
// this module's knowledge.
func (s Spec) Validate() error {
	switch s.Kind {
	case "", KindLocal:
		return nil
	case KindRemote:
		if s.Host == "" || s.DataRoot == "" {
			return fmt.Errorf("target: remote target needs host and dataRoot")
		}
		return nil
	case KindServer:
		if s.Server == "" || s.DataRoot == "" {
			return fmt.Errorf("target: server target needs a server name and dataRoot")
		}
		return nil
	default:
		return fmt.Errorf("target: unknown target kind %q", s.Kind)
	}
}
