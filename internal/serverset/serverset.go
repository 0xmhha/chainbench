// Package serverset loads the server inventory — where chainbench may run
// nodes, on what ports, under what data root, and how to reach it — from a YAML
// config the repository never carries.
//
// Host addresses and port assignments are site-specific and sensitive, so they
// belong in a gitignored file rather than in code or in a test definition. The
// same entry shape describes this machine and a remote host: only Kind and the
// ssh block differ, so a caller composing a network reads one structure either
// way and the local/remote difference stays a property of the data.
//
// It is a control-plane concern and never dials. It resolves a chosen server to
// remote.Credentials, layering server values over file defaults, and lets the
// environment override secrets (CHAINBENCH_REMOTE_*) so passwords need not live
// in the file at all.
package serverset

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/core/target"
)

// DefaultConfigFile is the inventory path used when --server-config is omitted.
// It is gitignored; only DefaultSampleFile is tracked.
const DefaultConfigFile = "remote-server-config.yaml"

// DefaultSampleFile is the tracked template an operator copies.
const DefaultSampleFile = "remote-server-config.sample.yaml"

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
	// KindLocal runs nodes on this machine. It is the zero value, so an entry
	// that says nothing about access is local.
	KindLocal Kind = "local"
	// KindRemote runs nodes on another host over SSH.
	KindRemote Kind = "remote"
)

// Ports is a server's port plan: the two disjoint bands node ports are assigned
// from. Zero fields inherit the file defaults, and any still zero fall back to
// the built-in defaults (Defaults.Ports) so an inventory can name only what it
// needs to move.
type Ports struct {
	// P2PBase is the first node's devp2p port; P2PStep is the per-node stride.
	P2PBase int `yaml:"p2pBase,omitempty"`
	P2PStep int `yaml:"p2pStep,omitempty"`
	// RPCBase is the first node's HTTP port; RPCStep is the per-node stride.
	// ws = http+1, auth = http+2, and metrics = http+3 when the step allows.
	RPCBase int `yaml:"rpcBase,omitempty"`
	RPCStep int `yaml:"rpcStep,omitempty"`
}

// SSH is how a remote server is reached. Secrets are better supplied through
// CHAINBENCH_REMOTE_PASS / _KEY_FILE than written here, and the loader never
// echoes these values.
type SSH struct {
	User     string `yaml:"user,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Password string `yaml:"password,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
	// Sudo reports whether the login may elevate. It is carried, not consumed:
	// the bring-up decides whether a step needs it, and netmap only passes it
	// along (netmap-design NM-e).
	Sudo bool `yaml:"sudo,omitempty"`
}

// Defaults are the file-level values every server inherits for the fields it
// omits.
type Defaults struct {
	// Slots is how many nodes one server may host.
	Slots int `yaml:"slots,omitempty"`
	// DataRoot is where a node's data plane lives on the server.
	DataRoot string `yaml:"dataRoot,omitempty"`
	Ports    Ports  `yaml:"ports,omitempty"`
	SSH      SSH    `yaml:"ssh,omitempty"`
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
	Host     string `yaml:"host"`
	Slots    int    `yaml:"slots,omitempty"`
	DataRoot string `yaml:"dataRoot,omitempty"`
	Ports    Ports  `yaml:"ports,omitempty"`
	SSH      SSH    `yaml:"ssh,omitempty"`
}

// BandSpec is one port band: where it starts and how far apart consecutive
// slots sit.
type BandSpec struct {
	Base int `yaml:"base"`
	Step int `yaml:"step"`
}

// BandsSpec is the two disjoint bands a slot draws from.
type BandsSpec struct {
	P2P BandSpec `yaml:"p2p,omitempty"`
	RPC BandSpec `yaml:"rpc,omitempty"`
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

// Config is the parsed inventory.
type Config struct {
	// Version is the file format version, so a later change can reject an old
	// file by name instead of by a confusing field error.
	Version int `yaml:"version"`
	// PoolSpec is the declared resource (v2's single subject).
	PoolSpec PoolSpec `yaml:"pool"`
	// SSH reaches every non-loopback host in the pool.
	SSH SSH `yaml:"ssh,omitempty"`
	// DataRoot is where a node's data plane lives on the target.
	DataRoot string `yaml:"dataRoot,omitempty"`

	// Defaults and Servers are derived from the pool, not parsed: the surfaces
	// that select "a server" (--server, --fleet, srv://) predate the pool and
	// still read them. When those surfaces move onto the pool, the derivation
	// goes with them.
	Defaults Defaults `yaml:"-"`
	Servers  []Server `yaml:"-"`
	// path is where this config was read from, for provenance in messages.
	path string
}

// SupportedVersion is the inventory format this build reads. v2 replaced the
// list of servers with a pool: the two allocation shapes an operator used to
// choose between (this machine with stepped ports, one node per host) are the
// same grid, so the file describes resources and the allocator decides
// placement (netmap-design 2.2a).
const SupportedVersion = 2

// Path is the file this config came from, for reporting where a port plan
// originated.
func (c *Config) Path() string { return c.path }

// Load reads and validates the inventory at path. It rejects unknown fields so
// a typo fails loudly rather than silently leaving a default in place.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("serverset: config %s not found (copy %s and fill it in)", path, DefaultSampleFile)
		}
		return nil, fmt.Errorf("serverset: read %s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var c Config
	if err := dec.Decode(&c); err != nil {
		if hint := legacyHint(b); hint != "" {
			return nil, fmt.Errorf("serverset: %s looks like the pre-v%d format: %s", path, SupportedVersion, hint)
		}
		return nil, fmt.Errorf("serverset: parse %s: %w", path, err)
	}
	c.path = path
	c.expand()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// loopback reports whether nodes on addr run on this machine. It is what
// decides SSH, and it is derived rather than declared: a "kind" field can
// disagree with the address it sits next to, and then the file says two things.
func loopback(addr string) bool {
	if addr == "localhost" {
		return true
	}
	ip := net.ParseIP(addr)
	return ip != nil && ip.IsLoopback()
}

// expand derives the server view from the pool. Every host becomes one entry
// carrying the pool's slots, bands, data root and access, so the selection
// surfaces keep working while the pool becomes the thing that is actually
// declared.
func (c *Config) expand() {
	slots := c.PoolSpec.Slots
	if slots < 1 {
		slots = 1
	}
	ports := Ports{
		P2PBase: c.PoolSpec.Ports.P2P.Base, P2PStep: c.PoolSpec.Ports.P2P.Step,
		RPCBase: c.PoolSpec.Ports.RPC.Base, RPCStep: c.PoolSpec.Ports.RPC.Step,
	}
	builtin := BuiltinPorts()
	if ports.P2PBase == 0 {
		ports.P2PBase = builtin.P2PBase
	}
	if ports.P2PStep == 0 {
		ports.P2PStep = builtin.P2PStep
	}
	if ports.RPCBase == 0 {
		ports.RPCBase = builtin.RPCBase
	}
	if ports.RPCStep == 0 {
		ports.RPCStep = builtin.RPCStep
	}
	c.Defaults = Defaults{Slots: slots, DataRoot: c.DataRoot, Ports: ports, SSH: c.SSH}
	c.Servers = make([]Server, 0, len(c.PoolSpec.Hosts))
	for i, h := range c.PoolSpec.Hosts {
		kind := KindRemote
		if loopback(h.Addr) {
			kind = KindLocal
		}
		name := h.Name
		if name == "" {
			name = h.Addr
		}
		c.Servers = append(c.Servers, Server{
			Index: i + 1, Name: name, Kind: kind, Host: h.Addr,
			Slots: slots, DataRoot: c.DataRoot, Ports: ports, SSH: c.SSH,
		})
	}
}

// Pool is the inventory as netmap allocates from it.
func (c *Config) Pool() netmap.Pool {
	hosts := make([]netmap.Host, 0, len(c.Servers))
	for _, s := range c.Servers {
		hosts = append(hosts, netmap.Host{Name: s.Name, Addr: s.Host})
	}
	return netmap.Pool{
		Hosts: hosts,
		Slots: c.Defaults.Slots,
		Ports: netmap.Bands{
			P2PBase: c.Defaults.Ports.P2PBase, P2PStep: c.Defaults.Ports.P2PStep,
			RPCBase: c.Defaults.Ports.RPCBase, RPCStep: c.Defaults.Ports.RPCStep,
		},
		Source: c.path,
	}
}

// BuiltinPool is the pool used when no inventory names one: this machine, the
// built-in bands, and room for a development-sized network.
func BuiltinPool(slots int) netmap.Pool {
	p := BuiltinPorts()
	if slots < 1 {
		slots = 1
	}
	return netmap.Pool{
		Hosts:  []netmap.Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots:  slots,
		Ports:  netmap.Bands{P2PBase: p.P2PBase, P2PStep: p.P2PStep, RPCBase: p.RPCBase, RPCStep: p.RPCStep},
		Source: builtinSource,
	}
}

// legacyHint recognizes the flat pre-v1 inventory (top-level ssh fields, no
// version) and says how to migrate, because the decoder's own "field not found"
// error does not.
func legacyHint(b []byte) string {
	text := string(b)
	if strings.Contains("\n"+text, "\nservers:") {
		return fmt.Sprintf("v%d replaced the server list with a pool: put every address under "+
			"`pool.hosts`, move `slots` and `ports` up to `pool`, and lift `ssh`/`dataRoot` to the "+
			"top level (see %s)", SupportedVersion, DefaultSampleFile)
	}
	hasTopLevelSSH := false
	for _, k := range []string{"\nuser:", "\nport:", "\npassword:", "\nkey_file:", "\nsshPort:", "\nhosts:"} {
		if strings.Contains("\n"+text, k) {
			hasTopLevelSSH = true
			break
		}
	}
	if !hasTopLevelSSH {
		return ""
	}
	return fmt.Sprintf("move the top-level user/port/password/key_file under `defaults.ssh:`, "+
		"add `version: %d`, and give each server a `kind: local|remote` (see %s)", SupportedVersion, DefaultSampleFile)
}

// validate enforces a usable inventory: a supported version, unique selectors,
// a host per server, and port steps that cannot produce colliding ports.
func (c *Config) validate() error {
	if c.Version != SupportedVersion {
		return fmt.Errorf("serverset: %s has version %d, want %d (see %s)", c.path, c.Version, SupportedVersion, DefaultSampleFile)
	}
	if len(c.PoolSpec.Hosts) == 0 {
		return fmt.Errorf("serverset: %s declares no pool hosts", c.path)
	}
	for i, h := range c.PoolSpec.Hosts {
		if h.Addr == "" {
			return fmt.Errorf("serverset: pool host %d (%q) has no addr", i+1, h.Name)
		}
	}
	if c.PoolSpec.Slots < 0 {
		return fmt.Errorf("serverset: %s declares %d slots per host, want >= 1", c.path, c.PoolSpec.Slots)
	}
	seenIndex := map[int]bool{}
	seenName := map[string]bool{}
	for i, s := range c.Servers {
		where := s.label(i)
		if s.Host == "" {
			return fmt.Errorf("serverset: server %s has no host", where)
		}
		if s.Index == 0 && s.Name == "" {
			return fmt.Errorf("serverset: server %s needs an index or a name to select it by", where)
		}
		switch s.Kind {
		case "", KindLocal, KindRemote:
		default:
			return fmt.Errorf("serverset: server %s has kind %q (want %s or %s)", where, s.Kind, KindLocal, KindRemote)
		}
		if s.Index != 0 {
			if seenIndex[s.Index] {
				return fmt.Errorf("serverset: duplicate server index %d", s.Index)
			}
			seenIndex[s.Index] = true
		}
		if s.Name != "" {
			if seenName[s.Name] {
				return fmt.Errorf("serverset: duplicate server name %q", s.Name)
			}
			seenName[s.Name] = true
		}
		if err := c.resolve(s).Ports.validate(where); err != nil {
			return err
		}
	}
	return nil
}

// validate rejects a port plan that would assign the same port twice. The floors
// are what make the plan collision-free by construction.
func (p Ports) validate(where string) error {
	if p.P2PStep < minP2PStep {
		return fmt.Errorf("serverset: server %s p2pStep is %d, want >= %d (etcd is derived as p2p+1)", where, p.P2PStep, minP2PStep)
	}
	if p.RPCStep < minRPCStep {
		return fmt.Errorf("serverset: server %s rpcStep is %d, want >= %d (http, ws, auth)", where, p.RPCStep, minRPCStep)
	}
	if p.P2PBase <= 0 || p.RPCBase <= 0 {
		return fmt.Errorf("serverset: server %s needs a positive p2pBase and rpcBase", where)
	}
	return nil
}

// HasMetrics reports whether this plan leaves room for a metrics endpoint.
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
func (c *Config) Server(index int) (Server, error) {
	for _, s := range c.Servers {
		if s.Index == index && index != 0 {
			return c.resolve(s), nil
		}
	}
	return Server{}, fmt.Errorf("serverset: no server with index %d in %s (available: %v)", index, c.path, c.indexes())
}

// ByName returns the server with the given name, fully resolved.
func (c *Config) ByName(name string) (Server, error) {
	for _, s := range c.Servers {
		if s.Name == name && name != "" {
			return c.resolve(s), nil
		}
	}
	return Server{}, fmt.Errorf("serverset: no server named %q in %s (available: %v)", name, c.path, c.names())
}

// Select resolves a server by name when one is given, otherwise by index, and
// otherwise the only server in a single-server inventory. It is what a command
// with an optional --server flag calls.
func (c *Config) Select(name string, index int) (Server, error) {
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
func (c *Config) resolve(s Server) Server {
	if s.Kind == "" {
		s.Kind = KindLocal
	}
	if s.Slots == 0 {
		s.Slots = c.Defaults.Slots
	}
	if s.Slots == 0 {
		s.Slots = 1
	}
	if s.DataRoot == "" {
		s.DataRoot = c.Defaults.DataRoot
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
func (c *Config) indexes() []int {
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
func (c *Config) names() []string {
	out := make([]string, 0, len(c.Servers))
	for _, s := range c.Servers {
		if s.Name != "" {
			out = append(out, s.Name)
		}
	}
	sort.Strings(out)
	return out
}

// Credentials builds the SSH credentials for a remote server. The environment
// overrides file values (CHAINBENCH_REMOTE_USER / _PASS / _KEY_FILE /
// _KEY_PASSPHRASE) so secrets can stay out of the file. It errors if the server
// is local (there is nothing to authenticate to) or if no user or auth resolves.
// env is injected for testing.
func (s Server) Credentials(env func(string) string) (remote.Credentials, error) {
	if !s.IsRemote() {
		return remote.Credentials{}, fmt.Errorf("serverset: server %s is local — it has no SSH credentials", s.label(0))
	}
	if env == nil {
		env = func(string) string { return "" }
	}
	user, pass, keyFile := s.SSH.User, s.SSH.Password, s.SSH.KeyFile
	if v := env("CHAINBENCH_REMOTE_USER"); v != "" {
		user = v
	}
	if v := env("CHAINBENCH_REMOTE_PASS"); v != "" {
		pass = v
	}
	if v := env("CHAINBENCH_REMOTE_KEY_FILE"); v != "" {
		keyFile = v
	}
	if user == "" {
		return remote.Credentials{}, fmt.Errorf(
			"serverset: server %s has no SSH user (set defaults.ssh.user or CHAINBENCH_REMOTE_USER)", s.label(0))
	}
	port := s.SSH.Port
	if port == 0 {
		port = defaultSSHPort
	}
	rc := remote.Credentials{User: user, Host: s.Host, Port: port, Password: pass}
	if keyFile != "" {
		key, err := remote.LoadPrivateKey(keyFile)
		if err != nil {
			return remote.Credentials{}, fmt.Errorf("serverset: server %s: %w", s.label(0), err)
		}
		rc.PrivateKey = key
		rc.Passphrase = env("CHAINBENCH_REMOTE_KEY_PASSPHRASE")
	}
	if rc.Password == "" && len(rc.PrivateKey) == 0 {
		return remote.Credentials{}, fmt.Errorf(
			"serverset: server %s has no SSH auth (set defaults.ssh.password/key_file or CHAINBENCH_REMOTE_PASS/CHAINBENCH_REMOTE_KEY_FILE)", s.label(0))
	}
	return rc, nil
}

// InventoryLookup returns a target.Inventory backed by the inventory file at
// path (empty uses DefaultConfigFile). It is how an srv://<name>/path target
// gets its host, port and credentials without any of those appearing in a
// command line, a spec file, or a persisted workspace.
//
// The file is opened on each lookup rather than cached: a lookup happens once
// per target, and an operator editing the inventory mid-session should not have
// to reason about which copy is in effect.
func InventoryLookup(path string) target.Inventory {
	return func(name string, env func(string) string) (remote.Credentials, error) {
		if path == "" {
			path = DefaultConfigFile
		}
		cfg, err := Load(path)
		if err != nil {
			return remote.Credentials{}, fmt.Errorf("serverset: %q needs the inventory: %w", name, err)
		}
		s, err := cfg.ByName(name)
		if err != nil {
			return remote.Credentials{}, err
		}
		return s.Credentials(env)
	}
}
