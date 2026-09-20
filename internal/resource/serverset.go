package resource

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// The server set: which machines a network may be composed on, how to reach
// them, and which port bands a node may take.
//
// It declares a POOL rather than a list of servers — addresses and how many
// port slots each may hold — so one host with several slots and many hosts with
// one slot each are the same grid described two ways.
//
// Loading and validating a set is in serverset_load.go, looking one up and
// inheriting defaults in serverset_lookup.go, and resolving a login into
// credentials in serverset_credentials.go.

const DefaultSetFile = "server-set.yaml"

// DefaultSampleFile is the tracked template an operator copies.
const DefaultSampleFile = "server-set.sample.yaml"

// defaultSSHPort is the SSH port assumed when neither server nor defaults set it.
const defaultSSHPort = 22

// Port-plan floors. They are not style preferences: the wemix binary derives its
// etcd port as p2p+1, so a p2p step of 1 makes etcd collide with the next node's
// p2p and block production stalls with no obvious cause. The rpc step must cover
// http/ws/auth, and one more slot buys the metrics endpoint.
const (
	minP2PStep     = 2
	minRPCStep     = 3
	metricsRPCStep = 4
)

// Kind is where a server's nodes actually run.
type Kind string

const (
	// KindLocal runs nodes on this  It is the zero value, so an entry
	// that says nothing about access is local.
	KindLocal Kind = "local"
	// KindRemote runs nodes on another host over SSH.
	KindRemote Kind = "remote"
)

// Ports is a server's port plan: the two disjoint bands node ports are assigned
// from. Zero fields inherit the file defaults, and any still zero fall back to
// the built-in defaults (Defaults.Ports) so a server set can name only what it
// needs to move.
type Ports struct {
	// P2PBase is the first node's devp2p port; P2PStep is the per-node stride.
	P2PBase int `yaml:"p2pBase,omitempty"`
	P2PStep int `yaml:"p2pStep,omitempty"`
	// RPCBase is the first node's HTTP port; RPCStep is the per-node stride.
	// ws = http+1, auth = http+2, and metrics = http+3 when the step allows —
	// unless the per-purpose bands below override the derivation.
	RPCBase int `yaml:"rpcBase,omitempty"`
	RPCStep int `yaml:"rpcStep,omitempty"`
	// WS, Auth, and Metrics are per-purpose bands (BandsSpec); nil derives.
	WS      *BandSpec `yaml:"-"`
	Auth    *BandSpec `yaml:"-"`
	Metrics *BandSpec `yaml:"-"`
}

// SSH is how a remote server is reached. The server set is the single source
// of these values; the loader never echoes them. An operator who keeps the set
// file free of secrets writes password_file / key_passphrase_file instead,
// pointing at a separate one-line file (0600) — the shape a secret manager
// renders to disk.
type SSH struct {
	User     string `yaml:"user,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Password string `yaml:"password,omitempty"`
	// PasswordFile names a file holding the password. Exactly one of Password
	// and PasswordFile may be set.
	PasswordFile string `yaml:"password_file,omitempty"`
	KeyFile      string `yaml:"key_file,omitempty"`
	// KeyPassphraseFile names a file holding the key's passphrase.
	KeyPassphraseFile string `yaml:"key_passphrase_file,omitempty"`
	// KnownHostsFile verifies dialed hosts against this file instead of the
	// default ~/.ssh/known_hosts. Like every connection setting, host-key
	// policy is data in this file — no environment variable configures it.
	KnownHostsFile string `yaml:"known_hosts_file,omitempty"`
	// InsecureHostKey skips host-key verification for this set's dials.
	// Closed-network, throwaway servers only; exactly one of this and
	// KnownHostsFile may be set.
	InsecureHostKey bool `yaml:"insecure_host_key,omitempty"`
	// Sudo reports whether the login may elevate. It is carried, not consumed:
	// the bring-up decides whether a step needs it, and netmap only passes it
	// along (netmap-design NM-e).
	Sudo bool `yaml:"sudo,omitempty"`
}

// Defaults are the file-level values every server inherits for the fields it
// omits.
type Defaults struct {
	// Slots is how many nodes one server may host.
	Slots int   `yaml:"slots,omitempty"`
	Ports Ports `yaml:"ports,omitempty"`
	SSH   SSH   `yaml:"ssh,omitempty"`
}

// Server is one place chainbench may run nodes. Index and Name are both
// selectors; a server needs at least one of them plus a host.
type Server struct {
	Index int    `yaml:"index,omitempty"`
	Name  string `yaml:"name,omitempty"`
	// Kind is local (default) or remote. It is the only field that decides
	// whether the ssh block matters.
	Kind Kind `yaml:"kind,omitempty"`
	// Host is the address nodes on this server are reachable at.
	Host  string `yaml:"host"`
	Slots int    `yaml:"slots,omitempty"`
	Ports Ports  `yaml:"ports,omitempty"`
	SSH   SSH    `yaml:"ssh,omitempty"`
}

// BandSpec is one port band: where it starts and how far apart consecutive
// slots sit.
type BandSpec struct {
	Base int `yaml:"base"`
	Step int `yaml:"step"`
}

// BandsSpec is the bands a slot draws from. The rpc band derives ws (http+1),
// auth (http+2) and metrics (http+3, when the step leaves room) unless the
// purpose declares its own band — a site whose firewall groups ports by
// purpose (auth in one range, http in another, ws in a third) writes each:
//
//	ports:
//	  p2p:     { base: 30301, step: 1 }
//	  rpc:     { base: 8601,  step: 1 }
//	  ws:      { base: 8701,  step: 1 }
//	  auth:    { base: 8501,  step: 1 }
//	  metrics: { base: 6060,  step: 0 }   # step 0: one shared scrape port
type BandsSpec struct {
	P2P     BandSpec  `yaml:"p2p,omitempty"`
	RPC     BandSpec  `yaml:"rpc,omitempty"`
	WS      *BandSpec `yaml:"ws,omitempty"`
	Auth    *BandSpec `yaml:"auth,omitempty"`
	Metrics *BandSpec `yaml:"metrics,omitempty"`
}

// HostSpec is one address the pool may place nodes on. It accepts either a
// bare address or a named entry, because most sites have nothing to say about
// a host beyond where it is, while a named one can be referenced as
// srv://<name>/path.
//
//	hosts: [10.0.0.11, 10.0.0.12]
//	hosts:
//	  - { name: bp1, addr: 10.0.0.11 }
type HostSpec struct {
	Name string `yaml:"name,omitempty"`
	Addr string `yaml:"addr"`
}

// UnmarshalYAML accepts the scalar shorthand as well as the mapping.
func (h *HostSpec) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var addr string
		if err := value.Decode(&addr); err != nil {
			return err
		}
		h.Addr, h.Name = addr, addr
		return nil
	}
	type plain HostSpec // avoid recursing into this method
	var out plain
	if err := value.Decode(&out); err != nil {
		return err
	}
	*h = HostSpec(out)
	if h.Name == "" {
		h.Name = h.Addr
	}
	return nil
}

// PoolSpec is the resource the network is allocated from: the addresses in the
// order they are consumed, how many port slots each may hold, and the bands the
// slots step through.
type PoolSpec struct {
	Hosts []HostSpec `yaml:"hosts"`
	Slots int        `yaml:"slots,omitempty"`
	Ports BandsSpec  `yaml:"ports,omitempty"`
}

// Set is the parsed server set.
type Set struct {
	// Version is the file format version, so a later change can reject an old
	// file by name instead of by a confusing field error.
	Version int `yaml:"version"`
	// PoolSpec is the declared resource (v2's single subject).
	PoolSpec PoolSpec `yaml:"pool"`
	// SSH reaches every non-loopback host in the pool.
	SSH SSH `yaml:"ssh,omitempty"`
	// DataRoot is no longer owned here — it moved to workspace-config so one DSL
	// runs across targets by swapping that file. The field is kept only to
	// reject a leftover value with a migration message rather than a confusing
	// unknown-field error; LoadSet errors when it is set.
	DataRoot string `yaml:"dataRoot,omitempty"`

	// Defaults and Servers are derived from the pool, not parsed: the surfaces
	// that select "a server" (--server, --all-servers, srv://) predate the pool and
	// still read them. When those surfaces move onto the pool, the derivation
	// goes with them.
	Defaults Defaults `yaml:"-"`
	Servers  []Server `yaml:"-"`
	// path is where this config was read from, for provenance in messages.
	path string
}

// SupportedVersion is the server-set format this build reads. v2 replaced the
// list of servers with a pool: the two allocation shapes an operator used to
// choose between (this machine with stepped ports, one node per host) are the
// same grid, so the file describes resources and the allocator decides
// placement (netmap-design 2.2a).

const SupportedVersion = 2

// Path is the file this config came from, for reporting where a port plan
// originated.
func (c *Set) Path() string { return c.path }

// LoadSet reads and validates the server set at path. It rejects unknown fields so
// a typo fails loudly rather than silently leaving a default in place.
func LoadSet(path string) (*Set, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if hint := LegacyNameHint(path); hint != "" {
				return nil, fmt.Errorf("serverset: config %s not found — %s", path, hint)
			}
			return nil, fmt.Errorf("serverset: config %s not found (copy %s and fill it in)", path, DefaultSampleFile)
		}
		return nil, fmt.Errorf("serverset: read %s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var c Set
	if err := dec.Decode(&c); err != nil {
		if hint := legacyHint(b); hint != "" {
			return nil, fmt.Errorf("serverset: %s looks like the pre-v%d format: %s", path, SupportedVersion, hint)
		}
		return nil, fmt.Errorf("serverset: parse %s: %w", path, err)
	}
	if c.DataRoot != "" {
		return nil, fmt.Errorf("serverset: %s sets dataRoot, which has moved to workspace-config — remove it here and put dataRoot in the --workspace-config file", path)
	}
	c.path = path
	if err := c.SSH.resolveSecretPaths(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("serverset: %s: %w", path, err)
	}
	c.expand()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// legacyConfigFile is the server set's pre-rename filename. It is recognized
// only to say "rename it" — silently reading it would leave two names alive.
const legacyConfigFile = "remote-server-config.yaml"

// LegacyNameHint reports whether the old-name file sits where the new-name file
// was looked for, so a missing server-set.yaml after an upgrade fails with the
// migration step instead of a generic not-found. Empty means no old file is in
// the way.
func LegacyNameHint(path string) string {
	if filepath.Base(path) != DefaultSetFile {
		return ""
	}
	old := filepath.Join(filepath.Dir(path), legacyConfigFile)
	if _, err := os.Stat(old); err != nil {
		return ""
	}
	return fmt.Sprintf("found %s (the old name); rename it: mv %s %s", old, old, path)
}

// resolveSecretPaths canonicalizes the file-reference fields so they mean the
// same thing wherever the command runs: a leading ~ expands to the home
// directory, and a relative path is anchored to the server-set file's own
