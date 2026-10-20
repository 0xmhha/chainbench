package chainsetup

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
)

func crossForkLaunchSpec() nodeconfig.Spec {
	return nodeconfig.Spec{Chain: nodeconfig.Chain{Dialect: "geth114"}, Role: node.RoleBP, DataDir: "/owned/node2", ConfigPath: "/owned/node2.toml", Ports: node.Endpoints{P2P: 31010, HTTP: 8610, WS: 8611}}
}
func TestCrossForkLaunchPreservesCurrentRoleNodeAndCommandOverrides(t *testing.T) {
	w := &Workspace{state: State{LaunchSet: map[string][]string{"all": {"maxpeers=1", "http.api=eth"}, "bp": {"maxpeers=2"}, "en": {"http.api=old-en-only"}, "node2": {"maxpeers=3", "http.api=eth,istanbul,wemix"}}, LaunchCommand: []string{"maxpeers=4"}}}
	args, err := w.crossForkNodeArgs(node.Record{Index: 2, Role: "bp"}, crossForkLaunchSpec())
	if err != nil {
		t.Fatal(err)
	}
	value := func(flag string) string {
		for i, a := range args {
			if a == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	if value("--maxpeers") != "4" || value("--http.api") != "eth,istanbul,wemix" {
		t.Fatal("fork restart lost recorded launch precedence", args)
	}
	if strings.Contains(strings.Join(args, " "), "old-en-only") {
		t.Fatal("previous role override leaked after promotion")
	}
}
func TestCrossForkLaunchRejectsInvalidRecordedOverrides(t *testing.T) {
	w := &Workspace{state: State{LaunchSet: map[string][]string{"node2": {"not-a-knob=true"}}}}
	if _, err := w.crossForkNodeArgs(node.Record{Index: 2, Role: "bp"}, crossForkLaunchSpec()); err == nil {
		t.Fatal("invalid recorded fork override ignored")
	}
}
