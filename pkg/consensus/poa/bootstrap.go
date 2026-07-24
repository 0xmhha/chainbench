package poa

import "github.com/0xmhha/chainbench/pkg/core/node"

// Step is one action in the poa (wemix) bootstrap sequence.
type Step struct {
	// Name is a short identifier.
	Name string
	// Detail describes what the step does.
	Detail string
	// OnBootNode is true for steps that run only on the boot node.
	OnBootNode bool
}

// BootstrapPlan returns the ordered wemix bring-up sequence, mirroring
// ../script/wemix-upgrade: initialize the boot node, deploy the governance
// contracts, initialize the etcd cluster, then start the remaining nodes. It is
// returned as data so the setup phase (and tests) can inspect it; executing the
// steps requires a built gwemix binary and etcd and is wired in a later slice
// (docs/CHAINBENCH_GO_REDESIGN.md §3.4, §11).
func BootstrapPlan() []Step {
	return []Step{
		{Name: "init-boot", Detail: "initialize the boot node datadir from genesis", OnBootNode: true},
		{Name: "start-boot", Detail: "start the boot node", OnBootNode: true},
		{Name: "deploy-governance", Detail: "deploy governance contracts via the boot node", OnBootNode: true},
		{Name: "init-etcd", Detail: "initialize the etcd cluster membership", OnBootNode: true},
		{Name: "start-nodes", Detail: "initialize and start the remaining nodes"},
	}
}

// BootRole reports whether a role acts as the wemix boot node (governance
// deploy + etcd init happen here).
func BootRole(r node.Role) bool {
	return r == node.RoleBoot || r == node.RoleValidator
}
