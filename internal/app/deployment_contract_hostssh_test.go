package app

import "testing"

// TestDeploymentContractOffersNoPerHostSSH: the Web binds each server's login
// to a personal credential, and shared documents refuse any host field besides
// name and addr. The editor contract therefore offers no per-host ssh block,
// which would otherwise invite a login into a shared document.
func TestDeploymentContractOffersNoPerHostSSH(t *testing.T) {
	hosts := DeploymentContract()["server-set"].Properties["pool"].Properties["hosts"]
	if hosts.Items == nil {
		t.Fatal("the contract lost the host item")
	}
	if _, ok := hosts.Items.Properties["ssh"]; ok {
		t.Fatal("the contract offers a per-host ssh block")
	}
	for _, field := range []string{"name", "addr"} {
		if _, ok := hosts.Items.Properties[field]; !ok {
			t.Fatalf("the contract lost host %s", field)
		}
	}
}
