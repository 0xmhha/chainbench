package testengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
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
// reading a recorded one.
const keySourceGenerate = "generate"

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

// composition is what one suite composes: a single-binary network through
// the workspace steps, or a mixed-binary handoff. Exactly one is set.
type composition struct {
	up      *chainsetup.NetUpIn
	handoff *upgrade.HandoffInputs
}

// compositionOf reads the network a spec declares and applies the caller's
// overrides (a binary path, a key set, a validator count, a server) on top.
// A v1 spec declares through its chain block; a v2 case through its env.
func compositionOf(ctx context.Context, spec dsl.Spec, in RunSuiteIn) (composition, error) {
	chain := spec.Chain.Name
	if in.Chain != "" && in.Chain != chain {
		return composition{}, fmt.Errorf("the request names chain %q but the spec declares %q", in.Chain, chain)
	}
	keysDir := in.KeysDir
	keysSource := in.KeysSource
	keysValidators := 0
	if k := spec.EnvKeys; k != nil {
		if keysSource == "" {
			keysSource = k.Source
		}
		if keysDir == "" {
			keysDir = expand(k.Ref)
		}
		keysValidators = k.Validators
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
	}
	overlayPath, err := writeOverlay(ctx, in.DataDir, spec.Chain.GenesisOverlay)
	if err != nil {
		return composition{}, err
	}

	if u := spec.EnvUpgrade; u != nil {
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
			return composition{}, fmt.Errorf("a handoff composes from its profile and template; env hardforks, topology, launch, and config do not apply")
		}
		return composition{handoff: &upgrade.HandoffInputs{
			ProfilePath:    expand(u.Profile),
			Template:       expand(u.Template),
			PresetDir:      keysDir,
			FromBinary:     expand(spec.Chain.Binaries[dsl.BinaryBefore]),
			ToBinary:       expand(spec.Chain.Binaries[dsl.BinaryAfter]),
			GenesisOverlay: overlayPath,
			DataDir:        in.DataDir,
		}}, nil
	}

	// A node table (topology.nodes[]) declares each node's role and binary
	// explicitly; its absence keeps the count form (validators/endpoints).
	inlineTopo, resolvedBins, topoBinary, err := inlineTopologyOf(chain, spec.Topology, spec.Chain.Binaries)
	if err != nil {
		return composition{}, err
	}

	binary := in.Binary
	if binary == "" {
		binary = expand(spec.Chain.Binary)
	}
	if binary == "" {
		// With a node table but no single binary, launch falls back per node to
		// the first node's binary; a node names its own binary over this.
		binary = topoBinary
	}
	if binary == "" {
		return composition{}, fmt.Errorf("the spec declares no binary and none was given")
	}

	var validators, endpoints, proxies int
	var syncMode string
	var autoBP bool
	if inlineTopo == nil {
		validators, endpoints, proxies, syncMode, autoBP, err = topologyOf(spec.Topology)
		if err != nil {
			return composition{}, err
		}
		// An explicit --validators is a named count: it turns dynamic sizing off
		// rather than being filled over.
		if in.Validators > 0 {
			validators = in.Validators
			autoBP = false
		}
		if autoBP {
			// The unified model's default shape: one pn (the discovery hub on the
			// last server) and one en unless the spec said otherwise; the composer
			// fills the rest with validators once it knows the server count.
			if proxies == 0 {
				proxies = 1
			}
			if endpoints == 0 {
				endpoints = 1
			}
		} else if validators <= 0 {
			validators = suiteDefaultValidators
		}
	}
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
		Validators: validators, Endpoints: endpoints, Proxies: proxies, EndpointSyncMode: syncMode,
		AutoSize: autoBP,
		Topology: inlineTopo, Binaries: resolvedBins,
		Server: in.Server, Docker: in.Docker,
		ChainID:         in.ChainID,
		GenesisSet:      hardforkSets(spec.Hardforks),
		OverlayPath:     overlayPath,
		GenesisExisting: spec.Chain.GenesisExisting,
		LaunchSet:       launch,
		LaunchScoped:    spec.EnvLaunch,
		ConfigSet:       spec.EnvConfig,
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
	if spec.Placement != "" {
		tgt, perr := resource.Parse(spec.Placement)
		if perr != nil {
			return composition{}, fmt.Errorf("testengine: env target %q: %w", spec.Placement, perr)
		}
		up.Target = tgt
	}
	// The workspace-config owns the target data root (it moved off the server
	// set). Setting it here means `new` records it and every later step — and a
	// server selection through Retarget, which keeps a data root already set —
	// resolves paths under the root the environment named, not the workspace dir.
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
		if err := applyPreset(up, wc, spec); err != nil {
			return composition{}, err
		}
	}
	return composition{up: up}, nil
}

// applyPreset expands a prepared input preset onto the composition: the
// preset's finished genesis and its keyring stand in for declaring them in the
// DSL, which is the point of naming a bundle. A field the DSL already declared
// is a conflict rather than a silent override. It runs only for inputs.mode
// prepared; a generated run has no preset (workspace-config validation ensures
// that).
func applyPreset(up *chainsetup.NetUpIn, wc resource.WorkspaceConfig, spec dsl.Spec) error {
	if wc.Inputs.Mode != resource.InputPrepared {
		return nil
	}
	name := wc.Inputs.Preset
	preset := wc.Presets[name] // validated to exist at parse time
	if preset.Genesis != "" {
		if up.GenesisExisting != "" || len(spec.Chain.GenesisOverlay) > 0 {
			return fmt.Errorf("testengine: preset %q sets a genesis, but the spec already declares one — declare it in one place", name)
		}
		up.GenesisExisting = preset.Genesis
	}
	if preset.Keyring != "" {
		if spec.EnvKeys != nil {
			return fmt.Errorf("testengine: preset %q sets a keyring, but the spec already declares keys — declare them in one place", name)
		}
		// A local key set is used in place; a keyring on a server (srv://) is
		// downloaded to a local directory by the keys step (materializeKeyring)
		// so the ring is read the one local way and a node signs with keys at a
		// known local path. A bare relative name is neither, and is rejected here
		// rather than mistaken for a local directory.
		if !strings.HasPrefix(preset.Keyring, "srv://") && !filepath.IsAbs(preset.Keyring) {
			return fmt.Errorf("testengine: preset %q keyring %q must be a local absolute path or a srv:// reference", name, preset.Keyring)
		}
		up.KeysDir = preset.Keyring
		up.KeysSource = "preset"
	}
	applyPresetConfigs(up, preset)
	return nil
}

// applyPresetConfigs resolves each node's logical config name to the preset's
// file. A node table's config value is a logical name when it is a key in the
// preset's configs map: the DSL names a config, and the environment's preset
// says which file that name is on this target, so one spec runs against
// different targets by swapping the map. A config value that is not a preset
// key is left as a direct file reference, which is how a node named its config
// before presets existed. With no node table there is nothing to map onto, and
// the map is simply unused.
func applyPresetConfigs(up *chainsetup.NetUpIn, preset resource.InputPreset) {
	if len(preset.Configs) == 0 || up.Topology == nil {
		return
	}
	for i := range up.Topology.Nodes {
		logical := up.Topology.Nodes[i].Config
		if logical == "" {
			continue
		}
		if file, ok := preset.Configs[logical]; ok {
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
	topoValidators   = "validators"
	topoBP           = "bp"
	topoEndpoints    = "endpoints"
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
		case topoValidators, topoBP:
			if s, ok := v.(string); ok {
				if s != topoMax {
					return 0, 0, 0, "", false, fmt.Errorf("topology.%s must be a number or %q, got %q", k, topoMax, s)
				}
				autoBP = true
				break
			}
			validators, err = countOf(k, v)
		case topoEndpoints, topoEN:
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
			err = fmt.Errorf("topology.%s is not a key the composer knows (validators|bp, endpoints|en, pn, syncMode)", k)
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

	ns, err := h.Launch(ctx)
	if err != nil {
		return fail("launch", err)
	}
	if len(ns.Nodes) == 0 {
		return fail("launch", fmt.Errorf("no nodes launched"))
	}
	teardown := func(ctx context.Context) error {
		_, errs := process.StopNodeSet(ctx, process.NewLocalDriver(), ns)
		if len(errs) > 0 {
			return fmt.Errorf("handoff: teardown: %v", errs)
		}
		return nil
	}
	producer := ns.Nodes[0]
	record("launch", fmt.Sprintf("%d node(s); producer %s", len(ns.Nodes), producer.RPCURL))
	live := func(name string, fn func() (string, error)) error {
		detail, err := fn()
		if err != nil {
			_ = teardown(ctx)
			return fmt.Errorf("handoff: %s: %w", name, err)
		}
		record(name, detail)
		return nil
	}
	if err := live("mesh", func() (string, error) {
		return fmt.Sprintf("%d endpoint(s) meshed", len(ns.Nodes)), h.WireMesh(ctx, ns)
	}); err != nil {
		return node.NodeSet{}, steps, nil, err
	}
	if err := live("governance", func() (string, error) {
		return "deployed (effect checked by verify-etcd)", h.DeployGovernance(ctx, producer)
	}); err != nil {
		return node.NodeSet{}, steps, nil, err
	}
	if err := live("etcd-init", func() (string, error) {
		return "called (effect checked by verify-etcd)", h.EtcdInit(ctx, producer)
	}); err != nil {
		return node.NodeSet{}, steps, nil, err
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
