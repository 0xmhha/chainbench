package netcompose

import (
	"fmt"

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
			return nil, fmt.Errorf("netcompose: remote target needs host and dataRoot")
		}
		creds, err := remoteCreds(s, env)
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
		return nil, fmt.Errorf("netcompose: unknown target kind %q", s.Kind)
	}
}

// remoteCreds assembles SSH credentials from the spec and environment. The env
// user overrides the spec user; auth is a password or a key file; secrets never
// touch the workspace state.
func remoteCreds(s TargetSpec, env func(string) string) (remote.Credentials, error) {
	if env == nil {
		env = func(string) string { return "" }
	}
	user := s.User
	if v := env("CHAINBENCH_REMOTE_USER"); v != "" {
		user = v
	}
	creds := remote.Credentials{User: user, Host: s.Host, Port: s.Port}
	if v := env("CHAINBENCH_REMOTE_PASS"); v != "" {
		creds.Password = v
	}
	if kf := env("CHAINBENCH_REMOTE_KEY_FILE"); kf != "" {
		key, err := remote.LoadPrivateKey(kf)
		if err != nil {
			return remote.Credentials{}, err
		}
		creds.PrivateKey = key
		creds.Passphrase = env("CHAINBENCH_REMOTE_KEY_PASSPHRASE")
	}
	if creds.User == "" {
		return remote.Credentials{}, fmt.Errorf("netcompose: remote target needs a user (--remote-user or CHAINBENCH_REMOTE_USER)")
	}
	if creds.Password == "" && len(creds.PrivateKey) == 0 {
		return remote.Credentials{}, fmt.Errorf("netcompose: remote target needs auth (CHAINBENCH_REMOTE_PASS or CHAINBENCH_REMOTE_KEY_FILE)")
	}
	return creds, nil
}
