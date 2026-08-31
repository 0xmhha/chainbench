package nodeconfig

import (
	"strings"
	"testing"
)

// --- Dialect ---

func TestDialectGenerations(t *testing.T) {
	g114 := Geth114()
	if _, ok := g114.Spelling(KeyRPCDeprecatedPersonal); !ok {
		t.Fatalf("geth114 must support %s", KeyRPCDeprecatedPersonal)
	}
	if _, ok := g114.Spelling(KeyBlockInterval); ok {
		t.Fatalf("geth114 must not expose wemix consensus knobs")
	}

	wemix := Geth110Wemix()
	if _, ok := wemix.Spelling(KeyRPCDeprecatedPersonal); ok {
		t.Fatalf("geth110-wemix predates the personal deprecation flag")
	}
	got, ok := wemix.Spelling(KeyBlockInterval)
	if !ok || got != "--wemix.block.interval" {
		t.Fatalf("wemix block interval spelling = %q ok=%v", got, ok)
	}
}

func TestDialectFor(t *testing.T) {
	if d := DialectFor("wemix"); d.ID != "geth110-wemix" {
		t.Fatalf("wemix -> %s", d.ID)
	}
	for _, chain := range []string{"stablenet", "wbft", "anything-else"} {
		if d := DialectFor(chain); d.ID != "geth114" {
			t.Fatalf("%s -> %s, want geth114", chain, d.ID)
		}
	}
}

// --- Args ---

func TestArgsUnsupportedKeyIsClassifiedError(t *testing.T) {
	a := NewArgs(Geth114())
	a.Set(KeyBlockInterval, "1", LayerEnv)
	if len(a.Problems()) != 1 {
		t.Fatalf("problems = %v, want exactly one", a.Problems())
	}
	if msg := a.Problems()[0].Error(); !strings.Contains(msg, "does not support") {
		t.Fatalf("problem message %q lacks classification", msg)
	}
}

func TestArgsSetIfSupportedSkipsSilently(t *testing.T) {
	a := NewArgs(Geth110Wemix())
	a.EnableIfSupported(KeyRPCDeprecatedPersonal, LayerFamily)
	if len(a.Problems()) != 0 || a.Has(KeyRPCDeprecatedPersonal) {
		t.Fatalf("harmless absence must skip: problems=%v has=%v",
			a.Problems(), a.Has(KeyRPCDeprecatedPersonal))
	}
}

func TestArgsBoolValueMismatch(t *testing.T) {
	a := NewArgs(Geth114())
	a.Set(KeyHTTP, "true", LayerEnv) // boolean via Set
	a.Enable(KeyHTTPPort, LayerEnv)  // valued via Enable
	if len(a.Problems()) != 2 {
		t.Fatalf("problems = %v, want two", a.Problems())
	}
}

func TestArgsOverrideKeepsPositionAndRecordsLayer(t *testing.T) {
	a := NewArgs(Geth114())
	a.Set(KeyHTTPPort, "8545", LayerRole)
	a.Set(KeySyncMode, "full", LayerEnv)
	a.Set(KeyHTTPPort, "9999", LayerCase) // override must not reshuffle
	want := []string{"--http.port", "9999", "--syncmode", "full"}
	if got := a.Argv(); !equal(got, want) {
		t.Fatalf("argv = %v, want %v", got, want)
	}
	if l := a.WonBy(KeyHTTPPort); l != LayerCase {
		t.Fatalf("winner = %s, want %s", l, LayerCase)
	}
}

// --- Modules ---

func TestStorageRequiresDataDir(t *testing.T) {
	if err := (Storage{}).Apply(NewArgs(Geth114())); err == nil {
		t.Fatal("empty datadir must fail")
	}
	// Relative datadirs are legal: the engine roots them in the (possibly
	// CWD-relative) session tree.
	if err := (Storage{DataDir: "rel/path"}).Apply(NewArgs(Geth114())); err != nil {
		t.Fatalf("relative datadir must pass, got %v", err)
	}
}

func TestIdentityUnlockNeedsPassword(t *testing.T) {
	err := Identity{Unlock: "0xabc"}.Apply(NewArgs(Geth114()))
	if err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("err = %v, want password requirement", err)
	}
}

func TestMetricsPortWithoutEnableFails(t *testing.T) {
	if err := (Metrics{Port: 6060}).Apply(NewArgs(Geth114())); err == nil {
		t.Fatal("metrics port without --metrics must fail")
	}
}

func TestChainExtRejectsForeignKey(t *testing.T) {
	m := ChainExt{Values: map[Key]string{KeyHTTPPort: "8545"}}
	if err := m.Apply(NewArgs(Geth110Wemix())); err == nil {
		t.Fatal("non-chainext key must fail")
	}
}

func TestChainExtOnModernDialectIsError(t *testing.T) {
	b := NewBuilder(Geth114(),
		Storage{DataDir: "/d"},
		ChainExt{Values: map[Key]string{KeyBlockInterval: "1"}},
	)
	if _, err := b.Build(); err == nil {
		t.Fatal("wemix knob on geth114 must be an explicit error, not a skip")
	}
}

// --- Builder ---

// TestBuildWbftValidatorSnapshot pins the canonical argv for the fullest local
// shape: a wbft validator with identity, endpoints, and family policy.
func TestBuildWbftValidatorSnapshot(t *testing.T) {
	b := NewBuilder(Geth114(),
		Identity{
			NodeKeyFile: "/keys/node1/nodekey", Unlock: "0xAA", PasswordFile: "/keys/password",
			AllowInsecureUnlock: true, Etherbase: "0xAA",
		},
		Storage{DataDir: "/data/node1", ConfigFile: "/data/config_node1.toml"},
		P2P{Port: 30301},
		HTTPRPC{Enabled: true, Port: 8545},
		WSRPC{Enabled: true, Port: 8546},
		RPCPolicy{DeprecatedPersonal: true, UnprotectedTxs: true},
		Mining{Mine: true},
	)
	got, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--nodekey", "/keys/node1/nodekey",
		"--unlock", "0xAA",
		"--password", "/keys/password",
		"--allow-insecure-unlock",
		"--miner.etherbase", "0xAA",
		"--datadir", "/data/node1",
		"--config", "/data/config_node1.toml",
		"--port", "30301",
		"--http",
		"--http.port", "8545",
		"--ws",
		"--ws.port", "8546",
		"--rpc.enabledeprecatedpersonal",
		"--rpc.allow-unprotected-txs",
		"--mine",
	}
	if !equal(got, want) {
		t.Fatalf("argv:\n got %v\nwant %v", got, want)
	}
}

func TestBuildCrossChecks(t *testing.T) {
	// unlock without allow-insecure-unlock
	b := NewBuilder(Geth114(),
		Identity{Unlock: "0xAA", PasswordFile: "/p"},
		Storage{DataDir: "/d"},
		HTTPRPC{Enabled: true},
	)
	if _, err := b.Build(); err == nil || !strings.Contains(err.Error(), "allow-insecure-unlock") {
		t.Fatalf("err = %v, want allow-insecure-unlock rule", err)
	}
	// api on disabled endpoint reaches Build through an override
	b = NewBuilder(Geth114(), Storage{DataDir: "/d"}).
		WithOverrides(Override{Key: KeyHTTPAPI, Value: "eth"})
	if _, err := b.Build(); err == nil || !strings.Contains(err.Error(), "http.api") {
		t.Fatalf("err = %v, want http.api rule", err)
	}
}

func TestBuildJoinsAllProblems(t *testing.T) {
	b := NewBuilder(Geth114(),
		Identity{Unlock: "0xAA"}, // missing password
		Storage{},                // missing datadir
		Metrics{Port: 6060},      // port without enable
	)
	_, err := b.Build()
	if err == nil {
		t.Fatal("want joined errors")
	}
	for _, frag := range []string{"password", "datadir", "metrics"} {
		if !strings.Contains(err.Error(), frag) {
			t.Fatalf("joined error %q lacks %q", err, frag)
		}
	}
}

func TestBuildOverridesWin(t *testing.T) {
	b := NewBuilder(Geth114(),
		Storage{DataDir: "/d"},
		P2P{Port: 30301},
	).WithOverrides(
		Override{Key: KeyPort, Value: "40000"},                  // replaces, keeps position
		Override{Key: KeyMaxPeers, Value: "9", Layer: LayerEnv}, // appends
	)
	got, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--datadir", "/d", "--port", "40000", "--maxpeers", "9"}
	if !equal(got, want) {
		t.Fatalf("argv = %v, want %v", got, want)
	}
}

// --- Family shim ---

func TestParseFamilyFlags(t *testing.T) {
	p, err := ParseFamilyFlags([]string{
		"--allow-insecure-unlock", "--rpc.enabledeprecatedpersonal",
		"--rpc.allow-unprotected-txs", "--mine",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !p.AllowInsecureUnlock || !p.DeprecatedPersonal || !p.UnprotectedTxs || !p.Mine {
		t.Fatalf("policy = %+v", p)
	}
	if _, err := ParseFamilyFlags([]string{"--verbosity"}); err == nil {
		t.Fatal("unknown family flag must fail")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
