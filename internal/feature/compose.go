package feature

import "github.com/0xmhha/chainbench/internal/app"

// The compose stage: the nine steps that build and run a network, in the order
// they resolve (chainsetup's composeNeeds says the same thing for the workspace
// side).
//
// They register rather than move: each already goes through app, so what the
// registry adds is one place that knows the feature exists, what it is called,
// what it takes, and whether it only looks. The surfaces still render them by
// hand today; S1's own gate is that `chain up` is unchanged on three chains,
// and TestComposeFeatures_TagsMatchTheCommands holds the tags to the flags the
// commands declare so the later switch is a deletion rather than a change.
func init() {
	Register(Registration{
		Name: "chain.new", Stage: StageCompose,
		Summary: "Initialize a composition workspace for a chain (and its target)",
	}, app.NetNew)
	Register(Registration{
		Name: "chain.place", Stage: StageCompose,
		Summary: "Build the node table: roles, hosts, deterministic non-colliding ports",
	}, app.NetAllocate)
	Register(Registration{
		Name: "chain.keys", Stage: StageCompose,
		Summary: "Ensure the key set exists and covers the node count (preset or generate)",
	}, app.NetKeys)
	Register(Registration{
		Name: "chain.genesis", Stage: StageCompose,
		Summary: "Build the genesis from the key set and write it to the target",
	}, app.NetGenesis)
	Register(Registration{
		Name: "chain.config", Stage: StageCompose,
		Summary: "Render each node's TOML config",
	}, app.NetConfig)
	Register(Registration{
		Name: "chain.build", Stage: StageCompose,
		Summary: "Assemble each node's launch command without running it",
	}, app.NetLaunchOpts)
	Register(Registration{
		Name: "chain.deploy", Stage: StageCompose,
		Summary: "Put the launch inputs on the target and verify they are present",
	}, app.NetProvision)
	Register(Registration{
		Name: "chain.init", Stage: StageCompose,
		Summary: "Initialize each node's datadir from the built genesis",
	}, app.NetInit)
	Register(Registration{
		Name: "chain.start", Stage: StageCompose,
		Summary: "Launch every stopped node and record its PID",
	}, app.NetStart)

	// Reads of a composed network. They are the compose stage's queries, and
	// the same declaration the CLI's query group and MCP's read-only tool list
	// are meant to converge on.
	Register(Registration{
		Name: "chain.status", Stage: StageCompose, ReadOnly: true,
		Summary: "Show the workspace composition state and which steps have run",
	}, app.NetStatus)
	Register(Registration{
		Name: "chain.health", Stage: StageCompose, ReadOnly: true,
		Summary: "Probe every node's HTTP RPC for its latest block",
	}, app.NetHealth)
	Register(Registration{
		Name: "chain.verify-validators", Stage: StageCompose, ReadOnly: true,
		Summary: "Check the running chain recognizes exactly the composed keys as its validators",
	}, app.VerifyValidators)
	Register(Registration{
		Name: "chain.enode", Stage: StageCompose, ReadOnly: true,
		Summary: "Show each node's enode (derived from keys and place; writes nothing)",
	}, app.NetEnodes)
	Register(Registration{
		Name: "chain.logs", Stage: StageCompose, ReadOnly: true,
		Summary: "Show the last lines of one node's log",
	}, app.NetLogs)

}
