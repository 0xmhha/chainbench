package serverset_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/serverset"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "remote-server-config.yaml")
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

func load(t *testing.T, body string) *serverset.Config {
	t.Helper()
	cfg, err := serverset.Load(write(t, body))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// TestPool_ReadsTheGrid: the file's subject is the resource, and the pool is
// what the allocator is handed.
func TestPool_ReadsTheGrid(t *testing.T) {
	p := load(t, sample).Pool()
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
	if !strings.Contains(p.Source, "remote-server-config.yaml") {
		t.Errorf("source = %q, want it to name the file", p.Source)
	}
}

// TestPool_UnsetBandsFallBackToTheBuiltins keeps a minimal inventory usable:
// an operator listing addresses should not have to restate the ports.
func TestPool_UnsetBandsFallBackToTheBuiltins(t *testing.T) {
	p := load(t, "version: 2\npool:\n  hosts: [127.0.0.1]\n").Pool()
	b := serverset.BuiltinPorts()
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
	for _, s := range []serverset.Server{local, remote} {
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

func TestPlacement_LocalAndRemoteReadTheSameFields(t *testing.T) {
	cfg := load(t, sample)

	local, _ := cfg.ByName("local")
	lp := cfg.Placement(local, 1, 100)
	if lp.Mode != place.LocalStepped || lp.Remote {
		t.Errorf("local placement mode = %v remote=%v", lp.Mode, lp.Remote)
	}
	if lp.Capacity.SlotsPerHost != 8 || lp.DataRoot != "/var/lib/chainbench" {
		t.Errorf("local placement = %+v", lp)
	}

	remoteSrv, _ := cfg.ByName("bp1")
	rp := cfg.Placement(remoteSrv, 1, 100)
	if rp.Mode != place.RemotePerHost || !rp.Remote {
		t.Errorf("remote placement mode = %v remote=%v", rp.Mode, rp.Remote)
	}
	// The port plan is read from the same fields either way — only the mode
	// and the host differ.
	if lp.Config.RPCBase != rp.Config.RPCBase || lp.Config.P2PStep != rp.Config.P2PStep {
		t.Errorf("port plans diverged: local=%+v remote=%+v", lp.Config, rp.Config)
	}
	if len(rp.Config.Hosts) != 1 || rp.Config.Hosts[0] != "10.0.0.1" {
		t.Errorf("remote placement hosts = %v", rp.Config.Hosts)
	}
	for _, p := range []serverset.Placement{lp, rp} {
		if !strings.Contains(p.Source, "remote-server-config.yaml") {
			t.Errorf("source does not name the file: %q", p.Source)
		}
	}
}

func TestBuiltin_NamesItselfAsTheSource(t *testing.T) {
	// Where a port came from must never be a guess, including when no file
	// was involved.
	p := serverset.Builtin(1, 100)
	if p.Mode != place.LocalStepped || p.Remote {
		t.Errorf("builtin placement = %+v", p)
	}
	if !strings.Contains(p.Source, "built-in") {
		t.Errorf("source = %q, want it to say built-in", p.Source)
	}
	if p.Config.RPCBase != serverset.BuiltinPorts().RPCBase {
		t.Errorf("builtin ports not used: %+v", p.Config)
	}

	pool := serverset.BuiltinPool(4)
	if len(pool.Hosts) != 1 || pool.Slots != 4 || !strings.Contains(pool.Source, "built-in") {
		t.Errorf("builtin pool = %+v", pool)
	}
}

func TestFleet_SpreadsOneNodePerHost(t *testing.T) {
	cfg := load(t, `
version: 2
pool:
  hosts: [10.0.0.1, 10.0.0.2, 10.0.0.3]
  slots: 1
ssh: {user: ubuntu}
`)
	p, err := cfg.Fleet(1, 100)
	if err != nil {
		t.Fatalf("Fleet: %v", err)
	}
	if p.Mode != place.RemotePerHost || len(p.Config.Hosts) != 3 {
		t.Errorf("fleet placement = %+v", p)
	}
	if p.Capacity.Hosts != 3 || p.Capacity.SlotsPerHost != 1 {
		t.Errorf("fleet capacity = %+v", p.Capacity)
	}
}

func TestFleet_RejectsAMixedInventory(t *testing.T) {
	// Half on this machine and half over SSH is two port regimes at once, and
	// the allocator has no way to express that.
	cfg := load(t, sample)
	if _, err := cfg.Fleet(1, 100); err == nil {
		t.Fatal("want an error for a mixed local/remote fleet")
	}
}

func TestCredentials_EnvOverridesTheFile(t *testing.T) {
	cfg := load(t, sample)
	s, _ := cfg.ByName("bp1")
	env := map[string]string{"CHAINBENCH_REMOTE_USER": "deploy", "CHAINBENCH_REMOTE_PASS": "envpass"}

	c, err := s.Credentials(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("Credentials: %v", err)
	}
	if c.User != "deploy" || c.Password != "envpass" {
		t.Errorf("env did not override the file: user=%s", c.User)
	}
	if c.Host != "10.0.0.1" || c.Port != 22 {
		t.Errorf("creds = %+v", c)
	}
}

func TestCredentials_LocalServerHasNone(t *testing.T) {
	// Asking a local server for SSH credentials is a caller mistake worth
	// naming, not something to answer with an empty struct.
	cfg := load(t, sample)
	s, _ := cfg.ByName("local")
	if _, err := s.Credentials(nil); err == nil {
		t.Fatal("want an error asking a local server for credentials")
	}
}

func TestCredentials_NoAuthErrors(t *testing.T) {
	cfg := load(t, "version: 2\npool:\n  hosts: [10.0.0.9]\nssh: {user: ubuntu}\n")
	s, _ := cfg.ByName("10.0.0.9")
	if _, err := s.Credentials(func(string) string { return "" }); err == nil {
		t.Fatal("want a no-auth error")
	}
}

func TestLoad_RejectsBadInventories(t *testing.T) {
	for _, c := range []struct{ name, body, want string }{
		{"unknown field", "version: 2\npool:\n  hosts: [h]\n  slotz: 2\n", "slotz"},
		{"no hosts", "version: 2\npool:\n  hosts: []\n", "no pool hosts"},
		{"host without address", "version: 2\npool:\n  hosts: [{name: bp1}]\n", "no addr"},
		{"wrong version", "version: 99\npool:\n  hosts: [h]\n", "version 99"},
		{"p2p step too small", "version: 2\npool:\n  hosts: [h]\n  ports: {p2p: {base: 30303, step: 1}, rpc: {base: 8545, step: 10}}\n", "p2pStep"},
		{"rpc step too small", "version: 2\npool:\n  hosts: [h]\n  ports: {p2p: {base: 30303, step: 10}, rpc: {base: 8545, step: 2}}\n", "rpcStep"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := serverset.Load(write(t, c.body))
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
	_, err := serverset.Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err == nil {
		t.Fatal("want an error for a missing file")
	}
	if !strings.Contains(err.Error(), serverset.DefaultSampleFile) {
		t.Errorf("error = %v, want it to point at the sample", err)
	}
}

// TestLoad_V1FormatSaysHowToMigrate: v2 dropped the server list, and a file
// written for v1 must be told what changed rather than failing on a field name.
func TestLoad_V1FormatSaysHowToMigrate(t *testing.T) {
	_, err := serverset.Load(write(t, "version: 1\nservers:\n  - name: bp1\n    host: 10.0.0.1\n"))
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
const DefaultSample = "remote-server-config.sample.yaml"

func TestPorts_MetricsNeedsRoomInTheStep(t *testing.T) {
	if !(serverset.Ports{RPCStep: 4}).HasMetrics() {
		t.Error("a step of 4 leaves room for metrics")
	}
	if (serverset.Ports{RPCStep: 3}).HasMetrics() {
		t.Error("a step of 3 has no room for metrics")
	}
}

func TestSampleFileParses(t *testing.T) {
	// The tracked template must stay loadable: it is the first thing an
	// operator copies.
	cfg, err := serverset.Load(filepath.Join("..", "..", serverset.DefaultSampleFile))
	if err != nil {
		t.Fatalf("the tracked sample does not load: %v", err)
	}
	if err := cfg.Pool().Validate(); err != nil {
		t.Fatalf("the tracked sample's pool does not validate: %v", err)
	}
}
