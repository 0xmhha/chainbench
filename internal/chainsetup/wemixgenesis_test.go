package chainsetup_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/wemix" // register the wemix plugin
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// wemixPlacement is a small placed network: a producer and two others.
func wemixPlacement(t *testing.T) *node.Map {
	t.Helper()
	m, err := resource.Assign(resource.Pool{
		Hosts:       []resource.Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots:       4,
		Ports:       resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
		Reservation: poa.Family{}.PortReservation(),
	}, []resource.Request{{Role: node.RoleBP}, {Role: node.RoleBP}, {Role: node.RoleEN}})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	return m
}

// TestWemixGenesisSource_AssemblesAValidConfigAndCallsTheBinary follows what a
// wemix genesis actually needs: a governance config the chain accepts, a
// template carrying the manifest's chain id, and the binary to write the file.
// The runner is faked so the assembly is checked without a build; the live
// generation is the family bring-up's gate.
func TestWemixGenesisSource_AssemblesAValidConfigAndCallsTheBinary(t *testing.T) {
	plugin, err := registry.Get("wemix")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	dir := t.TempDir()

	var gotArgs []string
	src := chainsetup.WemixGenesisSource{
		KeysDir: filepath.Join(repoRoot(t), "keys", "preset"),
		Binary:  "gwemix",
		WorkDir: dir,
		Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			gotArgs = args
			// Stand in for the binary: write a file where it was told to.
			out := args[len(args)-1]
			return nil, os.WriteFile(out, []byte(`{"config":{"chainId":8285},"alloc":{"0xabc":{"balance":"1"}}}`), 0o644)
		},
	}

	art, err := src.Genesis(context.Background(), plugin, chainsetup.GenesisRequest{Validators: 2, Nodes: wemixPlacement(t)})
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	// The generator is invoked as the chain documents it.
	if strings.Join(gotArgs[:2], " ") != "wemix genesis" {
		t.Fatalf("args = %v, want the wemix genesis subcommand", gotArgs)
	}
	if len(art.Genesis) == 0 {
		t.Fatal("no genesis returned")
	}

	// The config reaches the caller, because deploy-governance reads it later
	// and rebuilding it there is how two steps come to disagree.
	cfgBytes, ok := art.Extra["wemix-config.json"]
	if !ok {
		t.Fatalf("the governance config must travel with the genesis, got %v", keys(art.Extra))
	}
	var cfg poa.Config
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		t.Fatalf("config is not valid JSON: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("the chain would reject this config: %v", err)
	}
	// The boot member is the producer, named at the address it will listen on.
	m := cfg.Members[0]
	if !m.Bootnode || m.IP != "127.0.0.1" || m.Port != 31000 {
		t.Fatalf("boot member = %+v, want the producer at its placed p2p port", m)
	}
	if len(cfg.Accounts) == 0 {
		t.Fatal("no funded accounts: every test would have to arrange gas first")
	}

	// The chain id is in the template before the binary runs, because the
	// generator passes the template's config through untouched.
	tmpl, err := os.ReadFile(filepath.Join(dir, "genesis-template.json"))
	if err != nil {
		t.Fatalf("prepared template: %v", err)
	}
	if !strings.Contains(string(tmpl), "8285") {
		t.Fatalf("the prepared template does not carry the manifest chain id:\n%s", tmpl)
	}
}

func TestWemixGenesisSource_RefusesWhatItCannotDo(t *testing.T) {
	plugin, _ := registry.Get("wemix")
	keys := filepath.Join(repoRoot(t), "keys", "preset")

	// No binary: this genesis cannot be produced in Go, and saying so beats
	// returning a substituted template that starts the wrong chain.
	_, err := chainsetup.WemixGenesisSource{KeysDir: keys}.Genesis(
		context.Background(), plugin, chainsetup.GenesisRequest{Validators: 2, Nodes: wemixPlacement(t)})
	if err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("error = %v, want it to name the missing binary", err)
	}

	// No placement: the config names the producer's host and port.
	_, err = chainsetup.WemixGenesisSource{KeysDir: keys, Binary: "gwemix"}.Genesis(
		context.Background(), plugin, chainsetup.GenesisRequest{Validators: 2})
	if err == nil || !strings.Contains(err.Error(), "placement") {
		t.Fatalf("error = %v, want it to name the missing placement", err)
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
