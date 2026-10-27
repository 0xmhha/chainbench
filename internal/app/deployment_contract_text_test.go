package app

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

// Every field the deployment contract offers has a title the editor can use as
// its label, so a field the engine adds fails here until it is named, and no
// name is kept for a field the engine dropped.
func TestDeploymentContractNamesEveryField(t *testing.T) {
	seen := map[string]bool{}
	var walk func(path string, f DeploymentField)
	walk = func(path string, f DeploymentField) {
		seen[path] = true
		if f.Title == "" {
			t.Errorf("%s has no title", path)
		}
		for key, child := range f.Properties {
			walk(path+"."+key, child)
		}
		if f.Items != nil {
			walk(path+"[]", *f.Items)
		}
		if f.Additional != nil {
			walk(path+"{}", *f.Additional)
		}
	}
	for kind, field := range DeploymentContract() {
		walk(kind, field)
	}
	for path := range deploymentFieldText {
		if !seen[path] {
			t.Errorf("%s is named but the contract has no such field", path)
		}
	}
}

// A default is offered only where the engine itself fills that value when the
// field is left out, and it is the engine's own value.
func TestDeploymentContractDefaultsAreTheEngines(t *testing.T) {
	c := DeploymentContract()
	set, config := c["server-set"], c["workspace-config"]
	ports := set.Properties["pool"].Properties["ports"]
	builtin := resource.BuiltinPorts()
	for _, c := range []struct {
		name      string
		got, want any
	}{
		{"server-set version", set.Properties["version"].Default, resource.SupportedVersion},
		{"slots", set.Properties["pool"].Properties["slots"].Default, 1},
		{"p2p base", ports.Properties["p2p"].Properties["base"].Default, builtin.P2PBase},
		{"rpc step", ports.Properties["rpc"].Properties["step"].Default, builtin.RPCStep},
		{"ssh port", set.Properties["ssh"].Properties["port"].Default, 22},
		{"workspace-config version", config.Properties["version"].Default, resource.SupportedWorkspaceVersion},
	} {
		if c.got != c.want {
			t.Errorf("%s default %v, want %v", c.name, c.got, c.want)
		}
	}
	if d := config.Properties["dataRoot"].Default; d != nil {
		t.Errorf("dataRoot has no engine default, yet the contract offers %v", d)
	}
	if config.Properties["dataRoot"].MinLength == nil || set.Properties["pool"].Properties["hosts"].Items.Properties["addr"].MinLength == nil {
		t.Error("a field the engine refuses when empty does not say so")
	}
}
