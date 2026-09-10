// Package chainsetup composes a chain network: it turns a declaration into a
// running set of nodes, one named step at a time, over a workspace that records
// what has been done.
//
// The order is fixed and is the one list `chain up` and `chain resume` share
// (upStepNames):
//
//	new -> place -> keys -> genesis -> config -> build -> deploy -> init -> start
//
// place comes before keys because a governance member carries the ip and port
// that placement hands out; deriving the genesis before the layout existed is
// the ordering bug the wemix bring-up was built on and had to correct.
//
// Every step is also a command and an MCP tool of the same name, so a step can
// be run, re-run, and inspected on its own. Re-running is the normal case:
// steps are written to be idempotent against what the workspace already
// records, which is what lets `chain resume` continue a composition whose
// process died rather than starting over.
//
// The workspace (control state) is always local; a step's data plane — files
// and processes — lives on the target, chosen once at `chain new` and reached
// through core/filestore and core/process. That split is why the same steps run
// unchanged against this machine, a docker container standing in for a server,
// and a real SSH host.
//
// Allocation is the one moment two runs can hand out the same port slot: each
// derives the inventory from the workspaces it can see, and two runs that look
// before either has saved both see it free. NetAllocate holds the server set's
// lock from the look to the save — a lock, not a second record, so the
// workspaces stay the only account of what is taken.
package chainsetup
