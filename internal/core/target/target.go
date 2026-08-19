// Package target owns the single-path syntax that lets every layer name a
// location without branching on local versus remote: a bare path is this
// machine, user@host:/path is an SSH host.
//
// It lives at the primitive layer because the callers span the whole stack —
// the composition steps, the app use cases, and the keyring, which reads key
// material from wherever it already exists. Parsing a path must not require
// importing an orchestration package (worklist §1g F2).
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
}

// IsRemote reports whether the target is a remote SSH host.
func (s TargetSpec) IsRemote() bool { return s.Kind == TargetRemote }

// ParseTarget parses the single-path target syntax (key point 2: the upper
// layers see one "path", local or remote):
//
//	/data/net1                        local path
//	user@host:/data/net1              remote over SSH (port 22 / env)
//	ssh://user@host:2222/data/net1    remote with an explicit port
//
// Credentials never appear in the syntax — they come from the environment
// when the target is resolved (see TargetSpec).
func ParseTarget(s string) (TargetSpec, error) {
	if s == "" {
		return TargetSpec{}, fmt.Errorf("target: empty target")
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
	if user, rest, ok := strings.Cut(s, "@"); ok {
		host, path, ok := strings.Cut(rest, ":")
		if !ok || host == "" || path == "" {
			return TargetSpec{}, fmt.Errorf("target: bad target %q (want user@host:/path)", s)
		}
		return TargetSpec{Kind: TargetRemote, Host: host, User: user, DataRoot: path}, nil
	}
	return TargetSpec{Kind: TargetLocal, DataRoot: s}, nil
}

// Target is the resolved data plane: step functions materialize files through
// Sink and run processes through Driver at DataRoot, without branching on local
// vs remote. This is the one seam that hides the location difference.
type Target struct {
	Spec     TargetSpec
	DataRoot string
	Sink     provision.FileSink
	Driver   driver.Driver
}

// Resolve builds the live Target from its spec. A local target uses the local
// filesystem and driver; a remote target opens an SSH-backed FileSink and driver,
// reading credentials from env. The host-key policy comes from
// remote.ResolveHostKeyCallback (known_hosts, or the loud insecure opt-in).
func (s TargetSpec) Resolve(env func(string) string) (*Target, error) {
	switch s.Kind {
	case "", TargetLocal:
		return &Target{Spec: s, DataRoot: s.DataRoot, Sink: provision.LocalFileSink{}, Driver: driver.NewLocalDriver()}, nil
	case TargetRemote:
		if s.Host == "" || s.DataRoot == "" {
			return nil, fmt.Errorf("target: remote target needs host and dataRoot")
		}
		creds, err := remote.CredentialsFromEnv(s.User, s.Host, s.Port, env)
		if err != nil {
			return nil, err
		}
		hostKey, err := remote.ResolveHostKeyCallback(env)
		if err != nil {
			return nil, err
		}
		run := driver.SSHRunner(creds, hostKey)
		return &Target{Spec: s, DataRoot: s.DataRoot, Sink: driver.NewRemoteFileSink(run), Driver: driver.NewRemoteDriver(run)}, nil
	default:
		return nil, fmt.Errorf("target: unknown target kind %q", s.Kind)
	}
}
