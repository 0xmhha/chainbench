package chainsetup

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The request a composition is made from, and what it leaves behind.
//
// It sits below the verb that takes it because the record holds it: a
// workspace persists the request it was composed from, which is what a resume
// composes from and what the comparison reads to decide whether a second run
// wants the same chain. A type the record embeds cannot live above the record.

// UpStage is how far NetUp takes the composition.
type UpStage string

const (
	// UpDeploy composes and writes the artifacts (genesis, configs, argv) but
	// starts nothing, so an external launcher can boot from the result. It ends
	// after the deploy step (the old name was "provision").
	UpDeploy UpStage = "deploy"
	// UpStart additionally initializes the datadirs and launches the nodes.
	UpStart UpStage = "start"
)

// NetUpIn describes the network to compose. It is the union of the step inputs,
// in the order the steps consume them.
type NetUpIn struct {
	// DataDir is the workspace directory.
	DataDir string `json:"dataDir,omitempty"`
	// Stage is how far to go; empty means UpStart.
	Stage UpStage `json:"stage,omitempty"`

	// Chain identity (step: new).
	Chain        string        `json:"chain,omitempty"`
	ManifestPath string        `json:"manifestPath,omitempty"`
	TemplatePath string        `json:"templatePath,omitempty"`
	KeysDir      string        `json:"keysDir,omitempty"`
	Target       resource.Spec `json:"target,omitempty"`
	// Binary is the node executable. Required for UpStart; for a remote target
	// it is a path on that host.
	Binary string `json:"binary,omitempty"`

	// Layout (step: allocate).
	BPCount          int    `json:"bp,omitempty"`
	ENCount          int    `json:"en,omitempty"`
	PNCount          int    `json:"pn,omitempty"`
	EndpointSyncMode string `json:"endpointSyncMode,omitempty"`
	TopologyPath     string `json:"topologyPath,omitempty"`
	// AutoSize fills the validator count to the server set (bp: "max"): one node
	// per server, less the proxies and endpoints. It needs a server-set target.
	AutoSize bool `json:"autoSize,omitempty"`
	// BlueprintPath is a network declaration: the layout AND the keys in one
	// document. It is what lets a network be composed with no preset directory
	// anywhere (N3).
	BlueprintPath string `json:"blueprintPath,omitempty"`
	// Topology, when set, is the inline per-node layout, the DSL's in-memory
	// equivalent of TopologyPath. It wins over Validators/Endpoints.
	Topology *node.Topology `json:"topology,omitempty"`
	// Binaries maps each per-node binary name the topology references to its
	// resolved path. Empty means every node runs Binary.
	Binaries map[string]string `json:"binaries,omitempty"`
	// Peering is the peer graph to wire ("mesh" default, "proxied").
	Peering string `json:"peering,omitempty"`
	// Server selects where the nodes run and on what ports, from the server
	// server set. Its zero value uses the built-in local plan.
	Server resource.ServerRef `json:"server,omitempty"`
	// Docker treats the servers as local docker containers (dials translated
	// through the localmap next to the server set); recorded at the new step.
	Docker bool `json:"docker,omitempty"`
	// WorkspaceConfigPath is the environment file owning the target data root and
	// purpose directories; recorded at new so later steps resolve portable
	// references under it.
	WorkspaceConfigPath string `json:"workspaceConfigPath,omitempty"`

	// Identities (step: keys).
	KeysSource string `json:"keysSource,omitempty"`
	// KeysValidators is how many of a generated key set join the validator set
	// (0 = all). It has effect only when KeysSource is "generate".
	KeysValidators int `json:"keysValidators,omitempty"`

	// Genesis customization (step: genesis).
	ChainID     int64    `json:"chainID,omitempty"`
	GenesisSet  []string `json:"genesisSet,omitempty"`
	OverlayPath string   `json:"overlayPath,omitempty"`
	// GenesisFork, when set, schedules a hardfork whose consensus configuration
	// comes from the chain that seals after it.
	GenesisFork *GenesisFork `json:"genesisFork,omitempty"`
	// BinaryChains names, per binary, the chain that binary runs when it is not
	// the composition's. It travels beside Binaries because it answers the same
	// question about the same name.
	BinaryChains map[string]string `json:"binaryChains,omitempty"`
	// GenesisPerBinary names, per binary, an overlay file whose genesis
	// fragment is merged onto the built genesis to make that binary's own
	// document. The nodes running it initialize from the result.
	GenesisPerBinary map[string]string `json:"genesisPerBinary,omitempty"`
	// GenesisExisting is a reference to a finished genesis file used verbatim
	// (genesis mode "existing"); empty builds from the template as usual.
	GenesisExisting string `json:"genesisExisting,omitempty"`

	// LaunchSet are launch-argv overrides applied to every node (step:
	// launchopts) — the "all" scope.
	LaunchSet []string `json:"launchSet,omitempty"`
	// LaunchScoped are launch-argv overrides by scope ("all", a role like "bp"/
	// "en", or "node<N>"), applied per node most-general-first. It is the DSL
	// env.launch form; the CLI passes LaunchSet.
	LaunchScoped map[string][]string `json:"launchScoped,omitempty"`

	// ConfigSet are per-scope config-knob overrides (step: config), keyed by
	// scope ("all" / "node<N>"). Each value is a list of dot-path "key=value".
	ConfigSet map[string][]string `json:"configSet,omitempty"`
}

// UpStepNames is the composition order, for reading the record.
//
// It no longer drives anything: the walk is the transition table, and the stage
// table in statedriven.go is what names each step's state. What is left needs
// the names in order — resume asks the record which step is the first one not
// marked done — and a test holds the two lists to the same nine names in the
// same order.
var UpStepNames = []string{"new", "place", "keys", "genesis", "config", "build", "deploy", "init", "start"}
