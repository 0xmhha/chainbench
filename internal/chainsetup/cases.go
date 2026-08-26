// Package chainsetup models how a chain network is brought up, as data: the
// ordered steps of each supported case and the customization points each step
// exposes. The `chainbench chain` command executes that model step by step, so
// "which step is broken" is answerable without reading the orchestration code.
//
// The cases mirror docs/dev/chain-setup/. Keeping them here rather than only in
// prose means the CLI and the documents cannot drift apart silently: a step
// renamed in code shows up in `chain steps` immediately.
package chainsetup

// Support says how far a case is known to work, measured rather than assumed.
type Support string

const (
	// Supported means a live bring-up has been observed end to end.
	Supported Support = "supported"
	// Partial means the bring-up starts but does not reach a producing chain.
	Partial Support = "partial"
	// Unsupported means no orchestration path exists yet.
	Unsupported Support = "unsupported"
)

// CaseStep is one action in a case's bring-up sequence.
type CaseStep struct {
	// ID is the stable identifier used by --stop-after and in reports.
	ID string
	// Title is a short imperative description.
	Title string
	// Detail says what actually happens, including the boundary that owns it.
	Detail string
	// Implemented is false for a step the framework does not perform yet; the
	// executor stops there with that fact stated rather than pretending.
	Implemented bool
}

// Knob is one customization point: what can be changed, where it is set, and
// what it affects.
type Knob struct {
	Name   string
	Where  string
	Effect string
}

// Case is one way of standing a network up.
type Case struct {
	ID    string
	Title string
	// Bootstrap is the manifest bootstrap.type the case follows.
	Bootstrap string
	// Support is the measured state, with Note giving the reason when it is not
	// Supported.
	Support Support
	Note    string
	// Binaries lists the executables the case needs, in the caller's terms.
	Binaries []string
	// Doc is the companion document under docs/dev/chain-setup/.
	Doc   string
	Steps []CaseStep
	Knobs []Knob
}

// StepIndex returns the position of the step with the given id, or -1.
func (c Case) StepIndex(id string) int {
	for i, s := range c.Steps {
		if s.ID == id {
			return i
		}
	}
	return -1
}

// Cases returns every modelled bring-up case, in document order.
func Cases() []Case {
	return []Case{wemixCase(), handoffCase(), wbftCase(), stablenetCase()}
}

// Find returns the case with the given id.
func Find(id string) (Case, bool) {
	for _, c := range Cases() {
		if c.ID == id {
			return c, true
		}
	}
	return Case{}, false
}

// IDs lists the known case ids, for error messages and shell completion.
func IDs() []string {
	cs := Cases()
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

// staticSteps is the bring-up shared by chains whose genesis already carries the
// validator set, so consensus holds the moment the nodes start.
func staticSteps() []CaseStep {
	return []CaseStep{
		{ID: "resolve-chain", Title: "Resolve the chain plugin", Detail: "registry.Get: manifest, chain id, consensus family, capabilities", Implemented: true},
		{ID: "resolve-binary", Title: "Resolve the node binary", Detail: "explicit --binary, else the manifest binary on PATH", Implemented: true},
		{ID: "load-preset", Title: "Load the key preset", Detail: "keys.LoadPreset: nodekeys, keystores, validator set, BLS keys, alloc", Implemented: true},
		{ID: "allocate", Title: "Allocate hosts and ports", Detail: "place.Allocate with a capacity check (validator minimum, port band)", Implemented: true},
		{ID: "genesis", Title: "Build the genesis", Detail: "PresetGenesisSource: chain template with the preset's validator set substituted", Implemented: true},
		{ID: "assemble-plan", Title: "Assemble the launch plan", Detail: "engine.AssemblePlan: per-node datadir, config path, launch args", Implemented: true},
		{ID: "arm", Title: "Arm each node", Detail: "render config, install identity flags, resolve the binary per node", Implemented: true},
		{ID: "provision", Title: "Materialize files", Detail: "genesis + per-node config through the file sink (upload-if-absent)", Implemented: true},
		{ID: "launch", Title: "Init datadirs and launch", Detail: "per-node init from the shared genesis, then start and track each process", Implemented: true},
		{ID: "health-gate", Title: "Gate on health", Detail: "poll until the head advances — proof the network produces blocks", Implemented: true},
	}
}

// staticKnobs are the customization points shared by the static-bootstrap cases.
func staticKnobs() []Knob {
	return []Knob{
		{Name: "validators", Where: "--validators, spec topology.bp", Effect: "block-producing node count (BFT needs 4 to tolerate one fault)"},
		{Name: "endpoints", Where: "--endpoints, spec topology.en", Effect: "non-producing RPC nodes"},
		{Name: "topology file", Where: "setup --topology <yaml>", Effect: "explicit per-node role, sync mode, and bootnode"},
		{Name: "sync mode", Where: "topology sync_mode", Effect: "full | snap | archive, per node"},
		{Name: "key preset", Where: "--keys (default keys/preset)", Effect: "node identities and the genesis validator set"},
		{Name: "preset size", Where: "chainbench keys generate --nodes N --validators V", Effect: "networks larger than the committed 5-node preset"},
		{Name: "genesis overrides", Where: "--set genesis.overrides.<key>=<v>", Effect: "single genesis config values, e.g. a delayed hardfork block"},
		{Name: "genesis overlay", Where: "--genesis-overlay <json>", Effect: "deep-merged genesis fragment plus advertised capabilities"},
		{Name: "port plan", Where: "ports.base_p2p/base_rpc and their steps", Effect: "port layout; p2p_step >= 2 (etcd is p2p+1), rpc_step >= 3 (ws, auth)"},
		{Name: "placement mode", Where: "place.Mode", Effect: "LocalStepped | LocalOSAssigned | RemotePerHost"},
		{Name: "remote host", Where: "--remote-host/--remote-user (+ SSH credentials)", Effect: "run the nodes on another machine over SSH"},
		{Name: "artifact root", Where: "--artifact-root", Effect: "where session artifacts and logs are written"},
	}
}

func stablenetCase() Case {
	return Case{
		ID: "stablenet", Title: "gstable only",
		Bootstrap: "static", Support: Supported,
		Binaries: []string{"gstable (go-stablenet: make gstable)"},
		Doc:      "docs/dev/chain-setup/case-4-stablenet.md",
		Steps:    staticSteps(),
		Knobs: append(staticKnobs(),
			Knob{Name: "boho fork block", Where: "--set genesis.overrides.bohoBlock=N", Effect: "delayed-hardfork scenarios"},
			Knob{Name: "account extra bits", Where: "--genesis-overlay internal/chains/stablenet/overlays/account-extra.json", Effect: "seeds authorized/blacklisted account state"},
		),
	}
}

func wbftCase() Case {
	return Case{
		ID: "wbft", Title: "gwbft only",
		Bootstrap: "static", Support: Supported,
		Note:     "the go-wbft make target produces a binary named gwemix, so --binary must be an explicit path",
		Binaries: []string{"gwemix from go-wbft (make gwemix)"},
		Doc:      "docs/dev/chain-setup/case-3-wbft.md",
		Steps:    staticSteps(),
		Knobs: append(staticKnobs(),
			Knob{Name: "useNCP governance", Where: "--genesis-overlay: croissant.wBFT.{useNCP,targetValidators,stabilizingStakersThreshold} + govContracts.govNCP.params.ncps", Effect: "makes governance drive the validator set instead of a static one"},
			Knob{Name: "epoch length", Where: "--genesis-overlay: croissant.wBFT.epochLength", Effect: "epoch-transition scenarios"},
		),
	}
}

// governanceSteps is the bring-up for a chain whose genesis carries no validator
// set: the nodes start idle and only produce once governance is deployed and the
// embedded etcd cluster forms.
//
// The order is the procedure's, not a tidier one. Placement comes before the
// genesis because a governance member carries the address and port it will be
// reachable at, so a genesis built first would name places nothing was put. And
// the producer launches alone, because the cluster forms from a node that is by
// itself; the rest arrive afterwards and join it.
//
// The four action steps are named by the consensus family, not here — see
// TestGovernanceSteps_MatchTheFamilysActions.
func governanceSteps() []CaseStep {
	return []CaseStep{
		{ID: "resolve-chain", Title: "Resolve the chain plugin", Detail: "registry.Get: wemix manifest (bootstrap.type governance-etcd)", Implemented: true},
		{ID: "resolve-binary", Title: "Resolve the node binary", Detail: "go-wemix gwemix (embeds etcd; no separate process)", Implemented: true},
		{ID: "load-preset", Title: "Load the key preset", Detail: "producer identities + unlockable accounts", Implemented: true},
		{ID: "allocate", Title: "Allocate hosts and ports", Detail: "netmap.Assign against the family's reservation: etcd peer is p2p+1, client p2p+2", Implemented: true},
		{ID: "wemix-genesis", Title: "Generate the genesis from a governance config", Detail: "poa.Config (every producer a member, addresses from the placement) then gwemix wemix genesis", Implemented: true},
		{ID: "assemble-plan", Title: "Assemble the launch plan", Detail: "engine.AssemblePlan: per-node datadir, config path, launch args", Implemented: true},
		{ID: "arm", Title: "Arm each node", Detail: "render config, install identity flags, resolve the binary per node", Implemented: true},
		{ID: "provision", Title: "Materialize files", Detail: "genesis + per-node config + the governance config deploy-governance reads back", Implemented: true},
		{ID: "launch-boot", Title: "Launch the producer alone", Detail: "the cluster only forms while no other node is up", Implemented: true},
		{ID: "deploy-governance", Title: "Deploy the governance contracts", Detail: "over the producer's IPC, once it is sealing", Implemented: true},
		{ID: "etcd-init", Title: "Initialize the etcd cluster", Detail: "admin.etcdInit() over IPC, with the producer still alone", Implemented: true},
		{ID: "verify-etcd", Title: "Verify the cluster formed", Detail: "admin.wemixInfo.etcd.cluster must be non-empty — the only evidence etcd-init worked", Implemented: true},
		{ID: "launch-rest", Title: "Launch the remaining nodes", Detail: "the rest of the network, against files already on the target", Implemented: true},
		{ID: "etcd-join", Title: "Join the remaining producers", Detail: "each asks the boot node for the cluster; a producer outside it takes no turn at sealing", Implemented: true},
		{ID: "health-gate", Title: "Gate on health", Detail: "poll until the head advances", Implemented: true},
	}
}

func wemixCase() Case {
	return Case{
		ID: "wemix", Title: "gwemix only",
		Bootstrap: "governance-etcd", Support: Supported,
		Binaries: []string{"gwemix from go-wemix (make gwemix USE_ROCKSDB=NO)", "go-wemix wemix/scripts/genesis-template.json"},
		Doc:      "docs/dev/chain-setup/case-1-wemix.md",
		Steps:    governanceSteps(),
		Knobs: []Knob{
			{Name: "governance env", Where: "poa.Env", Effect: "ballot durations, staking bounds, block reward, gas policy, reward distribution"},
			{Name: "members", Where: "poa.Member", Effect: "producer address, stake, devp2p id, ip/port, bootnode flag"},
			{Name: "role accounts", Where: "poa.Config", Effect: "staker, ecosystem, maintenance, feeCollector"},
			{Name: "alloc", Where: "poa.Account[]", Effect: "initial balances; the producer must be funded and unlockable"},
			{Name: "genesis template", Where: "--template", Effect: "must be go-wemix's own template, not chainbench's substitution template"},
			{Name: "etcd join", Where: "the rest phase's etcd-join action", Effect: "each remaining producer asks the boot node for the cluster; without it one node seals every block"},
			{Name: "bootstrap isolation", Where: "phase A of the sequence", Effect: "the producer must be the only node running while governance is deployed and etcd is initialized"},
		},
	}
}

// handoffSteps is the type-1 upgrade: producer and successor run different
// binaries from the start, concurrently, on one chain.
func handoffSteps() []CaseStep {
	return []CaseStep{
		{ID: "load-profile", Title: "Load the upgrade profile", Detail: "roles, fork, network id, ports, governance env, validator set", Implemented: true},
		{ID: "load-preset", Title: "Load the key preset", Detail: "plan_order maps plan position to preset node", Implemented: true},
		{ID: "wemix-config", Title: "Assemble the governance config", Detail: "producer member + governance env + alloc", Implemented: true},
		{ID: "base-genesis", Title: "Generate the producer's base genesis", Detail: "gwemix wemix genesis, then force chainId and petersburgBlock", Implemented: true},
		{ID: "build-plan", Title: "Compose the handoff plan", Detail: "lift the fork section out of the successor's own genesis, merge it plus the fork block, preflight", Implemented: true},
		{ID: "genesis-overlay", Title: "Apply the genesis overlay", Detail: "optional deep merge, e.g. useNCP and the NCP operator set", Implemented: true},
		{ID: "launch", Title: "Init datadirs and launch both binaries", Detail: "each node inits with its own binary from identical genesis bytes", Implemented: true},
		{ID: "wire-mesh", Title: "Wire the peer mesh", Detail: "admin_addPeer across every pair, after the HTTP servers are ready", Implemented: true},
		{ID: "deploy-governance", Title: "Deploy the governance contracts", Detail: "on the producer, over IPC. NOTE: must run with the producer alone; running it against a fully launched network leaves the etcd cluster empty", Implemented: true},
		{ID: "etcd-init", Title: "Initialize the etcd cluster", Detail: "admin.etcdInit() over IPC. Its return value is null even on success, so it proves nothing by itself", Implemented: true},
		{ID: "verify-etcd", Title: "Verify the cluster formed", Detail: "admin.wemixInfo.etcd.cluster must be non-empty; this is the only evidence etcd-init worked", Implemented: true},
		{ID: "await-fork", Title: "Wait for the handoff", Detail: "head passes the fork and a successor validator seals the first post-fork block", Implemented: true},
	}
}

func handoffCase() Case {
	return Case{
		ID: "wemix-wbft", Title: "gwemix -> gwbft hardfork handoff",
		Bootstrap: "governance-etcd", Support: Partial,
		Note: "the procedure is verified end to end by hand (block 100 handed over), but this automation still runs the old " +
			"single-phase order and therefore fails: the bootstrap has to happen with the producer alone, the successor " +
			"validators need their keystore and --unlock, and the mesh has to be re-wired after the final restart",
		Binaries: []string{"gwemix from go-wemix (producer)", "gwemix from go-wbft (validators)", "go-wemix wemix/scripts/genesis-template.json"},
		Doc:      "docs/dev/chain-setup/case-2-wemix-to-wbft.md",
		Steps:    handoffSteps(),
		Knobs: []Knob{
			{Name: "chain pair", Where: "profile upgrade.from / upgrade.to", Effect: "must match the from-chain manifest's declared upgrade target"},
			{Name: "fork", Where: "profile upgrade.at_fork / fork_block", Effect: "which fork hands over, and at what height"},
			{Name: "network id", Where: "profile upgrade.network_id", Effect: "must be uniform: go-wemix otherwise picks its own, independent of chain id"},
			{Name: "roles", Where: "profile roles.producers / roles.validators", Effect: "validators must be at least 4 for BFT progress"},
			{Name: "identity mapping", Where: "profile identities.plan_order", Effect: "plan position to preset node; position 0 is the producer"},
			{Name: "producer accounts", Where: "profile producers.members", Effect: "must be disjoint from the validator set, or the producer stalls"},
			{Name: "governance env", Where: "profile producers.governance", Effect: "the environment the base genesis is generated from"},
			{Name: "validator set", Where: "profile validators.{addresses,bls_public_keys,extra_data}", Effect: "the successor's post-fork validators"},
			{Name: "genesis overlay", Where: "--genesis-overlay", Effect: "useNCP, targetValidators, stabilizingStakersThreshold, govNCP ncps"},
			{Name: "ports", Where: "profile ports.*", Effect: "p2p and rpc bases and steps"},
			{Name: "validator identity", Where: "keystore in the datadir + --unlock/--miner.etherbase", Effect: "without it the successors cannot seal, and the chain stops at the fork block"},
			{Name: "mesh timing", Where: "after the final launch", Effect: "validators that cannot reach each other never reach quorum"},
		},
	}
}
