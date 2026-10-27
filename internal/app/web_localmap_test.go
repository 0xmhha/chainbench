package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func dockerSet(localmap string) DeploymentDocumentInput {
	content := `{"version":2,"pool":{"hosts":[{"name":"s1","addr":"172.30.0.11"},{"name":"s2","addr":"172.30.0.12"}],"slots":1,"ports":{"p2p":{"base":30301,"step":3},"rpc":{"base":8601,"step":10}}},"ssh":{"port":10022}` + localmap + `}`
	return DeploymentDocumentInput{Kind: "server-set", Name: "docker", ContractVersion: "2", AssetRefs: []string{}, Content: json.RawMessage(content)}
}

// A server pool on docker containers keeps the containers' real addresses and
// carries the map of how this machine reaches them. The map may name only the
// pool's own addresses, with ports a machine can listen on.
func TestServerSetLocalMapIsChecked(t *testing.T) {
	ok := dockerSet(`,"localmap":{"hosts":{"172.30.0.11":{"host":"127.0.0.1","ports":{"10022":2201,"8601":18601}},"172.30.0.12":{"host":"127.0.0.1","ports":{"10022":2202}}}}`)
	if err := ValidateDeploymentDocument(ok); err != nil {
		t.Fatalf("a valid localmap was refused: %v", err)
	}
	for want, localmap := range map[string]string{
		"/content/localmap/hosts/10.9.9.9":          `,"localmap":{"hosts":{"10.9.9.9":{"host":"127.0.0.1","ports":{}}}}`,
		"/content/localmap/hosts/172.30.0.11/ports": `,"localmap":{"hosts":{"172.30.0.11":{"host":"127.0.0.1","ports":{"10022":70000}}}}`,
		"/content/localmap/hosts/172.30.0.11/host":  `,"localmap":{"hosts":{"172.30.0.11":{"host":"","ports":{}}}}`,
		"/content/localmap/hosts/172.30.0.11":       `,"localmap":{"hosts":{"172.30.0.11":{"host":"127.0.0.1","ports":{},"user":"x"}}}`,
		"/content/localmap/hosts":                   `,"localmap":{"hosts":{}}`,
	} {
		err := ValidateDeploymentDocument(dockerSet(localmap))
		var field *WebFieldError
		if !errors.As(err, &field) || field.Path != want {
			t.Errorf("%s: %v", want, err)
		}
	}
}

// The engine reads the server set and its localmap as two files side by side;
// the server set it reads has no localmap field.
func TestWebServerSetIsWrittenAsTheEnginesTwoFiles(t *testing.T) {
	dir := t.TempDir()
	setPath := filepath.Join(dir, "server-set.yaml")
	doc := dockerSet(`,"localmap":{"hosts":{"172.30.0.11":{"host":"127.0.0.1","ports":{"10022":2201}}}}`)
	if !webDocker(doc) || webDocker(dockerSet("")) {
		t.Fatal("docker mode does not follow the localmap")
	}
	if err := writeWebServerSet(context.Background(), doc, setPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(setPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = resource.ParseSet(raw); err != nil {
		t.Fatalf("the engine cannot read the written server set: %v", err)
	}
	m, err := resource.LoadLocalMap(resource.LocalMapNear(setPath))
	if err != nil {
		t.Fatal(err)
	}
	if h, p := m.AddrMap(nil)("172.30.0.11", 10022); h != "127.0.0.1" || p != 2201 {
		t.Fatalf("mapped to %s:%d", h, p)
	}
	plain := filepath.Join(t.TempDir(), "server-set.yaml")
	if err = writeWebServerSet(context.Background(), dockerSet(""), plain); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(resource.LocalMapNear(plain)); !os.IsNotExist(err) {
		t.Fatalf("a pool without a localmap wrote one: %v", err)
	}
}

// The dashboard's own reads of a docker node's RPC and metrics go through the
// localmap the network was composed with, as the engine's dials do.
func TestMonitorReachesADockerNodeThroughItsLocalMap(t *testing.T) {
	f := newMonitorFixture(t)
	rpcPort, metricsPort := 0, 0
	{
		raw, err := os.ReadFile(filepath.Join(f.root, "networks", f.network, "chain-record.json"))
		if err != nil {
			t.Fatal(err)
		}
		var state State
		if err = json.Unmarshal(raw, &state); err != nil {
			t.Fatal(err)
		}
		rpcPort, metricsPort = state.Nodes[0].HTTP, state.Nodes[0].Metrics
	}
	set := filepath.Join(t.TempDir(), "server-set.yaml")
	if err := os.WriteFile(resource.LocalMapNear(set), []byte(fmt.Sprintf("hosts:\n  192.0.2.11: {host: 127.0.0.1, ports: {8601: %d, 6060: %d}}\n", rpcPort, metricsPort)), 0o600); err != nil {
		t.Fatal(err)
	}
	ns := node.Record{Index: 1, Label: "node1", Role: "bp", Host: "192.0.2.11", PID: 1, Endpoints: node.Endpoints{HTTP: 8601, Metrics: 6060}}
	reach := resource.Opener{ServerSet: set, Docker: true}.HTTPEndpoint
	samples, gaps := f.monitor().sampleNode(context.Background(), reach, ns, f.clock, f.clock.Add(-5*time.Second))
	sources := map[string]bool{}
	for _, s := range samples {
		sources[s.Source] = true
	}
	for _, g := range gaps {
		if strings.HasSuffix(g.Reason, "_unavailable") {
			t.Errorf("unreached: %+v", g)
		}
	}
	if !sources["rpc"] || !sources["metrics"] {
		t.Fatalf("samples %v", samples)
	}
	// Without the map the same node is unreachable, which is what the map is for.
	plain := resource.Opener{ServerSet: set}.HTTPEndpoint
	if _, gaps = f.monitor().sampleNode(context.Background(), plain, ns, f.clock, f.clock.Add(-5*time.Second)); len(gaps) == 0 {
		t.Fatal("a docker node was reached without its localmap")
	}
}

// A pool with a localmap composes, plans, runs and checks access in docker
// mode; a pool without one never does.
func TestADockerPoolRunsItsCommandsInDockerMode(t *testing.T) {
	docker := dockerSet(`,"localmap":{"hosts":{"172.30.0.11":{"host":"127.0.0.1","ports":{"10022":2201}}}}`)
	for _, c := range []struct {
		set  DeploymentDocumentInput
		want int
	}{{docker, 1}, {dockerSet(""), 0}} {
		p := webChainPayload{Set: DeploymentDocument{DeploymentDocumentInput: c.set}, ControlDir: "/control", ExecutionBinary: "/bin/gstable"}
		up := chainUpArgs(p, "start", "/k", "/s", "/w", []string{"--chain", "stablenet"}, "")
		p.TestRun = &webTestRun{Content: []json.RawMessage{json.RawMessage(`{}`)}, InputDir: "/inputs"}
		run, err := suiteCLIArgs(context.Background(), (&WebChainEngine{root: "/root"}).webSuiteInput(p), t.TempDir(), "")
		if err != nil {
			t.Fatal(err)
		}
		if count(up, "--docker") != c.want || count(run, "--docker") != c.want {
			t.Errorf("localmap %v: chain up %v, run %v", webDocker(c.set), up, run)
		}
	}
}
