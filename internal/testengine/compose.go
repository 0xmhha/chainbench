package testengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/resource"
)

// A suite composes the network its specs declare. The declaration is the
// env half of a v2 case (or a v1 spec's chain block); this file turns it into
// the request a composer takes. There are two composers — the workspace
// steps for a single-binary network, and the handoff for a mixed-binary one —
// and the declaration's shape picks between them. Nothing here asks which
// chain it is.

// suiteDefaultValidators sizes a network whose env declares no topology: the
// BFT floor that tolerates one fault.
const suiteDefaultValidators = 4

// Handoff composition timing.
const (
	// etcdFormWait bounds the wait for the producer's etcd cluster to form.
	etcdFormWait = 60 * time.Second
	// forkWait bounds the wait for the successor set to produce past the fork.
	// It is not the wait for one block: AwaitFork now requires ten, so a chain
	// with a one-second block period needs that much more headroom.
	//
	// The comment below is the original one, kept because the bound it chose is
	// still the bound: the wait for a successor to seal the first post-fork
	// block. The profile's fork height and block time decide the real figure;
	// this is the ceiling.
	forkWait = 180 * time.Second
)

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
const defaultKeysDir = "keys/preset"

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
func countFrom(from map[PlanField]PlanSource, f PlanField, count int) {
	if count > 0 {
		from[f] = SourceDeclaration
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
	up      *chainsetup.NetUpIn
	handoff *upgrade.HandoffInputs
	from    map[PlanField]PlanSource
}

// compositionOf reads the network a spec declares and applies the caller's
// overrides (a binary path, a key set, a validator count, a server) on top.
// A v1 spec declares through its chain block; a v2 case through its env.
func compositionOf(ctx context.Context, spec dsl.Spec, in RunSuiteIn) (composition, error) {
	chain := spec.Chain.Name
	if in.Chain != "" && in.Chain != chain {
		return composition{}, fmt.Errorf("the request names chain %q but the spec declares %q", in.Chain, chain)
	}
	from := map[PlanField]PlanSource{}
	keysDir := in.KeysDir
	keysSource := in.KeysSource
	if keysSource != "" {
		from[FieldKeysSource] = SourceCommand
	}
	if keysDir != "" {
		from[FieldKeysDir] = SourceCommand
	}
	keysValidators := 0
	if k := spec.EnvKeys; k != nil {
		if keysSource == "" && k.Source != "" {
			keysSource = k.Source
			from[FieldKeysSource] = SourceDeclaration
		}
		if keysDir == "" && k.Ref != "" {
			keysDir = expand(k.Ref)
			from[FieldKeysDir] = SourceDeclaration
		}
		keysValidators = k.Validators
	}
	if keysSource == "" {
		from[FieldKeysSource] = SourceHarness
	}
	if keysDir == "" {
		// A generated set — or a node table that pins per-node keys — goes to a
		// workspace-local dir, not the shared preset. The key sources reuse
		// whatever set already sits at the dir, so defaulting to keys/preset
		// would silently reuse the preset's identities (and ignore the pinned
		// keys, or fail when the network wants more than the preset holds)
		// instead of building a fresh set. Preset stays the default otherwise.
		if keysSource == keySourceGenerate || topologyHasKeys(spec.Topology) {
			keysDir = filepath.Join(in.DataDir, generatedKeysSubdir)
		} else {
			keysDir = defaultKeysDir
		}
		from[FieldKeysDir] = SourceHarness
	}
	var upgradeFork *chainsetup.GenesisFork
	perBinaryOverlay, err := writeOverlays(ctx, in.DataDir, spec.Chain.GenesisPerBinary)
	if err != nil {
		return composition{}, err
	}
	overlayPath, err := writeOverlay(ctx, in.DataDir, spec.Chain.GenesisOverlay)
	if err != nil {
		return composition{}, err
	}

	// An upgrade env that declares a node table composes like any other network:
	// the nodes say which build each of them runs, and the fork's configuration
	// is scheduled on the one genesis they all initialize from. One without a
	// table still goes to the handoff composer, which sizes the network from its
	// preset's roles.
	if u := spec.EnvUpgrade; u != nil && len(spec.Topology) > 0 {
		fork, ferr := forkOf(u)
		if ferr != nil {
			return composition{}, ferr
		}
		upgradeFork = fork
	} else if u := spec.EnvUpgrade; u != nil {
		if in.Binary != "" {
			return composition{}, fmt.Errorf("a handoff names its binaries by role in the env; --binary does not apply")
		}
		if in.ChainID != 0 || in.NetworkID != 0 || len(in.LaunchOpts) > 0 || in.KeysSource != "" {
			return composition{}, fmt.Errorf("a handoff composes from its declaration; genesis, launch, and key-source overrides do not apply")
		}
		// A handoff composes its network from the profile and template, so
		// env-level hardforks, topology, launch, and config have nowhere to go.
		// Refuse them loudly rather than parse an upgrade env that carries them
		// and silently drop half its declaration.
		if len(spec.Hardforks) > 0 || len(spec.Topology) > 0 || len(spec.EnvLaunch) > 0 || len(spec.EnvConfig) > 0 {
			return composition{}, fmt.Errorf("a handoff composes from its profile and template; env hardforks, topology, launch, and config do not apply — the network's size lives in the profile's roles (producers, validators), together with the identity order, validator addresses and extradata that have to agree with it, so run a different profile to run a different size")
		}
		from, to := u.From, u.To
		if from == "" {
			from = dsl.BinaryFrom
		}
		if to == "" {
			to = dsl.BinaryTo
		}
		hi := upgrade.HandoffInputs{
			ProfilePath:    upgradePresetPath(u),
			Template:       expand(u.Template),
			KeysDir:        keysDir,
			FromBinary:     expand(spec.Chain.Binaries[from]),
			ToBinary:       expand(spec.Chain.Binaries[to]),
			GenesisOverlay: overlayPath,
			DataDir:        in.DataDir,
		}
		// What the case says about the fork is checked against the preset that
		// decides it. A case naming the wrong fork or the wrong block would
		// otherwise run happily against another one and report a pass.
		if err := checkDeclaredFork(u, hi.ProfilePath); err != nil {
			return composition{}, err
		}
		// Where its nodes run, through the same resolver `chainbench upgrade`
		// uses. A handoff is a network like any other in this respect: it is
		// placed on a server set or on this machine, and which one is the
		// caller's to say. This surface used to fill seven fields and stop, so
		// a case declaring a handoff ran locally whatever the operator asked
		// for, and said nothing about the server, the target or the environment
		// file it had been given.
		wc, terr := upgradeTarget(in).Apply(&hi, handoffNodes(hi.ProfilePath))
		if terr != nil {
			return composition{}, terr
		}
		// And which file the binary names point at over there, the same way the
		// composition path places its own.
		if wc != nil {
			for _, b := range []*string{&hi.FromBinary, &hi.ToBinary} {
				placed, perr := chainsetup.PlaceBinary(*b, wc)
				if perr != nil {
					return composition{}, perr
				}
				*b = placed
			}
		}
		return composition{handoff: &hi}, nil
	}

	// A node table (topology.nodes[]) declares each node's role and binary
	// explicitly; its absence keeps the count form (validators/endpoints).
	inlineTopo, resolvedBins, topoBinary, err := inlineTopologyOf(chain, spec.Topology, spec.Chain.Binaries)
	if err != nil {
		return composition{}, err
	}
	// The count form has no node table to resolve names against, and the names
	// were being dropped with it. A declaration that names binaries is not only
	// talking about the nodes it composes: a case swaps one node onto a name
	// mid-test, and a name nothing resolved reaches exec as a path. That is how
	// "upgrade" became `exec: "upgrade": executable file not found`, with the
	// declaration saying plainly that upgrade means gstable.
	if inlineTopo == nil && len(spec.Chain.Binaries) > 0 {
		resolvedBins = map[string]string{}
		for name, path := range spec.Chain.Binaries {
			resolvedBins[name] = expand(path)
		}
	}

	binary := in.Binary
	from[FieldBinary] = SourceCommand
	if binary == "" {
		binary = expand(spec.Chain.Binary)
		from[FieldBinary] = SourceDeclaration
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
		from[FieldBinary] = SourceHarness
		// Neither the run nor the declaration named one, so the chain does. A
		// definition that repeats the chain's own name for its binary is how
		// one binary came to be spelled two ways across the specs; leaving it
		// out is now the normal case, and chainsetup places the name.
		p, err := registry.Get(chain)
		if err != nil {
			return composition{}, fmt.Errorf("no binary was given and chain %q is not known: %w", chain, err)
		}
		binary = p.Manifest().Binary
	}
	if binary == "" {
		return composition{}, fmt.Errorf("no binary was given and chain %q names none", chain)
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
			from[FieldNodesBP] = SourceCommand
		}
		if autoBP {
			// The unified model's default shape: one pn (the discovery hub on the
			// last server) and one en unless the spec said otherwise; the composer
			// fills the rest with validators once it knows the server count.
			if proxies == 0 {
				proxies = 1
				from[FieldNodesPN] = SourceHarness
			}
			if endpoints == 0 {
				endpoints = 1
				from[FieldNodesEN] = SourceHarness
			}
		} else if validators <= 0 {
			validators = suiteDefaultValidators
			from[FieldNodesBP] = SourceHarness
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
	from[FieldTarget] = SourceHarness
	if in.Server.All || in.Server.Name != "" || in.Server.SetPath != "" || in.Docker {
		from[FieldTarget] = SourceCommand
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
		if from[FieldTarget] == SourceHarness {
			from[FieldTarget] = SourceDeclaration
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
func topologyHasKeys(t map[string]any) bool {
	list, ok := t["nodes"].([]any)
	if !ok {
		return false
	}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if s, ok := m["key"].(string); ok && s != "" {
			return true
		}
	}
	return false
}

// topologyHasProxy reports whether a node-table topology declares a pn, so the
// composer selects the proxied graph for it exactly as it does for the count
// form. It is nil-safe: the count form passes no table.
func topologyHasProxy(t *node.Topology) bool {
	if t == nil {
		return false
	}
	for _, n := range t.Nodes {
		if node.Is(n.NodeRole(), node.RolePN) {
			return true
		}
	}
	return false
}

// inlineTopologyOf builds an in-memory node table from a topology.nodes[]
// declaration, so a spec can name each node's role and binary in one file. It
// returns nil when the declaration uses the count form (no nodes key), which
// keeps the existing validators/endpoints path.
//
// The second result maps each binary name a node references to its resolved
// path, drawn from the env's binaries map; the third is a fallback binary (the
// first node's) for a node that names none.
func inlineTopologyOf(chain string, t map[string]any, binaries map[string]string) (*node.Topology, map[string]string, string, error) {
	raw, ok := t["nodes"]
	if !ok {
		return nil, nil, "", nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, nil, "", fmt.Errorf("topology.nodes must be a list")
	}
	if len(list) == 0 {
		return nil, nil, "", fmt.Errorf("topology.nodes is empty")
	}
	topo := &node.Topology{Chain: chain, Nodes: make([]node.Entry, 0, len(list))}
	resolved := map[string]string{}
	fallback := ""
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, nil, "", fmt.Errorf("topology.nodes[%d] must be a mapping", i)
		}
		entry := node.Entry{Index: i + 1}
		for k, v := range m {
			s, isStr := v.(string)
			switch k {
			case "role":
				if !isStr {
					return nil, nil, "", fmt.Errorf("topology.nodes[%d].role must be a string", i)
				}
				entry.Role = s
			case "binary":
				if !isStr {
					return nil, nil, "", fmt.Errorf("topology.nodes[%d].binary must be a string", i)
				}
				entry.Binary = s
			case "sync", topoSyncMode, topoSyncModeSnak:
				if !isStr {
					return nil, nil, "", fmt.Errorf("topology.nodes[%d].%s must be a string", i, k)
				}
				entry.SyncMode = s
			case "bootnode":
				b, isBool := v.(bool)
				if !isBool {
					return nil, nil, "", fmt.Errorf("topology.nodes[%d].bootnode must be a boolean", i)
				}
				entry.Bootnode = b
			case "config":
				if !isStr {
					return nil, nil, "", fmt.Errorf("topology.nodes[%d].config must be a string (a path to a pre-written config file)", i)
				}
				entry.Config = expand(s)
			case "key":
				if !isStr {
					return nil, nil, "", fmt.Errorf("topology.nodes[%d].key must be a string (a key file path or 0x-hex)", i)
				}
				entry.Key = expand(s)
			case "index":
				n, ferr := countOf("nodes[].index", v)
				if ferr != nil {
					return nil, nil, "", ferr
				}
				entry.Index = n
			default:
				return nil, nil, "", fmt.Errorf("topology.nodes[%d].%s is not a key the composer knows (role, binary, sync, bootnode, index, config, key)", i, k)
			}
		}
		if entry.Role == "" {
			return nil, nil, "", fmt.Errorf("topology.nodes[%d] needs a role", i)
		}
		if entry.Binary != "" {
			path, named := binaries[entry.Binary]
			if !named {
				return nil, nil, "", fmt.Errorf("topology.nodes[%d].binary %q is not declared in binaries", i, entry.Binary)
			}
			p := expand(path)
			resolved[entry.Binary] = p
			if fallback == "" {
				fallback = p
			}
		}
		topo.Nodes = append(topo.Nodes, entry)
	}
	if err := topo.Validate(); err != nil {
		return nil, nil, "", err
	}
	return topo, resolved, fallback, nil
}

// Topology keys a declaration may use for its node counts.
const (
	topoBP           = "bp"
	topoEN           = "en"
	topoPN           = "pn"
	topoSyncMode     = "syncMode"
	topoSyncModeSnak = "sync_mode"
	// topoMax is the bp value that fills the network to the server set instead
	// of naming a count: bp becomes one node per server, less the pn and en.
	topoMax = "max"
)

// topologyOf reads the node counts a declaration gives: validators (or bp),
// endpoints (or en), and the endpoints' sync mode. A key it does not know is
// an error rather than a silently ignored intention.
//
// bp may be the word "max" instead of a number: autoBP is then true and the
// validator count is left for the composer to fill from the server set.
func topologyOf(t map[string]any) (validators, endpoints, proxies int, syncMode string, autoBP bool, err error) {
	for k, v := range t {
		switch k {
		case topoBP:
			if s, ok := v.(string); ok {
				if s != topoMax {
					return 0, 0, 0, "", false, fmt.Errorf("topology.%s must be a number or %q, got %q", k, topoMax, s)
				}
				autoBP = true
				break
			}
			validators, err = countOf(k, v)
		case topoEN:
			endpoints, err = countOf(k, v)
		case topoPN:
			proxies, err = countOf(k, v)
		case topoSyncMode, topoSyncModeSnak:
			s, ok := v.(string)
			if !ok {
				err = fmt.Errorf("topology.%s must be a string", k)
			}
			syncMode = s
		default:
			err = fmt.Errorf("topology.%s is not a key the composer knows (bp, en, pn, syncMode)", k)
		}
		if err != nil {
			return 0, 0, 0, "", false, err
		}
	}
	return validators, endpoints, proxies, syncMode, autoBP, nil
}

// countOf reads a node count, which JSON hands over as a float.
func countOf(key string, v any) (int, error) {
	switch n := v.(type) {
	case float64:
		if n < 0 || n != float64(int(n)) {
			return 0, fmt.Errorf("topology.%s must be a whole non-negative number, got %v", key, v)
		}
		return int(n), nil
	case int:
		if n < 0 {
			return 0, fmt.Errorf("topology.%s must be non-negative, got %d", key, n)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("topology.%s must be a number, got %T", key, v)
	}
}

// hardforkSets renders declared fork heights as the genesis step's config
// overrides: {"boho": 10} becomes bohoBlock=10. Sorted, so the recorded step
// detail is stable.
func hardforkSets(forks map[string]int) []string {
	names := make([]string, 0, len(forks))
	for name := range forks {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, fmt.Sprintf("%sBlock=%d", name, forks[name]))
	}
	return out
}

// writeOverlay puts a declared genesis overlay where the genesis step reads
// overlays from: a file under the workspace, written through the file seam
// like everything else the workspace holds. No overlay writes nothing and
// returns no path.
func writeOverlay(ctx context.Context, dataDir string, overlay map[string]any) (string, error) {
	if len(overlay) == 0 {
		return "", nil
	}
	b, err := json.MarshalIndent(map[string]any{"genesis": overlay}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("render genesis overlay: %w", err)
	}
	sum := sha256.Sum256(b)
	path := filepath.Join(dataDir, fmt.Sprintf("%s-%s.json", overlayFilePrefix, hex.EncodeToString(sum[:8])))
	if err := (filestore.Local{}).Write(ctx, path, b, 0o644); err != nil {
		return "", fmt.Errorf("write genesis overlay: %w", err)
	}
	return path, nil
}

// forkOf reads the fork a declaration schedules, taking from the preset what the
// case did not say.
//
// The preset decides the fork and the block; a case may repeat them and is held
// to the repetition (see checkDeclaredFork). Here the two are folded into the
// one instruction the genesis step acts on.
func forkOf(u *dsl.UpgradeV2) (*chainsetup.GenesisFork, error) {
	prof, err := upgrade.LoadProfile(upgradePresetPath(u))
	if err != nil {
		return nil, fmt.Errorf("upgrade preset: %w", err)
	}
	name, at := prof.Upgrade.AtFork, prof.Upgrade.ForkBlock
	if u.Fork != "" {
		name = u.Fork
	}
	if u.At != nil {
		at = *u.At
	}
	if name == "" {
		return nil, fmt.Errorf("upgrade: neither the case nor its preset names a fork")
	}
	to := u.To
	if to == "" {
		to = dsl.BinaryTo
	}
	return &chainsetup.GenesisFork{Name: name, At: at, Binary: to}, nil
}

// hardforkPresetDir is where a named hardfork preset lives.
const hardforkPresetDir = "presets/hardfork"

// upgradePresetPath is the preset file this declaration names: a path when it
// gave one, and otherwise the named preset under presets/hardfork.
func upgradePresetPath(u *dsl.UpgradeV2) string {
	if u.Profile != "" {
		return expand(u.Profile)
	}
	return filepath.Join(hardforkPresetDir, u.Preset+".yaml")
}

// checkDeclaredFork holds a case to what it said about the fork.
//
// The preset decides which fork and which block; a case may repeat them, and a
// repetition that disagrees is the case testing something other than what it
// claims. Saying nothing is fine — the preset answers.
func checkDeclaredFork(u *dsl.UpgradeV2, presetPath string) error {
	if u.Fork == "" && u.At == nil {
		return nil
	}
	prof, err := upgrade.LoadProfile(presetPath)
	if err != nil {
		return fmt.Errorf("upgrade preset: %w", err)
	}
	if u.Fork != "" && u.Fork != prof.Upgrade.AtFork {
		return fmt.Errorf("the case says it tests the %q fork and %s schedules %q", u.Fork, presetPath, prof.Upgrade.AtFork)
	}
	if u.At != nil && *u.At != prof.Upgrade.ForkBlock {
		return fmt.Errorf("the case says the fork is at block %d and %s schedules block %d", *u.At, presetPath, prof.Upgrade.ForkBlock)
	}
	return nil
}

// upgradeTarget is where a handoff's nodes run, read from the same request
// fields the composition path reads them from.
func upgradeTarget(in RunSuiteIn) upgrade.Target {
	return upgrade.Target{
		Server:              in.Server,
		AllServers:          in.Server.All,
		WorkspaceConfigPath: in.WorkspaceConfigPath,
		Docker:              in.Docker,
	}
}

// handoffNodes is how many nodes the profile places, which the placement needs
// before the handoff is built. A profile that cannot be read yields zero and
// lets NewHandoff report it, so one unreadable profile is not two errors.
func handoffNodes(profilePath string) int {
	prof, err := upgrade.LoadProfile(profilePath)
	if err != nil {
		return 0
	}
	return prof.Roles.Producers + prof.Roles.Validators
}

// writeOverlays renders one overlay file per binary, the same way the network's
// own overlay is rendered, and returns where each landed.
func writeOverlays(ctx context.Context, dataDir string, per map[string]map[string]any) (map[string]string, error) {
	if len(per) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(per))
	for _, name := range slices.Sorted(maps.Keys(per)) {
		path, err := writeOverlay(ctx, dataDir, per[name])
		if err != nil {
			return nil, fmt.Errorf("binary %s: %w", name, err)
		}
		out[name] = path
	}
	return out, nil
}

// handoffUp composes a mixed-binary network: the handoff's steps in order,
// each recorded, up to a successor sealing past the fork. It returns the
// running nodes and a teardown. A failure after the nodes launched stops
// them, because a handoff has no workspace a later command could reach them
// through; their logs stay under the data dir.
func handoffUp(ctx context.Context, in upgrade.HandoffInputs) (node.NodeSet, []string, func(context.Context) error, error) {
	var steps []string
	record := func(name, detail string) { steps = append(steps, name+": "+detail) }
	fail := func(name string, err error) (node.NodeSet, []string, func(context.Context) error, error) {
		return node.NodeSet{}, steps, nil, fmt.Errorf("handoff: %s: %w", name, err)
	}

	h, err := upgrade.NewHandoff(in)
	if err != nil {
		return fail("prepare", err)
	}
	record("prepare", h.Describe())
	cfg, err := h.WriteConfig(ctx)
	if err != nil {
		return fail("config", err)
	}
	record("config", cfg)
	base, err := h.BaseGenesis(ctx)
	if err != nil {
		return fail("base-genesis", err)
	}
	record("base-genesis", base)
	if err := h.ComposePlan(ctx, base); err != nil {
		return fail("plan", err)
	}
	record("plan", fmt.Sprintf("%d node(s); fork section %q merged", len(h.Plan.Nodes), h.Plan.AtFork))
	detail, err := h.ApplyOverlay()
	if err != nil {
		return fail("overlay", err)
	}
	record("overlay", detail)

	// One bring-up, the family's. This path used to launch every node at once
	// and deploy governance afterwards, which is the order a poa cluster cannot
	// form in: it forms only while the producer is alone. It worked because the
	// profile has one producer and the four successors run the other binary.
	ns, err := h.BringUp(ctx, record)
	if err != nil {
		return fail("bring-up", err)
	}
	teardown := func(ctx context.Context) error {
		_, errs := process.StopNodeSet(ctx, process.NewLocalDriver(), ns)
		if len(errs) > 0 {
			return fmt.Errorf("handoff: teardown: %v", errs)
		}
		return nil
	}
	producer := ns.Nodes[0]
	record("producer", producer.RPCURL)
	live := func(name string, fn func() (string, error)) error {
		detail, err := fn()
		if err != nil {
			_ = teardown(ctx)
			return fmt.Errorf("handoff: %s: %w", name, err)
		}
		record(name, detail)
		return nil
	}
	if err := live("verify-etcd", func() (string, error) {
		info, err := h.VerifyEtcd(ctx, producer, etcdFormWait)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("governance %s, etcd cluster %q", info.Governance, info.Cluster()), nil
	}); err != nil {
		return node.NodeSet{}, steps, nil, err
	}
	if err := live("await-fork", func() (string, error) { return h.AwaitFork(ctx, ns, forkWait) }); err != nil {
		return node.NodeSet{}, steps, nil, err
	}
	return ns, steps, teardown, nil
}

// handoffEndpoints orders a handoff network's RPC URLs successors first: the
// producer cannot import post-fork blocks, so it must not be the primary the
// tests read from.
func handoffEndpoints(ns node.NodeSet) []string {
	var successors, producers []string
	for _, n := range ns.Nodes {
		if n.Index == 0 {
			producers = append(producers, n.RPCURL)
			continue
		}
		successors = append(successors, n.RPCURL)
	}
	return append(successors, producers...)
}

// sameComposition checks that every spec in a suite declares the SAME network,
// not merely the same chain.
//
// One run composes one network, and it composes it from the first spec. A spec
// further down the list that declares a different genesis, topology or binary
// does not get the network it asked for — it runs against the first spec's, and
// its assertions are answered by the wrong chain. That failure is silent, which
// is the worst kind: six genesis-string cases each declaring their own
// authorizedAccounts would all be answered by the first one's genesis and five
// of them would report a wrong count as a real result.
//
// So the disagreement is refused here, before anything is allocated, and the
// message names what differs so the caller can split the run.
func sameComposition(specs []dsl.Spec) error {
	if len(specs) < 2 {
		return nil
	}
	want := compositionKey(specs[0])
	var others []string
	for _, s := range specs[1:] {
		if compositionKey(s) != want {
			others = append(others, s.ID)
		}
	}
	if len(others) == 0 {
		return nil
	}
	return fmt.Errorf(
		"one run composes one network, from the first spec (%s); these declare a different one: %s. "+
			"run them separately, or give them the same env",
		specs[0].ID, strings.Join(others, ", "))
}

// compositionKey is what makes two specs the same network to compose: the
// binaries, genesis, config, topology, hardforks and placement. It deliberately
// mirrors the reuse fingerprint's inputs — a run that may share one network is
// exactly a run whose specs would fingerprint alike.
func compositionKey(s dsl.Spec) string {
	key := struct {
		Binary    string            `json:"binary"`
		Binaries  map[string]string `json:"binaries"`
		Config    string            `json:"config"`
		Genesis   map[string]any    `json:"genesis"`
		Topology  map[string]any    `json:"topology"`
		Hardforks map[string]int    `json:"hardforks"`
		Placement string            `json:"placement"`
	}{
		Binary: s.Chain.Binary, Binaries: s.Chain.Binaries, Config: s.Chain.Config,
		Genesis: s.Chain.GenesisOverlay, Topology: s.Topology,
		Hardforks: s.Hardforks, Placement: s.Placement,
	}
	b, err := json.Marshal(key)
	if err != nil {
		return fmt.Sprintf("composition-error:%v", err)
	}
	return string(b)
}

// sameChain checks that every parsed spec declares the chain the first one
// does: one suite composes one network.
func sameChain(specs []dsl.Spec) error {
	if len(specs) == 0 {
		return nil
	}
	want := specs[0].Chain.Name
	var others []string
	for _, s := range specs[1:] {
		if s.Chain.Name != want {
			others = append(others, s.ID+"="+s.Chain.Name)
		}
	}
	if len(others) > 0 {
		return fmt.Errorf("every spec in a suite must declare one chain; %s declares %s, but: %s",
			specs[0].ID, want, strings.Join(others, ", "))
	}
	return nil
}
