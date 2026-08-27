package resource_test

import (
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "server-set.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

// sample is a mixed pool: this machine and two remote hosts. Whether a host is
// reached over SSH is derived from its address, so the file never says "local"
// beside a remote IP.
const sample = `
version: 2
pool:
  hosts:
    - { name: local, addr: 127.0.0.1 }
    - { name: bp1, addr: 10.0.0.1 }
    - 10.0.0.7
  slots: 8
  ports:
    p2p: { base: 30303, step: 10 }
    rpc: { base: 8545, step: 10 }
ssh:
  user: ubuntu
  port: 22
  password: filepass
  sudo: true
dataRoot: /var/lib/chainbench
`

// remoteSample is sample with every host routable, so it has a whole-set pool.
// A set that mixes this machine with remote hosts deliberately does not.
const remoteSample = `
version: 2
pool:
  hosts:
    - { name: bp0, addr: 10.0.0.9 }
    - { name: bp1, addr: 10.0.0.1 }
    - 10.0.0.7
  slots: 8
  ports:
    p2p: { base: 30303, step: 10 }
    rpc: { base: 8545, step: 10 }
ssh:
  user: ubuntu
dataRoot: /var/lib/chainbench
`

func load(t *testing.T, body string) *resource.Set {
	t.Helper()
	cfg, err := resource.LoadSet(write(t, body))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// TestPool_ReadsTheGrid: the file's subject is the resource, and the pool is
// what the allocator is handed.
func TestPool_ReadsTheGrid(t *testing.T) {
	p, err := load(t, remoteSample).Pool(1, 100)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	if len(p.Hosts) != 3 || p.Slots != 8 {
		t.Fatalf("pool = %d hosts x %d slots", len(p.Hosts), p.Slots)
	}
	if p.Cap() != 24 {
		t.Errorf("Cap() = %d, want 24", p.Cap())
	}
	if p.Ports.P2PBase != 30303 || p.Ports.RPCBase != 8545 {
		t.Errorf("bands = %+v", p.Ports)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("pool from a valid file must validate: %v", err)
	}
	// A bare address names itself, so it can still be selected and referenced.
	if p.Hosts[2].Name != "10.0.0.7" {
		t.Errorf("unnamed host = %q, want it named by its address", p.Hosts[2].Name)
	}
	// Where a port came from is never a guess.
	if !strings.Contains(p.Source, "server-set.yaml") {
		t.Errorf("source = %q, want it to name the file", p.Source)
	}
}

// TestPool_UnsetBandsFallBackToTheBuiltins keeps a minimal inventory usable:
// an operator listing addresses should not have to restate the ports.
func TestPool_UnsetBandsFallBackToTheBuiltins(t *testing.T) {
	p, err := load(t, "version: 2\npool:\n  hosts: [127.0.0.1]\n").Pool(1, 100)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	b := resource.BuiltinPorts()
	if p.Ports.P2PBase != b.P2PBase || p.Ports.RPCStep != b.RPCStep {
		t.Errorf("bands = %+v, want the built-ins %+v", p.Ports, b)
	}
	if p.Slots != 1 {
		t.Errorf("slots = %d, want 1 by default", p.Slots)
	}
}

// TestServers_DerivedFromTheHostAddress: SSH is decided by the address, not by
// a field that can disagree with the address it sits next to.
func TestServers_DerivedFromTheHostAddress(t *testing.T) {
	cfg := load(t, sample)

	local, err := cfg.ByName("local")
	if err != nil {
		t.Fatalf("ByName(local): %v", err)
	}
	if local.IsRemote() {
		t.Error("a loopback address must not be reached over SSH")
	}
	remote, err := cfg.ByName("bp1")
	if err != nil {
		t.Fatalf("ByName(bp1): %v", err)
	}
	if !remote.IsRemote() {
		t.Error("a routable address must be reached over SSH")
	}
	// The pool's slots, bands, data root and access reach every host: the pool
	// is one resource, not a list of individually-configured machines.
	for _, s := range []resource.Server{local, remote} {
		if s.Slots != 8 || s.DataRoot != "/var/lib/chainbench" || s.Ports.RPCBase != 8545 {
			t.Errorf("%s did not inherit the pool: %+v", s.Name, s)
		}
	}
	if !remote.SSH.Sudo {
		t.Error("sudo is carried through to the host that may need it")
	}
}

func TestSelect_ByNameByIndexAndTheLoneHost(t *testing.T) {
	cfg := load(t, sample)

	if s, err := cfg.Select("10.0.0.7", 0); err != nil || s.Host != "10.0.0.7" {
		t.Errorf("Select by name = %+v, %v", s, err)
	}
	if s, err := cfg.Select("", 1); err != nil || s.Name != "local" {
		t.Errorf("Select by index = %+v, %v", s, err)
	}
	// With several hosts and no selector, the caller has to say which.
	if _, err := cfg.Select("", 0); err == nil {
		t.Error("want an error selecting nothing from a multi-host pool")
	}
	// With exactly one, there is nothing to disambiguate.
	one := load(t, "version: 2\npool:\n  hosts: [{name: only, addr: 127.0.0.1}]\n")
	if s, err := one.Select("", 0); err != nil || s.Name != "only" {
		t.Errorf("Select on a lone host = %+v, %v", s, err)
	}
}

// TestPlacement_LocalAndRemoteReadTheSameFields: local and remote differ in the
// address and in whether the data plane is reached over SSH — nothing else. The
// pool is read from the same fields either way, which is why composing a
// network does not branch on it.
func TestPoolFor_LocalAndRemoteReadTheSameFields(t *testing.T) {
	cfg := load(t, sample)

	local, _ := cfg.ByName("local")
	lp := cfg.PoolFor(local, 1, 100)
	if lp.Slots != 8 {
		t.Errorf("local pool = %+v", lp)
	}

	remoteSrv, _ := cfg.ByName("bp1")
	rp := cfg.PoolFor(remoteSrv, 1, 100)
	if lp.Ports != rp.Ports {
		t.Errorf("port bands diverged: local=%+v remote=%+v", lp.Ports, rp.Ports)
	}
	if len(rp.Hosts) != 1 || rp.Hosts[0].Addr != "10.0.0.1" {
		t.Errorf("remote pool hosts = %v", rp.Hosts)
	}
	for _, p := range []netmap.Pool{lp, rp} {
		if !strings.Contains(p.Source, "server-set.yaml") {
			t.Errorf("source does not name the file: %q", p.Source)
		}
	}
}

// TestResolveServer_TargetCarriesTheLocality pins where local/remote survives
// now that the pool no longer stores it: in the machine spec, derived from the
// address rather than recorded beside it.
func TestResolveServer_TargetCarriesTheLocality(t *testing.T) {
	set := write(t, sample)

	out, err := resource.ResolveServer(resource.ServerRef{SetPath: set, Name: "local"}, 1, 100)
	if err != nil {
		t.Fatalf("ResolveServer(local): %v", err)
	}
	if out.Target.IsRemote() {
		t.Errorf("a loopback address must not be remote: %+v", out.Target)
	}
	if out.Target.DataRoot != "/var/lib/chainbench" {
		t.Errorf("target data root = %q", out.Target.DataRoot)
	}

	out, err = resource.ResolveServer(resource.ServerRef{SetPath: set, Name: "bp1"}, 1, 100)
	if err != nil {
		t.Fatalf("ResolveServer(bp1): %v", err)
	}
	if !out.Target.IsRemote() {
		t.Errorf("a routable address must be remote: %+v", out.Target)
	}
}

func TestBuiltin_NamesItselfAsTheSource(t *testing.T) {
	// Where a port came from must never be a guess, including when no file
	// was involved.
	p := resource.Builtin(1, 100)
	if !strings.Contains(p.Source, "built-in") {
		t.Errorf("source = %q, want it to say built-in", p.Source)
	}
	if p.Ports.RPCBase != resource.BuiltinPorts().RPCBase {
		t.Errorf("builtin ports not used: %+v", p.Ports)
	}

	pool := resource.BuiltinPool(4)
	if len(pool.Hosts) != 1 || pool.Slots != 4 || !strings.Contains(pool.Source, "built-in") {
		t.Errorf("builtin pool = %+v", pool)
	}
}

func TestPool_SpreadsOneNodePerHost(t *testing.T) {
	cfg := load(t, `
version: 2
pool:
  hosts: [10.0.0.1, 10.0.0.2, 10.0.0.3]
  slots: 1
ssh: {user: ubuntu}
`)
	p, err := cfg.Pool(1, 100)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	if len(p.Hosts) != 3 || p.Slots != 1 || p.Cap() != 3 {
		t.Errorf("set pool = %d hosts x %d slots", len(p.Hosts), p.Slots)
	}
}

func TestPool_RejectsAMixedSet(t *testing.T) {
	// Half on this machine and half over SSH is two port regimes at once, and
	// the allocator has no way to express that.
	cfg := load(t, sample)
	if _, err := cfg.Pool(1, 100); err == nil {
		t.Fatal("want an error for a mixed local/remote set")
	}
}

// TestCredentials_TheFileIsTheSingleSource pins the rule that makes a named
// server's login predictable: the set file decides, and an exported
// CHAINBENCH_REMOTE_* left over from another environment changes nothing.
func TestCredentials_TheFileIsTheSingleSource(t *testing.T) {
	t.Setenv(remote.EnvUser, "deploy")
	t.Setenv(remote.EnvPass, "envpass")
	cfg := load(t, sample)
	s, _ := cfg.ByName("bp1")

	c, err := s.Credentials()
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if c.User == "deploy" || c.Password == "envpass" {
		t.Errorf("the environment redirected a login the file defines: %+v", c)
	}
	if c.Host != "10.0.0.1" || c.Port != 22 {
		t.Errorf("creds = %+v", c)
	}
}

// TestCredentials_PasswordFile pins the way secrets stay out of the set file
// itself: password_file references a one-line 0600 file, exactly one of the
// two password fields may be set, and an empty file is refused.
func TestCredentials_PasswordFile(t *testing.T) {
	dir := t.TempDir()
	pf := filepath.Join(dir, "pass")
	if err := os.WriteFile(pf, []byte("frompf\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ring := "version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: ubuntu, password_file: " + pf + "}\n"
	cfg := load(t, ring)
	s, _ := cfg.ByName("10.0.0.9")
	c, err := s.Credentials()
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if c.Password != "frompf" {
		t.Errorf("password_file not read (trailing newline must be trimmed): %q", c.Password)
	}

	// Both password fields at once is a file mistake, so it must fail when the
	// file is read — not later, when the server is first dialed.
	if _, err := resource.LoadSet(write(t, "version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: ubuntu, password: x, password_file: "+pf+"}\n")); err == nil {
		t.Fatal("Load accepted both ssh.password and ssh.password_file")
	}

	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	ec := load(t, "version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: ubuntu, password_file: "+empty+"}\n")
	se, _ := ec.ByName("10.0.0.9")
	if _, err := se.Credentials(); err == nil {
		t.Fatal("accepted an empty secret file")
	}
}

func TestCredentials_LocalServerHasNone(t *testing.T) {
	// Asking a local server for SSH credentials is a caller mistake worth
	// naming, not something to answer with an empty struct.
	cfg := load(t, sample)
	s, _ := cfg.ByName("local")
	if _, err := s.Credentials(); err == nil {
		t.Fatal("want an error asking a local server for credentials")
	}
}

func TestCredentials_NoAuthErrors(t *testing.T) {
	cfg := load(t, "version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: ubuntu}\n")
	s, _ := cfg.ByName("10.0.0.9")
	if _, err := s.Credentials(); err == nil {
		t.Fatal("want a no-auth error")
	}
}

func TestLoad_RejectsBadInventories(t *testing.T) {
	for _, c := range []struct{ name, body, want string }{
		{"unknown field", "version: 2\npool:\n  hosts: [h]\n  slotz: 2\n", "slotz"},
		{"no hosts", "version: 2\npool:\n  hosts: []\n", "no pool hosts"},
		{"host without address", "version: 2\npool:\n  hosts: [{name: bp1}]\n", "no addr"},
		{"wrong version", "version: 99\npool:\n  hosts: [h]\n", "version 99"},
		// p2p step 1 is legal at load time now — the wemix family's larger
		// reservation is checked where the family is known (portplan). A zero
		// step means "unset" and inherits; what the file itself still refuses
		// is a stride that cannot advance.
		{"p2p step negative", "version: 2\npool:\n  hosts: [h]\n  ports: {p2p: {base: 30303, step: -1}, rpc: {base: 8545, step: 10}}\n", "p2pStep"},
		{"rpc step too small", "version: 2\npool:\n  hosts: [h]\n  ports: {p2p: {base: 30303, step: 10}, rpc: {base: 8545, step: 2}}\n", "rpcStep"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := resource.LoadSet(write(t, c.body))
			if err == nil {
				t.Fatalf("want an error mentioning %q", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %v, want it to mention %q", err, c.want)
			}
		})
	}
}

func TestLoad_MissingFilePointsAtTheSample(t *testing.T) {
	_, err := resource.LoadSet(filepath.Join(t.TempDir(), "absent.yaml"))
	if err == nil {
		t.Fatal("want an error for a missing file")
	}
	if !strings.Contains(err.Error(), resource.DefaultSampleFile) {
		t.Errorf("error = %v, want it to point at the sample", err)
	}
}

// TestLoad_V1FormatSaysHowToMigrate: v2 dropped the server list, and a file
// written for v1 must be told what changed rather than failing on a field name.
func TestLoad_V1FormatSaysHowToMigrate(t *testing.T) {
	_, err := resource.LoadSet(write(t, "version: 1\nservers:\n  - name: bp1\n    host: 10.0.0.1\n"))
	if err == nil {
		t.Fatal("want an error for the v1 format")
	}
	for _, want := range []string{"pool.hosts", "slots", "ssh", DefaultSample} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("migration hint missing %q: %v", want, err)
		}
	}
}

// DefaultSample is spelled out so the hint keeps pointing at a file that exists.
const DefaultSample = "server-set.sample.yaml"

func TestPorts_MetricsNeedsRoomInTheStep(t *testing.T) {
	if !(resource.Ports{RPCStep: 4}).HasMetrics() {
		t.Error("a step of 4 leaves room for metrics")
	}
	if (resource.Ports{RPCStep: 3}).HasMetrics() {
		t.Error("a step of 3 has no room for metrics")
	}
}

func TestSampleFileParses(t *testing.T) {
	// The tracked template must stay loadable: it is the first thing an
	// operator copies.
	cfg, err := resource.LoadSet(filepath.Join("..", "..", resource.DefaultSampleFile))
	if err != nil {
		t.Fatalf("the tracked sample does not load: %v", err)
	}
	// The sample is deliberately mixed (this machine plus remote hosts), so it
	// has no whole-set pool; every server's own pool must still validate.
	for i := range cfg.Servers {
		srv, err := cfg.Server(i + 1)
		if err != nil {
			t.Fatalf("server %d: %v", i+1, err)
		}
		if err := cfg.PoolFor(srv, 1, 100).Validate(); err != nil {
			t.Fatalf("the sample's %s pool does not validate: %v", srv.Name, err)
		}
	}
}

// TestLoad_KeyPassphraseFileNeedsKeyFile: a passphrase with no key to unlock
// is a file mistake, and file mistakes fail at Load.
func TestLoad_KeyPassphraseFileNeedsKeyFile(t *testing.T) {
	_, err := resource.LoadSet(write(t, "version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: u, password: p, key_passphrase_file: /tmp/pp}\n"))
	if err == nil || !strings.Contains(err.Error(), "key_passphrase_file") {
		t.Fatalf("err = %v, want the key_passphrase_file-without-key_file rule", err)
	}
}

// TestCredentials_InsecureSecretFileRefused: the plaintext password gets the
// same 0600 rule the key file already has.
func TestCredentials_InsecureSecretFileRefused(t *testing.T) {
	pf := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(pf, []byte("p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := load(t, "version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: u, password_file: "+pf+"}\n")
	s, _ := cfg.ByName("10.0.0.9")
	if _, err := s.Credentials(); err == nil || !strings.Contains(err.Error(), "permissions") {
		t.Fatalf("err = %v, want an insecure-permissions refusal", err)
	}
}

// TestCredentials_RelativeSecretPathAnchorsToTheFile: `password_file: pass`
// means the file next to the server set, wherever the command runs from.
func TestCredentials_RelativeSecretPathAnchorsToTheFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pass"), []byte("sibling\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "server-set.yaml")
	if err := os.WriteFile(p, []byte("version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: u, password_file: pass}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir()) // somewhere the sibling file is NOT
	cfg, err := resource.LoadSet(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	s, _ := cfg.ByName("10.0.0.9")
	c, err := s.Credentials()
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if c.Password != "sibling" {
		t.Errorf("password = %q, want the set file's sibling", c.Password)
	}
}

// TestLoad_OldNameGetsAMigrationHint: after the rename, the old file sitting at
// the default location must produce the migration step, not a generic
// not-found — and never be read silently.
func TestLoad_OldNameGetsAMigrationHint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "remote-server-config.yaml"), []byte("version: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := resource.LoadSet(filepath.Join(dir, "server-set.yaml"))
	if err == nil || !strings.Contains(err.Error(), "rename") {
		t.Fatalf("err = %v, want a rename hint for the old file name", err)
	}
}

// TestLoad_PerPurposeBands pins the firewall-shaped scheme: a site that groups
// ports by purpose (auth 85xx, http 86xx, ws 87xx, p2p 303xx, one metrics
// port) declares each band, and tight steps become legal because nothing
// derives from the rpc band any more.
func TestLoad_PerPurposeBands(t *testing.T) {
	cfg := load(t, "version: 2\n"+
		"pool:\n"+
		"  hosts: [10.0.0.1]\n"+
		"  slots: 4\n"+
		"  ports:\n"+
		"    p2p:     { base: 30301, step: 1 }\n"+
		"    rpc:     { base: 8601,  step: 1 }\n"+
		"    ws:      { base: 8701,  step: 1 }\n"+
		"    auth:    { base: 8501,  step: 1 }\n"+
		"    metrics: { base: 6060,  step: 0 }\n"+
		"ssh: {user: dev, password: pw}\n")
	s := cfg.Servers[0]
	if s.Ports.WS == nil || s.Ports.WS.Base != 8701 || s.Ports.Auth.Base != 8501 || s.Ports.Metrics.Base != 6060 {
		t.Fatalf("per-purpose bands lost in load: %+v", s.Ports)
	}

	// Deriving with a step of 1 must still be refused: ws/auth would collide.
	if _, err := resource.LoadSet(write(t, "version: 2\npool:\n  hosts: [h]\n  ports: {p2p: {base: 30301, step: 1}, rpc: {base: 8601, step: 1}}\n")); err == nil {
		t.Fatal("derived scheme with rpc step 1 was accepted")
	}
}
