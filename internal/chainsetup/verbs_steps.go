package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The composition verbs, as a surface calls them: keys, allocate, genesis,
// config, launch options.
//
// These carry more than a call because a declaration reaches them here — a
// blueprint to read, an overlay to stage, a peering to resolve. The lifecycle
// verbs, which do not, are in verbs_lifecycle.go.

func withWorkspace(d Deps, dataDir string, fn func(*Workspace) (string, error)) (string, error) {
	return inWorkspace(d, dataDir, fn)
}

// inWorkspace is withWorkspace for a step that reports more than a line.
//
// A step that decides something — which of three key sources it used, which of
// two targets it shipped to — has to be able to say so, and a string is what it
// says to a person rather than to the caller. The lock, the save and the way
// the two errors are joined are the same for both, so they are written once
// here and withWorkspace is the string case of it.
func inWorkspace[T any](d Deps, dataDir string, fn func(*Workspace) (T, error)) (T, error) {
	var zero T
	ws, err := Open(dataDir, d.Clock)
	if err != nil {
		return zero, err
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)

	// One run at a time per workspace. A second run would compose over the
	// first's half-built network and blame the collision on the chain; the
	// refusal names who holds it instead. A lock left by a run that died is
	// taken over — that run is gone, and its wreckage is what the operator is
	// here to clear — but never in silence.
	held, prev, state, lerr := ws.Acquire(d.command())
	if lerr != nil {
		return zero, lerr
	}
	defer func() { _ = held.Release() }()
	if state == session.LockStale {
		d.logf("took over a lock left by a run that is no longer running (%s) — nodes it started may still be up; `chain status` shows what is there", prev.Describe())
	}

	out, stepErr := fn(ws)
	saveErr := ws.Save()
	switch {
	case stepErr != nil && saveErr != nil:
		return zero, fmt.Errorf("%w (and the workspace could not be saved: %v — processes this step started may not be recorded)", stepErr, saveErr)
	case stepErr != nil:
		return zero, stepErr
	case saveErr != nil:
		return zero, saveErr
	}
	return out, nil
}

// Placement bounds shared by the composition steps: a BFT network needs at
// least one validator, and a local port band holds this many nodes.
const (
	minValidators = 1
	portBand      = 100
)

// StepOut is the common result of one mutating step: its recorded detail line,
// and the state it ended in when the step knows one.
type StepOut struct {
	Detail string
	// Reached is the lifecycle state this step ended in. It is zero for a step
	// whose work has not moved into its handler yet, and the handler then
	// reports what the request asked for instead of what the step did. A step
	// that sets it has stopped being guessed at.
	Reached lifecycle.Status
}

// NetKeysIn selects where node identities come from.
type NetKeysIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	// Source is preset (default) | generate | declared.
	Source string `cb:"keys-source" default:"keyPreset" help:"keyPreset (use the recorded key set) | generate (create a fresh set)"`
	// BlueprintPath is the declaration the keys come from when Source is
	// "declared" (or when a blueprint is given and Source is silent).
	BlueprintPath string
	Nodes         int `cb:"nodes" help:"identities the set must cover (default: the allocated node count)"`
	Validators    int `cb:"validators" help:"identities joining the validator set (generate; 0 = all)"`
}

// NetKeys ensures the workspace's key set exists and covers the node count.
func NetKeys(ctx context.Context, d Deps, in NetKeysIn) (StepOut, error) {
	bp, err := readBlueprint(in.BlueprintPath)
	if err != nil {
		return StepOut{}, err
	}
	source := in.Source
	// A blueprint that carries keys is the source unless the caller asked for
	// another one. Making the operator name it twice would let the two answers
	// disagree, and the composition would take the one they did not mean.
	if source == "" && bp != nil {
		source = "declared"
	}
	done, err := inWorkspace(d, in.DataDir, func(ws *Workspace) (KeysDone, error) {
		return ws.Keys(ctx, KeysOpts{Source: source, Blueprint: bp, Nodes: in.Nodes, Validators: in.Validators})
	})
	return StepOut{Detail: done.Detail, Reached: done.Source}, err
}

// NetAllocateIn sizes the network.
type NetAllocateIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	BPCount int    `cb:"bp" default:"4" help:"bp (block-producing) node count"`
	ENCount int    `cb:"en" help:"en (endpoint, non-producing) node count"`
	PNCount int    `cb:"pn" help:"pn (proxy-tier) node count; a family with no proxy tier refuses it"`
	// Peering is the peer graph ("mesh" default, "proxied").
	Peering string `cb:"peering" help:"peer graph: mesh (default, every node dials every other) | proxied (bp <-> pn <-> en; endpoints never dial a producer)"`
	// EndpointSyncMode switches endpoints off full sync ("snap"/"archive") so a
	// re-sync test can exercise that path. Empty leaves every node on full.
	EndpointSyncMode string `cb:"endpoint-syncmode" help:"sync mode for endpoints (snap|archive); default full"`
	// TopologyPath is a per-node layout YAML (role, sync mode, bootnode). It
	// replaces the counts, which cannot express a per-node choice.
	TopologyPath string `cb:"topology" help:"per-node layout YAML (role/sync-mode/bootnode/binary); overrides --bp/--en/--pn"`
	// BlueprintPath is a network declaration (N1). It is the widest of the
	// three layout sources and wins over both the counts and a topology.
	BlueprintPath string
	// Server selects where the nodes are placed and on what ports, from the
	// operator's server set. Its zero value uses the built-in local plan.
	Server resource.ServerRef
	// Topology, when set, is the inline per-node layout (role/sync/bootnode/
	// binary), the DSL's way to declare what --topology gives a file. It wins
	// over Validators/Endpoints.
	Topology *node.Topology
	// Binaries maps per-node binary names to paths (with a per-node topology).
	Binaries map[string]string
	// BinaryChains names, per binary, the chain that binary runs when it is not
	// the composition's.
	BinaryChains map[string]string
	// AutoSize fills the validator count to the server set (bp: "max"): the
	// network is one node per server, less the proxies and endpoints asked for.
	// It needs a server-set target and is ignored when a topology or blueprint
	// gives the layout explicitly.
	AutoSize bool
}

// NetAllocate builds the node table (roles, paths, deterministic ports).
func NetAllocate(_ context.Context, d Deps, in NetAllocateIn) (StepOut, error) {
	bp, err := readBlueprint(in.BlueprintPath)
	if err != nil {
		return StepOut{}, err
	}
	topo := in.Topology
	if bp != nil && (topo != nil || in.TopologyPath != "") {
		// Both describe the layout, and picking one silently would leave the
		// other's author reading a network that is not theirs.
		return StepOut{}, fmt.Errorf("chainsetup: allocate: a blueprint and a topology both describe the layout — give one")
	}
	if topo == nil && in.TopologyPath != "" {
		loaded, err := node.Load(in.TopologyPath)
		if err != nil {
			return StepOut{}, err
		}
		topo = &loaded
	}
	// Allocation is the one moment two runs can hand out the same slot: each
	// derives the set's inventory from the workspaces it can see, and two
	// runs that look before either has saved both see it free. The set's
	// lock is held from the look to the save — a lock, not a second record
	// (the workspaces stay the only record of what is taken).
	setPath := in.Server.SetPath
	if setPath == "" {
		if ws, err := Open(in.DataDir, d.Clock); err == nil {
			setPath = ws.State().ServerSet
		}
	}
	release, err := acquireSetLock(setPath, d)
	if err != nil {
		return StepOut{}, err
	}
	defer release()
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		// A set the workspace already recorded (chain new --server-set) is the
		// default: --docker and its set arrive as a pair, and a later
		// --server-set on this step still wins.
		if in.Server.SetPath == "" {
			in.Server.SetPath = ws.State().ServerSet
		}
		resolved, err := resource.ResolveServer(in.Server, minValidators, portBand)
		if err != nil {
			return "", err
		}
		if resolved.HasTarget {
			if err := ws.Retarget(resolved.Target); err != nil {
				return "", err
			}
		}
		return ws.Allocate(AllocateOpts{
			BPCount: in.BPCount, ENCount: in.ENCount, PNCount: in.PNCount,
			EndpointSyncMode: in.EndpointSyncMode, Topology: topo, Blueprint: bp,
			Peering: peeringOf(bp, in.Peering),
			Pool:    resolved.Pool, SetPath: in.Server.SetPath, Binaries: in.Binaries,
			BinaryChains: in.BinaryChains,
			AutoSize:     in.AutoSize,
		})
	})
	return StepOut{Detail: detail}, err
}

// readBlueprint loads a network declaration, or returns nil when none is named.
func readBlueprint(path string) (*blueprint.Blueprint, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: read blueprint %s: %w", path, err)
	}
	bp, err := blueprint.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &bp, nil
}

// peeringOf lets a flag override the declaration, and the declaration answer
// when the flag is silent. A flag is a person typing now, which is the one
// thing that outranks a document.
func peeringOf(bp *blueprint.Blueprint, flag string) string {
	if flag != "" || bp == nil {
		return flag
	}
	return bp.Peering
}

// NetGenesisIn customizes the built genesis.
//
// The cb tags are what a surface renders this from: one declaration behind the
// cobra flags a person types and the JSON schema an agent reads, so the two
// cannot describe the same argument differently (feature.Flags / feature.Schema,
// surface-unification-design §3.2). They are inert until a surface reads them —
// this struct is unchanged otherwise — and TestNetGenesis_TagsMatchTheCommand
// holds them to the flags the command declares by hand today, so the derivation
// is proven to reproduce the shipped surface before anything switches to it.
type NetGenesisIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	ChainID int64  `cb:"chain-id"               help:"override the manifest chain id (0 = manifest)"`
	// Set carries genesis config overrides as key=value on the bare config key,
	// e.g. "bohoBlock=10" to move a fork off genesis.
	Set []string `cb:"set" help:"override a genesis config key (repeatable), e.g. --set bohoBlock=10"`
	// OverlayPath is a JSON overlay file {capabilities, genesis}: the genesis
	// fragment is deep-merged and the capabilities are advertised.
	OverlayPath string `cb:"overlay" help:"JSON overlay file {capabilities,genesis} deep-merged into the genesis"`
	// GenesisExisting is a reference to a finished genesis file used verbatim
	// (genesis mode "existing"); empty builds from the template.
	GenesisExisting string
	// Fork, when set, schedules a hardfork whose consensus configuration comes
	// from the chain that seals after it.
	Fork *GenesisFork
	// PerBinary names, per binary, an overlay file in the same shape as
	// OverlayPath. Its genesis fragment is merged onto the built genesis to
	// make that binary's own document.
	PerBinary map[string]string
}

// NetGenesis builds the genesis from the key set and writes it to the target.
func NetGenesis(ctx context.Context, d Deps, in NetGenesisIn) (StepOut, error) {
	// The request is read before the workspace is opened: one that contradicts
	// itself needs no workspace, and refusing here keeps the lock and the state
	// out of a request that was never going to be carried out.
	opts, err := genesisOpts(in)
	if err != nil {
		return StepOut{}, err
	}
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.Genesis(ctx, opts)
	})
	return StepOut{Detail: detail}, err
}

// genesisOpts folds the flag-shaped genesis inputs into the step options: the
// key=value overrides and the overlay file's two halves.
// genesisOpts turns a genesis request into the options the step applies, and
// refuses a request that asks for both a finished genesis and a change to it.
//
// The two cannot both be honoured: a finished genesis is written byte for byte,
// so a chain id, a fork height or an overlay arriving alongside it is silently
// dropped. It used to be worse than silent — the completion detail still
// reported "chain id N (override)" and the advertised capabilities were derived
// from the dropped fork heights, so a capability-gated fork test would run
// against a chain that has no such fork. Saying no here, before anything is
// written, is the only answer that leaves the request and the result equal.
func genesisOpts(in NetGenesisIn) (GenesisOpts, error) {
	opts, err := buildGenesisOpts(in)
	if err != nil {
		return opts, err
	}
	return opts, opts.checkExistingIsUnchanged()
}

// checkExistingIsUnchanged reports a request that pairs a finished genesis with
// a change to it, naming every conflicting part so one message covers the whole
// request rather than one round trip per option.
func (o GenesisOpts) checkExistingIsUnchanged() error {
	if o.Existing == "" {
		return nil
	}
	var asked []string
	if o.ChainID != 0 {
		asked = append(asked, fmt.Sprintf("chain id %d", o.ChainID))
	}
	if len(o.Overrides) > 0 {
		asked = append(asked, fmt.Sprintf("genesis override(s) %s", strings.Join(slices.Sorted(maps.Keys(o.Overrides)), ", ")))
	}
	if len(o.Overlay) > 0 {
		asked = append(asked, "a genesis overlay")
	}
	if len(o.Variants) > 0 {
		asked = append(asked, fmt.Sprintf("a separate genesis for binary %s", strings.Join(slices.Sorted(maps.Keys(o.Variants)), ", ")))
	}
	if len(asked) == 0 {
		return nil
	}
	return fmt.Errorf(
		"chainsetup: genesis: %q is used verbatim, so %s cannot be applied — drop the change, or build the genesis instead of naming a finished one",
		o.Existing, strings.Join(asked, " and "))
}

func buildGenesisOpts(in NetGenesisIn) (GenesisOpts, error) {
	opts := GenesisOpts{ChainID: in.ChainID, Existing: in.GenesisExisting, Fork: in.Fork}
	for _, kv := range in.Set {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return opts, fmt.Errorf("chainsetup: genesis override expects key=value, got %q", kv)
		}
		if opts.Overrides == nil {
			opts.Overrides = map[string]string{}
		}
		opts.Overrides[k] = v
	}
	for _, name := range slices.Sorted(maps.Keys(in.PerBinary)) {
		overlay, err := readGenesisOverlay(in.PerBinary[name])
		if err != nil {
			return opts, err
		}
		// Capabilities describe the network, and a network advertises one set.
		// Accepting them here would let two binaries claim different ones with
		// no way to say which the network has.
		if len(overlay.Capabilities) > 0 {
			return opts, fmt.Errorf("chainsetup: genesis: the overlay for binary %q declares capabilities, which describe the whole network — declare them on the network's own overlay", name)
		}
		// Same reason: whether the chain stops is one answer for the network.
		if overlay.HaltsAt != 0 {
			return opts, fmt.Errorf("chainsetup: genesis: the overlay for binary %q declares haltsAt, which describes the whole network — declare it on the network's own overlay", name)
		}
		if opts.Variants == nil {
			opts.Variants = map[string][]byte{}
		}
		opts.Variants[name] = overlay.Genesis
	}
	if in.OverlayPath == "" {
		return opts, nil
	}
	overlay, err := readGenesisOverlay(in.OverlayPath)
	if err != nil {
		return opts, err
	}
	opts.Overlay = overlay.Genesis
	opts.Capabilities = overlay.Capabilities
	opts.HaltsAt = overlay.HaltsAt
	return opts, nil
}

// genesisOverlayFile is the {capabilities, genesis} document an overlay path
// holds. One reader for the network's overlay and for a binary's own, so the
// two cannot come to disagree about the shape.
type genesisOverlayFile struct {
	Capabilities []string        `json:"capabilities"`
	Genesis      json.RawMessage `json:"genesis"`
	// HaltsAt is the block this genesis makes the network stop one short of.
	HaltsAt int64 `json:"haltsAt,omitempty"`
}

func readGenesisOverlay(path string) (genesisOverlayFile, error) {
	var overlay genesisOverlayFile
	raw, err := os.ReadFile(path)
	if err != nil {
		return overlay, err
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return overlay, fmt.Errorf("chainsetup: bad genesis overlay %q: %w", path, err)
	}
	return overlay, nil
}

// NetConfigIn identifies the workspace and, optionally, per-node config
// overrides to record before rendering.
type NetConfigIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	// Node scopes a Set override to that 1-based node; 0 means every node.
	Node int `cb:"node" help:"scope --set to this 1-based node (default: every node)"`
	// Set are dot-path "key=value" config-knob overrides to record. Empty
	// renders with whatever overrides the workspace already holds.
	Set []string `cb:"set" help:"config knob override key=value (repeatable; supported keys: syncMode, httpHost, metricsHost)"`
	// ScopedSet records overrides for several scopes at once ("all", a role,
	// "node<N>"), the form the up flow and a DSL env pass. It is applied before
	// Set, most general scope first.
	ScopedSet map[string][]string
}

// NetConfig records any per-node overrides, then renders and writes each node's
// TOML config with them applied. Recording and rendering share one step so
// `chain config --node N --set k=v` both persists the override and reflects it.
func NetConfig(ctx context.Context, d Deps, in NetConfigIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		for _, scope := range sortedScopes(in.ScopedSet) {
			if err := ws.recordConfigSet(scope, in.ScopedSet[scope]); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		if len(in.Set) > 0 {
			scope := node.ScopeAll
			if in.Node > 0 {
				scope = fmt.Sprintf("node%d", in.Node)
			}
			if err := ws.recordConfigSet(scope, in.Set); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		return ws.Config(ctx)
	})
	return StepOut{Detail: detail}, err
}

// NetLaunchOptsIn customizes the assembled argv.
type NetLaunchOptsIn struct {
	DataDir string   `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	Set     []string `cb:"set" help:"high-precedence launch knob key=value (repeatable; bare key for booleans)"` // key=value overrides for every node (bare key for booleans)
	// ScopedSet records overrides for several scopes at once ("all", a role like
	// "bp"/"en", or "node<N>"), the form the up flow and a DSL env pass. Set is
	// folded into the "all" scope.
	ScopedSet map[string][]string
}

// NetLaunchOptsOut is the assembled per-node argv table.
type NetLaunchOptsOut struct {
	Detail string
	Nodes  []node.Record
}

// NetLaunchOpts assembles each node's launch argv (the single assembly site)
// and records it, returning the table so the surface can render the commands.
func NetLaunchOpts(_ context.Context, d Deps, in NetLaunchOptsIn) (NetLaunchOptsOut, error) {
	var nodes []node.Record
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		for _, scope := range sortedScopes(in.ScopedSet) {
			if err := ws.recordLaunchSet(scope, in.ScopedSet[scope]); err != nil {
				return "", fmt.Errorf("chainsetup: launchopts: %w", err)
			}
		}
		if err := ws.recordLaunchSet("all", in.Set); err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: %w", err)
		}
		det, err := ws.LaunchOpts()
		nodes = ws.State().Nodes
		return det, err
	})
	return NetLaunchOptsOut{Detail: detail, Nodes: nodes}, err
}

// NetProvisionIn identifies the workspace.
