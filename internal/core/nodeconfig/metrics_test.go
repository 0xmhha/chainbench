package nodeconfig

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// The metric assertion is the third verification source next to log and rpc, and
// it had never been usable: nothing bound the endpoint anywhere the harness
// could reach, and the config-less launch path emitted no metrics flags at all.
// Both halves are pinned here, because the failure mode is silence — an endpoint
// that binds to the wrong interface answers no one and looks like a dead node.

// TestTOML_MetricsBindsWhereTheHarnessCanReachIt pins the bind address. It used
// to default to 127.0.0.1, which is reachable only from the node's own machine,
// so a metric assertion could never work against the remote and docker targets
// this tool exists to drive.
func TestTOML_MetricsBindsWhereTheHarnessCanReachIt(t *testing.T) {
	toml := string(TOML(Spec{
		Chain: Chain{RPCNamespace: "istanbul"},
		Role:  node.RoleValidator,
		Ports: node.Endpoints{P2P: 30301, HTTP: 8501, Metrics: 6061},
	}))
	for _, want := range []string{"[Metrics]", "Enabled = true", `HTTP = "0.0.0.0"`, "Port = 6061"} {
		if !strings.Contains(toml, want) {
			t.Errorf("config is missing %q\n%s", want, toml)
		}
	}
	if strings.Contains(toml, `HTTP = "127.0.0.1"`) {
		t.Error("metrics bound to loopback: only the node's own machine could scrape it")
	}
}

// TestTOML_MetricsHostNarrowsTheBind keeps the knob working: an operator who
// wants the old behavior says so per node.
func TestTOML_MetricsHostNarrowsTheBind(t *testing.T) {
	toml := string(TOML(Spec{
		Chain:       Chain{RPCNamespace: "istanbul"},
		Role:        node.RoleValidator,
		Ports:       node.Endpoints{P2P: 30301, HTTP: 8501, Metrics: 6061},
		MetricsHost: "127.0.0.1",
	}))
	if !strings.Contains(toml, `HTTP = "127.0.0.1"`) {
		t.Errorf("MetricsHost was ignored\n%s", toml)
	}
}

// TestArgv_SaysMetricsOnTheCommandLine covers the launch that carries no config
// file — the handoff relaunch. It read its metrics settings from a config that
// was not there, so the same network answered a metric assertion before the fork
// and not after.
func TestArgv_SaysMetricsOnTheCommandLine(t *testing.T) {
	argv, err := Argv(Spec{
		Chain:       Chain{ID: "stablenet", NetworkID: 8283},
		Role:        node.RoleValidator,
		Ports:       node.Endpoints{P2P: 30301, HTTP: 8501, WS: 9501, Metrics: 6061},
		DataDir:     "/data/node1",
		NodekeyPath: "/data/node1/nodekey",
	})
	if err != nil {
		t.Fatalf("Argv: %v", err)
	}
	got := strings.Join(argv, " ")
	for _, want := range []string{"--metrics", "--metrics.addr 0.0.0.0", "--metrics.port 6061"} {
		if !strings.Contains(got, want) {
			t.Errorf("argv is missing %q\n%s", want, got)
		}
	}
}

// TestArgv_NoMetricsPortMeansNoMetricsFlags is the other side of the same rule:
// the Metrics module refuses a port without --metrics, so a node that was never
// allocated one must not get half the pair.
func TestArgv_NoMetricsPortMeansNoMetricsFlags(t *testing.T) {
	argv, err := Argv(Spec{
		Chain:       Chain{ID: "stablenet", NetworkID: 8283},
		Role:        node.RoleValidator,
		Ports:       node.Endpoints{P2P: 30301, HTTP: 8501, WS: 9501},
		DataDir:     "/data/node1",
		NodekeyPath: "/data/node1/nodekey",
	})
	if err != nil {
		t.Fatalf("Argv: %v", err)
	}
	if got := strings.Join(argv, " "); strings.Contains(got, "--metrics") {
		t.Errorf("argv names metrics for a node with no metrics port\n%s", got)
	}
}
