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
// appear in the syntax — they come from the environment at resolve time.

package target

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// TargetKind is where a network's data plane lives — this machine, or a remote
// SSH host.
type TargetKind string

const (
	// TargetLocal materializes files on the local filesystem and runs processes
	// locally. The empty kind is treated as local.
	TargetLocal TargetKind = "local"
	// TargetRemote materializes files and runs processes on a remote host over
	// SSH.
	TargetRemote TargetKind = "remote"
	// TargetServer names a host indirectly, by its entry in the operator's
	// server inventory. Resolving one needs that inventory (see ResolveWith).
	TargetServer TargetKind = "server"
)

// TargetSpec is the serializable descriptor of the compose target (persisted in
// workspace.json). It carries NO secrets: remote SSH credentials are read from
// the environment (CHAINBENCH_REMOTE_USER / _PASS / _KEY_FILE / _KEY_PASSPHRASE)
// only when the live Target is resolved. DataRoot is the network's data-root
// path ON the target (a local path, or a path on the remote host).
type TargetSpec struct {
	Kind     TargetKind `json:"kind"`
	DataRoot string     `json:"dataRoot"`
	Host     string     `json:"host,omitempty"`
	User     string     `json:"user,omitempty"`
	Port     int        `json:"port,omitempty"`
	// Server is the inventory entry name for a TargetServer spec. It is the
	// whole address: host, port and credentials stay in the inventory, so
	// neither a command line nor a persisted spec carries them.
	Server string `json:"server,omitempty"`
}

// IsRemote reports whether the target is on another host — named directly or
// through the inventory.
func (s TargetSpec) IsRemote() bool { return s.Kind == TargetRemote || s.Kind == TargetServer }

// ParseTarget parses the single-path target syntax, so that every layer above
// names a location the same way whether it is here or on another machine:
//
//	/data/net1                        this machine
//	srv://bp1/data/net1               the inventory entry "bp1"
//	user@host:/data/net1              remote over SSH (port 22 / env)
//	ssh://user@host:2222/data/net1    remote with an explicit port
//
// Prefer srv:// for anything that gets typed or scripted. The other two remote
// forms put a host address in the command line and in shell history, which is
// exactly what keeping the inventory out of the repository is meant to avoid;
// srv:// names an entry and lets the inventory hold the address.
//
// Parsing does no I/O: an srv:// spec records the entry name, and looking that
// name up happens in ResolveWith. A bare local path stays bare — writing
// localhost:/path for a local file would make the common case look like an
// exception.
//
// Credentials never appear in the syntax — they come from the environment or
// the inventory when the target is resolved.
func ParseTarget(s string) (TargetSpec, error) {
	if s == "" {
		return TargetSpec{}, fmt.Errorf("target: empty target")
	}
	if rest, ok := strings.CutPrefix(s, "srv://"); ok {
		name, path, ok := strings.Cut(rest, "/")
		if !ok || name == "" || path == "" {
			return TargetSpec{}, fmt.Errorf("target: bad target %q (want srv://<server>/path)", s)
		}
		return TargetSpec{Kind: TargetServer, Server: name, DataRoot: "/" + path}, nil
	}
	if strings.HasPrefix(s, "ssh://") {
		u, err := url.Parse(s)
		if err != nil {
			return TargetSpec{}, fmt.Errorf("target: bad target %q: %w", s, err)
		}
		if u.Host == "" || u.Path == "" {
			return TargetSpec{}, fmt.Errorf("target: target %q needs a host and a path", s)
		}
		port := 0
		if p := u.Port(); p != "" {
			n, err := strconv.Atoi(p)
			if err != nil {
				return TargetSpec{}, fmt.Errorf("target: bad port in target %q", s)
			}
			port = n
		}
		return TargetSpec{
			Kind: TargetRemote, Host: u.Hostname(), User: u.User.Username(),
			Port: port, DataRoot: u.Path,
		}, nil
	}
	if spec, ok := parseHostColonPath(s); ok {
		return spec, nil
	}
	if strings.Contains(s, "@") {
		return TargetSpec{}, fmt.Errorf("target: bad target %q (want [user@]host:/path)", s)
	}
	return TargetSpec{Kind: TargetLocal, DataRoot: s}, nil
}

// parseHostColonPath recognises the scp-style [user@]host:/path form.
//
// The colon is ambiguous — a local path may legally contain one — so the form
// is only accepted when it cannot be a local path: the host part must be
// non-empty and free of slashes, and the path part must be absolute. That
// leaves "./notes:draft/x" alone while still accepting "10.0.0.1:/keys/node1".
func parseHostColonPath(s string) (TargetSpec, bool) {
	rest, user := s, ""
	if u, r, ok := strings.Cut(s, "@"); ok {
		user, rest = u, r
	}
	host, path, ok := strings.Cut(rest, ":")
	if !ok || host == "" || strings.Contains(host, "/") || !strings.HasPrefix(path, "/") {
		return TargetSpec{}, false
	}
	if user == "" && strings.Contains(s, "@") {
		return TargetSpec{}, false
	}
	return TargetSpec{Kind: TargetRemote, Host: host, User: user, DataRoot: path}, true
}

// Target is the resolved data plane: step functions materialize files through
// Files and run processes through Driver at DataRoot, without branching on local
// vs remote. This is the one seam that hides the location difference.
type Target struct {
	Spec     TargetSpec
	DataRoot string
	Files    provision.FileStore
	Driver   driver.Driver
}

// Inventory turns a server-inventory entry name into SSH credentials.
//
// It is a function rather than a package dependency on purpose: parsing a path
// must not require reading the operator's inventory file, and this package must
// not know the inventory's format. The caller that already loaded the inventory
// supplies the lookup.
type Inventory func(name string, env func(string) string) (remote.Credentials, error)

// Resolve builds the live Target from its spec, without an inventory. An
// srv:// spec is rejected with an error naming what is missing rather than
// being silently treated as something else — see ResolveWith.
func (s TargetSpec) Resolve(env func(string) string) (*Target, error) {
	return s.ResolveWith(env, nil)
}

// ResolveWith builds the live Target, using inv to look up an srv:// entry.
//
// A local target uses the local filesystem and driver; the remote forms open an
// SSH-backed FileStore and driver. Credentials come from env for a directly
// named host and from the inventory for an srv:// entry. The host-key policy
// comes from remote.ResolveHostKeyCallback (known_hosts, or the loud insecure
// opt-in) in both cases.
func (s TargetSpec) ResolveWith(env func(string) string, inv Inventory) (*Target, error) {
	return s.ResolveWithMap(env, inv, nil)
}

// ResolveWithMap is ResolveWith with a dial-time address translation (--docker:
// local containers posing as servers). The map touches only the credentials the
// SSH transport dials; the spec, the data root, and everything composed onto
// the target keep the real addresses. Nil is no translation.
func (s TargetSpec) ResolveWithMap(env func(string) string, inv Inventory, m remote.AddrMap) (*Target, error) {
	switch s.Kind {
	case "", TargetLocal:
		return &Target{Spec: s, DataRoot: s.DataRoot, Files: provision.LocalFileStore{}, Driver: driver.NewLocalDriver()}, nil

	case TargetServer:
		if inv == nil {
			return nil, fmt.Errorf("target: %q names a server inventory entry, but no inventory was provided", s.Server)
		}
		creds, err := inv(s.Server, env)
		if err != nil {
			return nil, err
		}
		return s.resolveOver(creds, env, m)

	case TargetRemote:
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

// resolveOver builds the SSH-backed target for already-resolved credentials.
// Both remote forms end here, so they cannot drift apart — and so the dial-time
// address translation cannot be applied to one form and missed on the other.
func (s TargetSpec) resolveOver(creds remote.Credentials, env func(string) string, m remote.AddrMap) (*Target, error) {
	if s.DataRoot == "" {
		return nil, fmt.Errorf("target: remote target needs a path")
	}
	if m != nil {
		creds.Host, creds.Port = m(creds.Host, creds.Port)
	}
	hostKey, err := remote.ResolveHostKeyCallback(env)
	if err != nil {
		return nil, err
	}
	run := driver.SSHRunner(creds, hostKey)
	return &Target{
		Spec: s, DataRoot: s.DataRoot,
		Files: driver.NewRemoteFileStore(run), Driver: driver.NewRemoteDriver(run),
	}, nil
}
