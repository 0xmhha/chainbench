package external_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chains/external"
)

// baseManifest is a valid external manifest on the wbft family borrowing the
// stablenet protocol, with %s holes for the fields a test varies.
func writeManifest(t *testing.T, family, proto, template string) string {
	t.Helper()
	dir := t.TempDir()
	tmplField := ""
	if template != "" {
		tmplField = `"template": "` + template + `"`
	}
	protoField := ""
	if proto != "" {
		protoField = `"protocol": "` + proto + `",`
	}
	m := `{
		"id": "foonet", "binary": "gfoo", "chain_id": 9999, "network_id": 9999,
		"miner_recommit": "duration", "bootstrap": {"type": "static"},
		"consensus_family": "` + family + `", ` + protoField + `
		"genesis": {` + tmplField + `},
		"consensus": {"rpc_namespace": "istanbul", "validators_method": "istanbul_getValidators"},
		"probe": {"method": "istanbul_getValidators"}
	}`
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(m), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ExternalChain(t *testing.T) {
	mpath := writeManifest(t, "wbft", "stablenet", "foonet-genesis")
	tpath := filepath.Join(t.TempDir(), "genesis.json")
	if err := os.WriteFile(tpath, []byte(`{"config":{"chainId":9999}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := external.Load(mpath, tpath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.Manifest().ID != "foonet" || p.Manifest().Binary != "gfoo" {
		t.Errorf("manifest: %+v", p.Manifest())
	}
	if p.Family().ID() != "wbft" {
		t.Errorf("family: %q, want wbft", p.Family().ID())
	}
	if p.Protocol().Name != "stablenet" {
		t.Errorf("protocol: %q, want stablenet (borrowed)", p.Protocol().Name)
	}
	if !strings.Contains(string(p.GenesisTemplate()), "9999") {
		t.Errorf("genesis template not loaded: %q", p.GenesisTemplate())
	}
}

func TestLoad_ProtocolDefaultsToID(t *testing.T) {
	// no "protocol" field, id "wbft" -> resolves the wbft SDK protocol by id.
	dir := t.TempDir()
	m := `{"id":"wbft","binary":"gwbft","chain_id":8284,"network_id":8284,
		"miner_recommit":"duration","bootstrap":{"type":"static"},"consensus_family":"wbft",
		"genesis":{},"consensus":{"rpc_namespace":"istanbul","validators_method":"istanbul_getValidators"},
		"probe":{"method":"istanbul_getValidators"}}`
	path := filepath.Join(dir, "m.json")
	_ = os.WriteFile(path, []byte(m), 0o644)
	p, err := external.Load(path, "")
	if err != nil || p.Protocol().Name != "wbft" {
		t.Fatalf("protocol should default to id: %v (%v)", p.Protocol().Name, err)
	}
}

func TestLoad_Errors(t *testing.T) {
	if _, err := external.Load("/no/such/manifest.json", ""); err == nil {
		t.Error("missing manifest file should error")
	}
	// unknown consensus family.
	if _, err := external.Load(writeManifest(t, "raft", "stablenet", ""), ""); err == nil ||
		!strings.Contains(err.Error(), "family") {
		t.Errorf("unknown family should error: %v", err)
	}
	// unknown accounts protocol.
	if _, err := external.Load(writeManifest(t, "wbft", "nosuchproto", ""), ""); err == nil ||
		!strings.Contains(err.Error(), "protocol") {
		t.Errorf("unknown protocol should error: %v", err)
	}
	// declares a template but no --genesis-template path.
	if _, err := external.Load(writeManifest(t, "wbft", "stablenet", "foonet-genesis"), ""); err == nil ||
		!strings.Contains(err.Error(), "genesis-template") {
		t.Errorf("template-declared-without-path should error: %v", err)
	}
}
