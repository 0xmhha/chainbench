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

// sample is a mixed inventory: this machine hosting several nodes, and two
// remote hosts, one of which overrides the defaults.
const sample = `
version: 1
defaults:
  slots: 1
  dataRoot: /var/lib/chainbench
  ports:
    p2pBase: 30303
    p2pStep: 10
    rpcBase: 8545
    rpcStep: 10
  ssh:
    user: ubuntu
    port: 22
    password: filepass
servers:
  - index: 1
    name: local
    kind: local
    host: 127.0.0.1
    slots: 8
    dataRoot: /tmp/cb
  - index: 2
    name: bp1
    kind: remote
    host: 10.0.0.1
  - index: 7
    name: bp7
    kind: remote
    host: 10.0.0.7
    dataRoot: /srv/chainbench
    ports:
      rpcBase: 9545
    ssh:
      user: root
      port: 2222
`

func load(t *testing.T, body string) *serverset.Config {
	t.Helper()
	cfg, err := serverset.Load(write(t, body))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func TestServer_InheritsDefaultsAndAppliesOverrides(t *testing.T) {
	cfg := load(t, sample)

	bp1, err := cfg.Server(2)
	if err != nil {
		t.Fatalf("Server(2): %v", err)
	}
	if bp1.DataRoot != "/var/lib/chainbench" || bp1.Ports.RPCBase != 8545 || bp1.Slots != 1 {
		t.Errorf("bp1 did not inherit the defaults: %+v", bp1)
	}

	bp7, err := cfg.Server(7)
	if err != nil {
		t.Fatalf("Server(7): %v", err)
	}
	// An override replaces only the field it names; the rest still inherits.
	if bp7.Ports.RPCBase != 9545 {
		t.Errorf("bp7 rpcBase = %d, want the override 9545", bp7.Ports.RPCBase)
	}
	if bp7.Ports.P2PBase != 30303 || bp7.Ports.RPCStep != 10 {
		t.Errorf("bp7 lost the inherited port fields: %+v", bp7.Ports)
	}
	if bp7.DataRoot != "/srv/chainbench" {
		t.Errorf("bp7 dataRoot = %q", bp7.DataRoot)
	}
}

func TestServer_UnsetPortsFallBackToTheBuiltins(t *testing.T) {
	// An inventory that names only hosts still yields a usable plan, so a
	// config file is never a wall of ports an operator has to copy.
	cfg := load(t, "version: 1\nservers:\n  - name: local\n    host: 127.0.0.1\n")
	s, err := cfg.ByName("local")
	if err != nil {
		t.Fatalf("ByName: %v", err)
	}
	if s.Ports != serverset.BuiltinPorts() {
		t.Errorf("ports = %+v, want the built-ins %+v", s.Ports, serverset.BuiltinPorts())
	}
}

func TestSelect_ByNameByIndexAndTheLoneServer(t *testing.T) {
	cfg := load(t, sample)

	if s, err := cfg.Select("bp7", 0); err != nil || s.Host != "10.0.0.7" {
		t.Errorf("Select by name = %+v, %v", s, err)
	}
	if s, err := cfg.Select("", 1); err != nil || s.Name != "local" {
		t.Errorf("Select by index = %+v, %v", s, err)
	}
	// With several servers and no selector, the caller has to say which.
	if _, err := cfg.Select("", 0); err == nil {
		t.Error("want an error selecting nothing from a multi-server inventory")
	}
	// With exactly one, there is nothing to disambiguate.
	one := load(t, "version: 1\nservers:\n  - name: only\n    host: 127.0.0.1\n")
	if s, err := one.Select("", 0); err != nil || s.Name != "only" {
		t.Errorf("Select on a lone server = %+v, %v", s, err)
	}
}

func TestPlacement_LocalAndRemoteReadTheSameFields(t *testing.T) {
	cfg := load(t, sample)

	local, _ := cfg.ByName("local")
	lp := cfg.Placement(local, 1, 100)
	if lp.Mode != place.LocalStepped || lp.Remote {
		t.Errorf("local placement mode = %v remote=%v", lp.Mode, lp.Remote)
	}
	if lp.Capacity.SlotsPerHost != 8 || lp.DataRoot != "/tmp/cb" {
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
	// Both name where the plan came from.
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
}

func TestFleet_SpreadsOneNodePerHost(t *testing.T) {
	cfg := load(t, `
version: 1
defaults:
  ssh: {user: ubuntu}
servers:
  - name: bp1
    kind: remote
    host: 10.0.0.1
  - name: bp2
    kind: remote
    host: 10.0.0.2
  - name: bp3
    kind: remote
    host: 10.0.0.3
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
	cfg := load(t, "version: 1\nservers:\n  - name: r\n    kind: remote\n    host: h\n    ssh: {user: ubuntu}\n")
	s, _ := cfg.ByName("r")
	if _, err := s.Credentials(func(string) string { return "" }); err == nil {
		t.Fatal("want a no-auth error")
	}
}

func TestLoad_RejectsBadInventories(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"unknown field", "version: 1\nservers:\n  - name: a\n    host: h\n    passwrd: x\n", "passwrd"},
		{"duplicate index", "version: 1\nservers:\n  - index: 1\n    host: a\n  - index: 1\n    host: b\n", "duplicate server index"},
		{"duplicate name", "version: 1\nservers:\n  - name: a\n    host: x\n  - name: a\n    host: y\n", "duplicate server name"},
		{"no host", "version: 1\nservers:\n  - index: 1\n", "no host"},
		{"no selector", "version: 1\nservers:\n  - host: h\n", "index or a name"},
		{"empty inventory", "version: 1\nservers: []\n", "no servers"},
		{"bad kind", "version: 1\nservers:\n  - name: a\n    host: h\n    kind: sideways\n", "want local or remote"},
		{"wrong version", "version: 99\nservers:\n  - name: a\n    host: h\n", "version 99"},
		// A p2p step of 1 puts the derived etcd port on the next node's p2p
		// port; that stalls block production with no obvious cause.
		{"p2p step too small", "version: 1\ndefaults:\n  ports: {p2pStep: 1}\nservers:\n  - name: a\n    host: h\n", "p2pStep"},
		{"rpc step too small", "version: 1\ndefaults:\n  ports: {rpcStep: 2}\nservers:\n  - name: a\n    host: h\n", "rpcStep"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := serverset.Load(write(t, tc.body))
			if err == nil {
				t.Fatalf("want an error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestLoad_MissingFilePointsAtTheSample(t *testing.T) {
	_, err := serverset.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("want a not-found error")
	}
	if !strings.Contains(err.Error(), serverset.DefaultSampleFile) {
		t.Errorf("error should point at the sample, got: %v", err)
	}
}

func TestLoad_LegacyFormatSaysHowToMigrate(t *testing.T) {
	// The decoder's own "field not found" says nothing useful about a file
	// written for the previous format.
	_, err := serverset.Load(write(t, "user: ubuntu\nport: 22\nservers:\n  - index: 1\n    host: h\n"))
	if err == nil {
		t.Fatal("want an error for the pre-v1 format")
	}
	for _, want := range []string{"defaults.ssh", "version: 1", "kind:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("migration hint missing %q: %v", want, err)
		}
	}
}

func TestPorts_MetricsNeedsRoomInTheStep(t *testing.T) {
	tight := serverset.Ports{P2PBase: 1, P2PStep: 2, RPCBase: 1, RPCStep: 3}
	if tight.HasMetrics() {
		t.Error("an rpc step of 3 has no room for metrics")
	}
	if !serverset.BuiltinPorts().HasMetrics() {
		t.Error("the built-in plan should leave room for metrics")
	}
}

// TestSampleFileParses keeps the tracked template honest: an operator copies it
// verbatim, so it has to load.
func TestSampleFileParses(t *testing.T) {
	cfg, err := serverset.Load(filepath.Join("..", "..", serverset.DefaultSampleFile))
	if err != nil {
		t.Fatalf("the tracked sample does not load: %v", err)
	}
	local, err := cfg.ByName("local")
	if err != nil {
		t.Fatalf("sample has no local server: %v", err)
	}
	if local.IsRemote() {
		t.Error("the sample's local server reports as remote")
	}
	if _, err := cfg.ByName("bp1"); err != nil {
		t.Errorf("sample has no remote server: %v", err)
	}
}
