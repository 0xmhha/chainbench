package testengine

import (
	"github.com/0xmhha/chainbench/internal/core/lifecycle"

	"github.com/0xmhha/chainbench/internal/core/origin"

	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/preset"
	"github.com/0xmhha/chainbench/internal/resource"
)

// A suite composes the network its specs declare. The declaration is the env
// half of a v2 case (or a v1 spec's chain block); this file turns it into the
// request a composer takes. There are two composers — the workspace steps for a
// single-binary network, and the handoff for a mixed-binary one — and the
// declaration's shape picks between them. Nothing here asks which chain it is.
//
// Reading the topology out of a declaration is in topology.go, and the genesis
// a declaration asks for — overlays and scheduled forks — in genesis_decl.go.

const suiteDefaultValidators = 4

// overlayFilePrefix names the file a declared genesis overlay is written to for
// the genesis step, which reads overlays from a file. The content's digest is
// appended, so two environments declaring different overlays get two files.
//
// One fixed name was wrong twice over. Several specs run in one workspace, so
// the second overlay overwrote the first — and the recorded request of the
// earlier environment then pointed at the later one's bytes, which is the one
// place a composition's own record could not be trusted. And two different
// overlays produced the same path, so preflight's genesis comparison, which can
// only see what the request names, could not tell them apart.
const overlayFilePrefix = "env-genesis-overlay"

// defaultKeysDir is the key set a declaration that names none composes from.
const defaultKeysDir = preset.KeysDir

// keySourceGenerate is the key source that creates a fresh set rather than
// reading a recorded one; keySourceKeyPreset reads the recorded one and is what
// a declaration that names no source gets.
const (
	keySourceGenerate  = "generate"
	keySourceKeyPreset = "keyPreset"
)

// generatedKeysSubdir is where a generated set with no ref lands, under the
// workspace, so generate does not reuse the shared preset by default.
const generatedKeysSubdir = "keys"

// expand substitutes environment variables in a declared path or binary:
// $VAR, ${VAR}, and ${VAR:-default} — the last so a declaration can name the
// binary it expects (gwbft) while a machine that built it elsewhere points at
// the build. An unset variable with no default expands to nothing, and the
// composer reports the empty value where it matters.
func expand(s string) string {
	return os.Expand(s, func(name string) string {
		if i := strings.Index(name, ":-"); i >= 0 {
			if v := os.Getenv(name[:i]); v != "" {
				return v
			}
			return name[i+2:]
		}
		return os.Getenv(name)
	})
}

// refuseMachineConflict stops a run whose declaration and command name
// different machines.
//
// The data root already works this way: WorkspaceConfig.AdoptDataRoot refuses
// two answers rather than picking one, and names the line to change. The
// machine had no such rule, so a case declaring srv://alpha and a command
// passing --server beta composed on beta and said nothing — the plan printed
// "server beta" and credited the command, and alpha was gone. A test that runs
// on the wrong machine does not fail; it answers a question nobody asked.
//
// --docker is not a machine and is not checked here: it says how the server
// set's entries are reached (as local containers), not which entry to use.
func refuseMachineConflict(in RunSuiteIn, declared resource.Spec, placement string) error {
	commanded := in.Server.Name
	switch {
	case in.Server.All:
		commanded = "every server in the set"
	case commanded == "":
		return nil
	}
	// Which machine a spec names is resource's to say, not this one's: it owns
	// the locality rule (architecture-v2 §4). Server first, then a host; neither
	// means the declaration named a path and there is nothing to disagree with.
	named := declared.Server
	if named == "" {
		named = declared.Host
	}
	if named == "" {
		return nil
	}
	if named == commanded {
		return nil
	}
	return fmt.Errorf(
		"testengine: machine conflict: the env target %q says %q but --server says %q — name the machine in one place",
		placement, named, commanded)
}

// countFrom records that the declaration asked for a node count.
//
// A count left at zero records nothing. Zero means the role is absent, and
// nobody chose that — a later branch may still fill it, and that branch says so
// itself. Recording zero as a harness choice put two rows nobody asked about in
// front of every reader.
func countFrom(from map[PlanField]origin.Origin, f PlanField, count int) {
	if count > 0 {
		from[f] = origin.FromDeclaration
	}
}

// composition is what one suite composes: a single-binary network through
// the workspace steps, or a mixed-binary handoff. Exactly one is set.
//
// from records who chose each value that had more than one candidate. It is
// filled here rather than derived later because only this function sees the
// candidates: once the merge is done, a value that came from the command and
// one that came from the document are the same string.
type composition struct {
	up   *chainsetup.NetUpIn
	from map[PlanField]origin.Origin
}

// compositionOf reads the network a spec declares and applies the caller's
// overrides (a binary path, a key set, a validator count, a server) on top.
// A v1 spec declares through its chain block; a v2 case through its env.
func compositionOf(ctx context.Context, spec dsl.Spec, in RunSuiteIn) (composition, error) {
	chain := spec.Chain.Name
	if in.Chain != "" && in.Chain != chain {
		return composition{}, lifecycle.Mark(errContradicted, fmt.Errorf("the request names chain %q but the spec declares %q", in.Chain, chain))
	}
	from := map[PlanField]origin.Origin{}
	keysDir := in.KeysDir
	keysSource := in.KeysSource
	if keysSource != "" {
		from[FieldKeysSource] = origin.FromCommand
	}
	if keysDir != "" {
		from[FieldKeysDir] = origin.FromCommand
	}
	keysValidators := 0
	if k := spec.EnvKeys; k != nil {
		if keysSource == "" && k.Source != "" {
			keysSource = k.Source
			from[FieldKeysSource] = origin.FromDeclaration
		}
		if keysDir == "" && k.Ref != "" {
			keysDir = expand(k.Ref)
			from[FieldKeysDir] = origin.FromDeclaration
		}
		keysValidators = k.Validators
	}
	if keysSource == "" {
		from[FieldKeysSource] = origin.FromDefault
	}
	if keysDir == "" {
		// A generated set — or a node table that pins per-node keys — goes to a
		// workspace-local dir, not the shared preset. The key sources reuse
		// whatever set already sits at the dir, so defaulting to presets/keys
		// would silently reuse the preset's identities (and ignore the pinned
		// keys, or fail when the network wants more than the preset holds)
		// instead of building a fresh set. Preset stays the default otherwise.
		if keysSource == keySourceGenerate || topologyHasKeys(spec.Topology) {
			keysDir = filepath.Join(in.DataDir, generatedKeysSubdir)
		} else {
			keysDir = defaultKeysDir
		}
		from[FieldKeysDir] = origin.FromDefault
	}
	var upgradeFork *chainsetup.GenesisFork
	perBinaryOverlay, err := writeOverlays(ctx, in.DataDir, spec.Chain.GenesisPerBinary)
	if err != nil {
		return composition{}, err
	}
	overlayPath, err := writeOverlay(ctx, in.DataDir, spec.Chain.GenesisOverlay, spec.Chain.GenesisProvides, spec.Chain.GenesisHaltsAt)
	if err != nil {
		return composition{}, err
	}

	// An upgrade env that declares a node table composes like any other network:
	// the nodes say which build each of them runs, and the fork's configuration
	// is scheduled on the one genesis they all initialize from. One without a
	// table still goes to the handoff composer, which sizes the network from its
	// preset's roles.
	// A hardfork is composed like any other network: the nodes say which build
	// each of them runs, and the fork's configuration is scheduled on the one
	// genesis they all initialize from. It used to have a composer of its own —
	// its own plan, its own launcher, its own peer mesh — which is what this
	// track removed.
	if u := spec.EnvUpgrade; u != nil {
		if len(spec.Topology) == 0 {
			return composition{}, lifecycle.Mark(errIncomplete, fmt.Errorf("a hardfork declares which build each node runs, so its env needs a node table (topology.nodes[])"))
		}
		fork, ferr := forkOf(u)
		if ferr != nil {
			return composition{}, ferr
		}
		if err := checkForkIsOneTheChainKnows(u, chain, spec.Chain.BinaryChains); err != nil {
			return composition{}, err
		}
		upgradeFork = fork
	}

	// A node table (topology.nodes[]) declares each node's role and binary
	// explicitly; its absence keeps the count form (validators/endpoints).
	inlineTopo, resolvedBins, topoBinary, err := inlineTopologyOf(chain, spec.Topology, spec.Chain.Binaries)
	if err != nil {
		return composition{}, err
	}
	// Every name the declaration gives, whether or not a node is running it.
	//
	// A declaration that names binaries is not only talking about the nodes it
	// composes: a case swaps one node onto a name mid-test, and a restart moves
	// the WHOLE network onto one no node has yet. A name nothing resolved
	// reaches exec as a path — that is how "upgrade" became `exec: "upgrade":
	// executable file not found`, with the declaration saying plainly that
	// upgrade means gstable.
	//
	// The node table's own resolutions win, because a node may name a binary
	// the env's map does not.
	if len(spec.Chain.Binaries) > 0 {
		merged := make(map[string]string, len(spec.Chain.Binaries)+len(resolvedBins))
		for name, path := range spec.Chain.Binaries {
			merged[name] = expand(path)
		}
		for name, path := range resolvedBins {
			merged[name] = path
		}
		resolvedBins = merged
	}

	binary := in.Binary
	from[FieldBinary] = origin.FromCommand
	if binary == "" {
		binary = expand(spec.Chain.Binary)
		from[FieldBinary] = origin.FromDeclaration
	}
	if binary == "" {
		// The declaration's own word for what the rest of the network runs. A
		// node table that assigns binaries to SOME nodes leaves the others on
		// this; before it was read, they were put on whichever binary the first
		// assigned node happened to name — so a four-node network with two on
		// the successor ran all four on the successor, and the plan said so
		// without anything looking wrong.
		// Expanded like every other binary reference. A declaration writes
		// ${GSTABLE_BIN:-gstable} here as readily as anywhere else, and this
		// lookup read it raw — so the plan showed the placeholder and exec would
		// have been handed it.
		binary = expand(spec.Chain.Binaries[dsl.BinaryDefault])
	}
	if binary == "" {
		// No default declared: fall back per node to the first node's binary, as
		// before. A node names its own binary over this.
		binary = topoBinary
	}
	if binary == "" {
		from[FieldBinary] = origin.FromDefault
		// Neither the run nor the declaration named one, so the chain does. A
		// definition that repeats the chain's own name for its binary is how
		// one binary came to be spelled two ways across the specs; leaving it
		// out is now the normal case, and chainsetup places the name.
		p, err := registry.Get(chain)
		if err != nil {
			return composition{}, lifecycle.Mark(errUnknownName, fmt.Errorf("no binary was given and chain %q is not known: %w", chain, err))
		}
		binary = p.Manifest().Binary
	}
	if binary == "" {
		return composition{}, lifecycle.Mark(errIncomplete, fmt.Errorf("no binary was given and chain %q names none", chain))
	}

	var validators, endpoints, proxies int
	var syncMode string
	var autoBP bool
	if inlineTopo == nil {
		validators, endpoints, proxies, syncMode, autoBP, err = topologyOf(spec.Topology)
		if err != nil {
			return composition{}, err
		}
		countFrom(from, FieldNodesBP, validators)
		countFrom(from, FieldNodesEN, endpoints)
		countFrom(from, FieldNodesPN, proxies)
		// An explicit --bp is a named count: it turns dynamic sizing off rather
		// than being filled over.
		if in.BPCount > 0 {
			validators = in.BPCount
			autoBP = false
			from[FieldNodesBP] = origin.FromCommand
		}
		if autoBP {
			// The unified model's default shape: one pn (the discovery hub on the
			// last server) and one en unless the spec said otherwise; the composer
			// fills the rest with validators once it knows the server count.
			if proxies == 0 {
				proxies = 1
				from[FieldNodesPN] = origin.FromDefault
			}
			if endpoints == 0 {
				endpoints = 1
				from[FieldNodesEN] = origin.FromDefault
			}
		} else if validators <= 0 {
			validators = suiteDefaultValidators
			from[FieldNodesBP] = origin.FromDefault
		}
	}
	// A node table is not recorded here: it names every node, and the nodes row
	// already prints "declared per node". Saying it twice is not saying it
	// better.
	// The request's flat launch opts and the network id join the "all" scope;
	// the env's scoped launch (per role or node) travels in LaunchScopedSet.
	launch := append([]string(nil), in.LaunchOpts...)
	if in.NetworkID != 0 {
		launch = append(launch, fmt.Sprintf("%s=%d", nodeconfig.KeyNetworkID, in.NetworkID))
	}
	up := &chainsetup.NetUpIn{
		DataDir: in.DataDir, Stage: chainsetup.UpStart,
		Chain: chain, Binary: binary, KeysDir: keysDir, KeysSource: keysSource,
		KeysValidators: keysValidators, BlueprintPath: expand(spec.EnvBlueprint),
		ManifestPath: expand(spec.Chain.ManifestPath), TemplatePath: expand(spec.Chain.TemplatePath),
		BPCount: validators, ENCount: endpoints, PNCount: proxies, EndpointSyncMode: syncMode,
		AutoSize: autoBP,
		Topology: inlineTopo, Binaries: resolvedBins, BinaryChains: spec.Chain.BinaryChains,
		GenesisFork: upgradeFork,
		Server:      in.Server, Docker: in.Docker,
		ChainID:          in.ChainID,
		GenesisSet:       hardforkSets(spec.Hardforks),
		OverlayPath:      overlayPath,
		GenesisPerBinary: perBinaryOverlay,
		GenesisExisting:  spec.Chain.GenesisExisting,
		LaunchSet:        launch,
		LaunchScoped:     spec.EnvLaunch,
		ConfigSet:        spec.EnvConfig,
	}
	// A pn is a proxy tier: it exists to keep endpoints off the producers, so a
	// topology that declares one composes as the proxied graph (bp <-> pn <-> en,
	// endpoints never dial a producer) rather than the default full mesh — a pn
	// under mesh would defeat its own purpose. A pn is declared either as a
	// count (count form) or as a node-table role, and both mean the same tier.
	if proxies > 0 || topologyHasProxy(inlineTopo) {
		up.Peering = string(node.Proxied)
	}
	// The env's target selects where the network is placed (local data root, a
	// server-set entry, or an ssh host). It fed only the reuse fingerprint
	// before, so a declared target shifted the key without moving the nodes;
	// thread it to the composition so it actually places them.
	from[FieldTarget] = origin.FromDefault
	if in.Server.All || in.Server.Name != "" || in.Server.SetPath != "" || in.Docker {
		from[FieldTarget] = origin.FromCommand
	}
	if spec.Placement != "" {
		tgt, perr := resource.Parse(spec.Placement)
		if perr != nil {
			return composition{}, fmt.Errorf("testengine: env target %q: %w", spec.Placement, perr)
		}
		if err := refuseMachineConflict(in, tgt, spec.Placement); err != nil {
			return composition{}, err
		}
		up.Target = tgt
		if from[FieldTarget] == origin.FromDefault {
			from[FieldTarget] = origin.FromDeclaration
		}
	}
	// The workspace-config owns the target data root (it moved off the server
	// set). Setting it here means `new` records it and every later step — and a
	// server selection through Retarget, which keeps a data root already set —
	// resolves paths under the root the environment named, not the workspace dir.
	var wcOrNil *resource.WorkspaceConfig
	if in.WorkspaceConfigPath != "" {
		wc, werr := resource.LoadWorkspaceConfig(in.WorkspaceConfigPath)
		if werr != nil {
			return composition{}, werr
		}
		// The workspace-config owns the data root, and the rule for folding it
		// onto a target lives with the config. All this path adds is which
		// target it was: the env's placement, so a refusal names the line the
		// operator has to change.
		target, werr := wc.AdoptDataRoot(up.Target, fmt.Sprintf("the env target %q", spec.Placement))
		if werr != nil {
			return composition{}, fmt.Errorf("testengine: %w", werr)
		}
		up.Target = target
		up.WorkspaceConfigPath = in.WorkspaceConfigPath
		if err := applyExistingInputs(up, wc, spec); err != nil {
			return composition{}, err
		}
		wcOrNil = &wc
	}
	// Once, here, because this path asks two questions about the launch before
	// the launch runs: the plan it prints, and the preflight comparison that
	// decides whether to reuse what is composed. Both were asking in the other
	// language. chainsetup places again on the way in, which is harmless.
	if err := chainsetup.PlaceRequest(up, wcOrNil); err != nil {
		return composition{}, err
	}
	return composition{up: up, from: from}, nil
}

// applyExistingInputs expands a named bundle of inputs that are already on the
// target onto the composition: the bundle's finished genesis and its keyring
// stand in for declaring them in the DSL, which is the point of naming a
// bundle. A field the DSL already declared is a conflict rather than a silent
// override. It runs only for inputs.mode existing; a generated run names no
// bundle (workspace-config validation ensures that).
func applyExistingInputs(up *chainsetup.NetUpIn, wc resource.WorkspaceConfig, spec dsl.Spec) error {
	if wc.Inputs.Mode != resource.InputExisting {
		return nil
	}
	name := wc.Inputs.Name
	existing := wc.ExistingInputs[name] // validated to exist at parse time
	if existing.Genesis != "" {
		if up.GenesisExisting != "" || len(spec.Chain.GenesisOverlay) > 0 {
			return fmt.Errorf("testengine: existing inputs %q set a genesis, but the spec already declares one — declare it in one place", name)
		}
		up.GenesisExisting = existing.Genesis
	}
	if existing.Keyring != "" {
		if spec.EnvKeys != nil {
			return fmt.Errorf("testengine: existing inputs %q set a keyring, but the spec already declares keys — declare them in one place", name)
		}
		// A local key set is used in place; a keyring on a server (srv://) is
		// downloaded to a local directory by the keys step (materializeKeyring)
		// so the ring is read the one local way and a node signs with keys at a
		// known local path. A bare relative name is neither, and is rejected here
		// rather than mistaken for a local directory.
		if !strings.HasPrefix(existing.Keyring, "srv://") && !filepath.IsAbs(existing.Keyring) {
			return fmt.Errorf("testengine: existing inputs %q keyring %q must be a local absolute path or a srv:// reference", name, existing.Keyring)
		}
		up.KeysDir = existing.Keyring
		// The key SOURCE is a different preset: it says the ring is read as
		// recorded rather than generated. Naming the bundle "existing inputs"
		// is what keeps these two readable in one function.
		up.KeysSource = "keyPreset"
	}
	applyExistingConfigs(up, existing)
	return nil
}

// applyExistingConfigs resolves each node's logical config name to the file the
// bundle names. A node table's config value is a logical name when it is a key
// in the bundle's configs map: the DSL names a config, and the environment says
// which file that name is on this target, so one spec runs against different
// targets by swapping the map. A config value that is not a key is left as a
// direct file reference, which is how a node named its config before bundles
// existed. With no node table there is nothing to map onto, and the map is
// simply unused.
func applyExistingConfigs(up *chainsetup.NetUpIn, existing resource.ExistingInputs) {
	if len(existing.Configs) == 0 || up.Topology == nil {
		return
	}
	for i := range up.Topology.Nodes {
		logical := up.Topology.Nodes[i].Config
		if logical == "" {
			continue
		}
		if file, ok := existing.Configs[logical]; ok {
			up.Topology.Nodes[i].Config = file
		}
	}
}

// topologyHasKeys reports whether a node-table declaration pins any per-node
// key. A table that does builds its own key set (declared where given,
// generated where not), so its keys belong in a workspace-local dir rather than
// the shared preset. It reads the raw declaration defensively: a malformed
// nodes list is the node-table parser's error to report, not this peek's.
