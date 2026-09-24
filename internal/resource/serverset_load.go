package resource

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// Loading a set from disk and refusing one that cannot be believed.
//
// Validation runs at load rather than at dial: a set that names a band its
// firewall does not open, or a login with no way to authenticate, fails when
// the file is read instead of after provisioning has already happened.

// resolveSecretPaths canonicalizes the file-reference fields so they mean the
// same thing wherever the command runs: a leading ~ expands to the home
// directory, and a relative path is anchored to the server-set file's own
// directory (dir), not the process working directory.
func (s *SSH) resolveSecretPaths(dir string) error {
	for _, f := range []*string{&s.PasswordFile, &s.KeyFile, &s.KeyPassphraseFile, &s.KnownHostsFile} {
		if *f == "" {
			continue
		}
		p := *f
		if p == "~" || strings.HasPrefix(p, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("expand %s: %w", p, err)
			}
			p = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		} else if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		*f = p
	}
	return nil
}

// loopback reports whether nodes on addr run on this  It is what
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
func (c *Set) expand() {
	slots := c.PoolSpec.Slots
	if slots < 1 {
		slots = 1
	}
	ports := Ports{
		P2PBase: c.PoolSpec.Ports.P2P.Base, P2PStep: c.PoolSpec.Ports.P2P.Step,
		RPCBase: c.PoolSpec.Ports.RPC.Base, RPCStep: c.PoolSpec.Ports.RPC.Step,
		WS: c.PoolSpec.Ports.WS, Auth: c.PoolSpec.Ports.Auth, Metrics: c.PoolSpec.Ports.Metrics,
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
	c.Defaults = Defaults{Slots: slots, Ports: ports, SSH: c.SSH}
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
			Slots: slots, Ports: ports, SSH: c.SSH,
		})
	}
}

// BuiltinPool is the pool used when no server set names one: this machine, the
// built-in bands, and room for a development-sized network.
func BuiltinPool(slots int) Pool {
	p := BuiltinPorts()
	if slots < 1 {
		slots = 1
	}
	return Pool{
		Hosts:  []Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots:  slots,
		Ports:  Bands{P2P: Band{Base: p.P2PBase, Step: p.P2PStep}, RPC: Band{Base: p.RPCBase, Step: p.RPCStep}},
		Origin: builtinOrigin,
	}
}

// legacyHint recognizes the flat pre-v1 server-set file (top-level ssh fields, no
// version) and says how to migrate, because the decoder's own "field not found"
// error does not.
func legacyHint(b []byte) string {
	text := string(b)
	if strings.Contains("\n"+text, "\nservers:") {
		return fmt.Sprintf("v%d replaced the server list with a pool: put every address under "+
			"`pool.hosts`, move `slots` and `ports` up to `pool`, and lift `ssh` to the "+
			"top level (dataRoot now lives in workspace-config, not here; see %s)", SupportedVersion, DefaultSampleFile)
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

// validate enforces a usable server set: a supported version, unique selectors,
// a host per server, and port steps that cannot produce colliding ports.

func (c *Set) validate() error {
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
	if err := c.SSH.validateFields(); err != nil {
		return fmt.Errorf("serverset: %s: %w", c.path, err)
	}
	return nil
}

// validateFields enforces the ssh block's cross-field rules at load time — a
// bad combination must fail when the file is read (Load's fail-loud policy),
// not when the server is first dialed mid-run.
func (s SSH) validateFields() error {
	if s.Password != "" && s.PasswordFile != "" {
		return fmt.Errorf("ssh sets both password and password_file — keep exactly one")
	}
	if s.InsecureHostKey && s.KnownHostsFile != "" {
		return fmt.Errorf("ssh sets both insecure_host_key and known_hosts_file — keep exactly one")
	}
	if s.KeyPassphraseFile != "" && s.KeyFile == "" {
		return fmt.Errorf("ssh sets key_passphrase_file without key_file — the passphrase has no key to unlock")
	}
	return nil
}

// validate rejects a port plan that would assign the same port twice. The floors
// are what make the plan collision-free by construction.
func (p Ports) validate(where string) error {
	// The wemix family's derived etcd ports need more p2p room, but that is a
	// family fact checked where the family is known (portplan, at allocate);
	// the file itself only requires a usable stride.
	if p.P2PStep < 1 {
		return fmt.Errorf("serverset: server %s p2pStep is %d, want >= 1", where, p.P2PStep)
	}
	derived := p.WS == nil || p.Auth == nil
	if derived && p.RPCStep < minRPCStep {
		return fmt.Errorf("serverset: server %s rpcStep is %d, want >= %d (http, ws, auth — or declare ws/auth bands of their own)", where, p.RPCStep, minRPCStep)
	}
	if !derived && p.RPCStep < 1 {
		return fmt.Errorf("serverset: server %s rpcStep is %d, want >= 1", where, p.RPCStep)
	}
	if p.P2PBase <= 0 || p.RPCBase <= 0 {
		return fmt.Errorf("serverset: server %s needs a positive p2pBase and rpcBase", where)
	}
	for name, b := range map[string]*BandSpec{"ws": p.WS, "auth": p.Auth} {
		if b != nil && (b.Base <= 0 || b.Step < 1) {
			return fmt.Errorf("serverset: server %s %s band needs a positive base and step >= 1", where, name)
		}
	}
	if p.Metrics != nil && (p.Metrics.Base <= 0 || p.Metrics.Step < 0) {
		return fmt.Errorf("serverset: server %s metrics band needs a positive base and step >= 0", where)
	}
	return nil
}

// HasMetrics reports whether this plan leaves room for a metrics endpoint.
