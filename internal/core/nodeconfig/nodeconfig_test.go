package nodeconfig

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestGenerate_Validator(t *testing.T) {
	toml := string(Generate(Params{
		Role:         node.RoleValidator,
		Ports:        node.Endpoints{P2P: 30301, HTTP: 8501, WS: 9501, Auth: 8551, Metrics: 6061},
		KeystoreDir:  "/data/keystores/node1",
		RPCNamespace: "istanbul",
		StaticNodes:  []string{"enode://abc@127.0.0.1:30301?discport=0"},
	}))

	for _, want := range []string{
		`SyncMode = "full"`,
		"[Eth.Miner]", // validator gets a miner section
		`KeyStoreDir = "/data/keystores/node1"`,
		"AuthPort = 8551",
		"HTTPPort = 8501",
		"WSPort = 9501",   // WebSocket endpoint for eth_subscribe
		"WSModules = [",   // WS namespaces (same as HTTP, so "eth" is served)
		`Recommit = "2s"`, // wbft-family binaries decode Recommit from a string
		`"istanbul"`,      // namespace in HTTPModules
		`ListenAddr = ":30301"`,
		"NoDiscovery = true",
		"enode://abc@127.0.0.1:30301?discport=0",
		"Port = 6061",
	} {
		if !strings.Contains(toml, want) {
			t.Errorf("missing %q in:\n%s", want, toml)
		}
	}
}

func TestGenerate_EndpointHasNoMiner(t *testing.T) {
	toml := string(Generate(Params{
		Role:         node.RoleEndpoint,
		Ports:        node.Endpoints{P2P: 30302, HTTP: 8502},
		RPCNamespace: "wemix",
	}))
	if strings.Contains(toml, "[Eth.Miner]") {
		t.Error("endpoint should not have a miner section")
	}
	if !strings.Contains(toml, `"wemix"`) {
		t.Error("wemix namespace should appear in HTTPModules")
	}
	if !strings.Contains(toml, "StaticNodes = []") {
		t.Error("no static nodes should render an empty array")
	}
}

func TestGenerate_NanosMinerRecommit(t *testing.T) {
	toml := string(Generate(Params{
		Role:          node.RoleValidator,
		Ports:         node.Endpoints{P2P: 30301, HTTP: 8501},
		RPCNamespace:  "wemix",
		MinerRecommit: "nanos",
	}))
	// A "nanos" manifest binary decodes miner.Recommit only from an integer
	// number of nanoseconds, not a TOML string.
	if !strings.Contains(toml, "Recommit = 2000000000") {
		t.Errorf("nanos miner Recommit should be an integer, got:\n%s", toml)
	}
	if strings.Contains(toml, `Recommit = "2s"`) {
		t.Error("nanos miner Recommit must not be a string")
	}
}

// The recommit form is driven by the manifest MinerRecommit field, not by the
// chain/namespace: a "wemix" namespace with no (or "duration") MinerRecommit
// gets the default string form. This pins the de-hardcoding.
func TestGenerate_RecommitDecoupledFromNamespace(t *testing.T) {
	toml := string(Generate(Params{
		Role:         node.RoleValidator,
		Ports:        node.Endpoints{P2P: 30301, HTTP: 8501},
		RPCNamespace: "wemix", // namespace alone must NOT force nanos
	}))
	if !strings.Contains(toml, `Recommit = "2s"`) {
		t.Errorf("recommit should default to the string form regardless of namespace, got:\n%s", toml)
	}
}

func TestEnode(t *testing.T) {
	got := Enode("deadbeef", "127.0.0.1", 30303)
	want := "enode://deadbeef@127.0.0.1:30303?discport=0"
	if got != want {
		t.Errorf("Enode: got %q, want %q", got, want)
	}
}
