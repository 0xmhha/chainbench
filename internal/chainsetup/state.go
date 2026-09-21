package chainsetup

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// What a workspace records: the composition state, and the format version that
// says how to read it.
//
// The state is the composition's single account of itself — which steps ran,
// what they produced, where each node is. A later step, a resume, or another
// process reads this rather than inferring from files on disk, because a file
// can be present for reasons the composition never chose.

const StateFormatVersion = 1

// Step is a completed composition step (persistence model owned by session).
type Step = session.Step

// State is the persisted composition state accumulated across step commands. It
// grows as later steps land (placements, genesis path, node table); each field
// is optional so a partially-composed workspace round-trips. It holds no
// secrets — a server-set placement reads its login from the server-set file at
// resolve time, and a directly named target reads the environment.
type State struct {
	// FormatVersion is the shape this record was written in. A build reads only
	// the version it writes: this track removes old-format readers rather than
	// keeping one per era, so a record from another version is refused by name
	// instead of being half-understood.
	//
	// It is first because it decides whether the rest means anything.
	FormatVersion int `json:"formatVersion"`

	Chain string `json:"chain"`
	// CompositionID is a stable identifier for this composition, set once at
	// `new` and kept across resume and binary swap. It names the composition's
	// own node data directories, runtime files, and logs so two compositions on
	// one data root do not collide; it is derived from the workspace directory,
	// not the run time or a pid, so a resume keeps the same one.
	CompositionID string `json:"compositionId,omitempty"`
	// ManifestPath and TemplatePath name an external, project-supplied chain
	// manifest. When set they win over Chain, so a workspace composed for a
	// project's own chain resolves the same plugin on every later step.
	ManifestPath string `json:"manifestPath,omitempty"`
	TemplatePath string `json:"templatePath,omitempty"`
	Binary       string `json:"binary,omitempty"`
	KeysDir      string `json:"keysDir,omitempty"`
	// BPCount is how many bp nodes the placement resolved to. The genesis step
	// sizes the validator set from it: the producers ARE the validator set, and
	// counting the placements rather than the request is what makes a topology
	// decide it.
	BPCount     int           `json:"bp,omitempty"`
	Target      resource.Spec `json:"target"`
	GenesisPath string        `json:"genesisPath,omitempty"`
	// BinaryChains maps a binary name — the same names Binaries uses — to the
	// chain that binary runs. Absent, or a name it does not hold, means the
	// composition's own chain.
	//
	// It exists because the chain is what answers a node's launch: which flag
	// vocabulary its binary accepts, which RPC namespace it serves, and what its
	// consensus asks of it. A network running two builds of two different chains
	// had every node answering with one of them.
	BinaryChains map[string]string `json:"binaryChains,omitempty"`
	// GenesisPaths maps a binary name — the same names Binaries uses — to the
	// genesis the nodes running that binary initialize from. Absent, or a name
	// it does not hold, means GenesisPath: one document for the network, which
	// is what every composition of a single binary wants and what this was
	// before.
	//
	// It exists because a network can run two binaries that do not accept the
	// same genesis. A handoff across a fork is the case: the successor needs
	// fork settings in its genesis, and whether the predecessor tolerates them
	// is a fact about those two builds, not something a composer can assume.
	// Keyed by binary name rather than by node because that is the grouping the
	// declaration already makes — a node says which binary it runs, and its
	// genesis follows from that.
	GenesisPaths map[string]string `json:"genesisPaths,omitempty"`
	// GenesisConfigPaths maps a binary name to the genesis its nodes read from
	// their config file rather than from a genesis document. It is the other
	// half of GenesisPaths: a fork carried by config leaves one genesis for the
	// whole network and puts what the post-fork build needs here.
	GenesisConfigPaths map[string]string `json:"genesisConfigPaths,omitempty"`
	// Fork is the hardfork this network is composed to cross, recorded by the
	// genesis step. Nil for a network that crosses none.
	Fork *GenesisFork `json:"fork,omitempty"`
	// LaunchInputs is what each launch input hashed to when this workspace
	// wrote it, keyed by path on the target.
	//
	// It exists so deploy can tell "the file is there" from "the file is the
	// one we built". Without it deploy checked only for presence and reported
	// the inputs "reused, not rewritten", which is true of a genesis someone
	// edited and of a config left by a previous composition — and the nodes
	// then launch from it. A hash costs nothing to record and turns a silent
	// wrong launch into a refusal that names the file.
	LaunchInputs map[string]string `json:"launchInputs,omitempty"`
	Nodes        []node.Record     `json:"nodes,omitempty"`
	// Steps is what has been done to this composition, by name. It holds two
	// kinds: a rung of the composition ladder (UpStepNames) and an operation on
	// a network already up (OpStepNames). Every reader names the subset it
	// means, so the two do not collide — but which kind a name is has to be
	// read off one of those two lists, not guessed from the map.
	Steps map[string]Step `json:"steps"`
	// Peering is the peer graph the composition wires ("mesh" default,
	// "proxied" for bp <-> pn <-> en). Empty means mesh, so a workspace written
	// before the field keeps the graph it was composed with.
	Peering string `json:"peering,omitempty"`
	// Bootnode is the 1-based index of the topology's bootnode, or 0 when the
	// layout came from plain counts. Informational: every composed node lists
	// every other as a static node, so peering does not depend on it.
	Bootnode int `json:"bootnode,omitempty"`
	// PortOrigin names where the port plan came from (a server set entry,
	// or the built-in defaults), so an operator reading the state never has to
	// guess why a node listens where it does.
	PortOrigin string `json:"portOrigin,omitempty"`
	// ServerSet is the server-set file the placement came from, recorded so
	// later steps resolve the same file — and, in docker mode, find the
	// localmap next to it.
	ServerSet string `json:"serverSet,omitempty"`
	// WorkspaceConfig is the environment file (--workspace-config) this
	// composition was set up with, recorded so later steps and a resume resolve
	// portable file references under the same data root and purpose directories.
	WorkspaceConfig string `json:"workspaceConfig,omitempty"`
	// Docker records that this composition treats its servers as local docker
	// containers: the harness's own dials are translated through the localmap
	// next to ServerSet. It is recorded once at `chain new --docker` so a
	// multi-step run cannot be half-mapped, and it never changes what is
	// composed — genesis, static-nodes and the node table keep real addresses.
	Docker bool `json:"docker,omitempty"`
	// Capabilities is what the composed network advertises to capability-gated
	// test cases (chain manifest + ws + delayed-fork markers + overlay claims).
	// The genesis step derives it, since that is where the customizations that
	// change what the network can do are applied.
	Capabilities []string `json:"capabilities,omitempty"`
	// HaltsAt is the block this network's genesis makes it stop one short of,
	// and 0 when it keeps producing. Recorded beside the capabilities because
	// the genesis step is what learns it, and read back by the readiness gate:
	// a chain that is meant to stop is not a chain that is failing to start.
	HaltsAt int64 `json:"haltsAt,omitempty"`
	// ConfigSet holds per-scope config-knob overrides, keyed by scope: "all"
	// for every node, "node<N>" for one. Each value is a list of dot-path
	// "key=value" strings applied at config render (all first, then the node's
	// own — node wins). Stored key-agnostically so the surface and the storage
	// do not change when the set of overridable knobs grows.
	ConfigSet map[string][]string `json:"configSet,omitempty"`
	// ConfigProvenance records, per node, the overrides that shaped its config
	// and the checksum of the config that resulted, so a run can show which
	// config each node got and that it read back intact. The config step
	// recomputes it each time it runs — a fresh config is a new revision.
	ConfigProvenance []ConfigProvenance `json:"configProvenance,omitempty"`
	// LaunchSet holds per-scope launch-argv overrides, keyed by scope: "all" for
	// every node, a role ("bp"/"en") for that role, "node<N>" for one.
	// Each value is a list of "key" (boolean flag) or "key=value" applied at
	// argv assembly, most-general-first (all, then role, then node — node wins).
	LaunchSet map[string][]string `json:"launchSet,omitempty"`
	// Binaries maps a per-node binary name (as topology entries reference it)
	// to its resolved path. Empty means every node runs the single Binary. It
	// is how one network runs mixed builds concurrently.
	Binaries map[string]string `json:"binaries,omitempty"`
	// Request is what `chain up` was asked to compose, recorded at the new
	// step so a run that dies before the results exist can be resumed from
	// what it was asked, not re-asked. Its DataDir is left empty: the
	// workspace's location is where this file is. It is the one fact of a
	// composition that is otherwise nowhere on disk (F1).
	Request *NetUpIn `json:"request,omitempty"`
}

// Workspace is an open composition workspace: the session-owned persistence
// (control directory + manifest + step stamps) plus the netcompose domain
// state the steps accumulate.
