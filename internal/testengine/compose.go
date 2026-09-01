package testengine

import (
	"context"
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
	// forkWait bounds the wait for a successor to seal the first post-fork
	// block. The profile's fork height and block time decide the real figure;
	// this is the ceiling.
	forkWait = 180 * time.Second
)

// overlayFile is where a declared genesis overlay is written for the genesis
// step, which reads overlays from a file.
const overlayFile = "env-genesis-overlay.json"

// defaultKeysDir is the key set a declaration that names none composes from.
const defaultKeysDir = "keys/preset"

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
	if k := spec.EnvKeys; k != nil {
		if keysSource == "" {
			keysSource = k.Source
		}
		if keysDir == "" {
			keysDir = expand(k.Ref)
		}
	}
	if keysDir == "" {
		keysDir = defaultKeysDir
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
		return composition{handoff: &upgrade.HandoffInputs{
			ProfilePath:    expand(u.Profile),
			Template:       expand(u.Template),
			PresetDir:      keysDir,
			FromBinary:     expand(spec.Chain.Binaries[dsl.RoleProducer]),
			ToBinary:       expand(spec.Chain.Binaries[dsl.RoleValidator]),
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

	var validators, endpoints int
	var syncMode string
	if inlineTopo == nil {
		validators, endpoints, syncMode, err = topologyOf(spec.Topology)
		if err != nil {
			return composition{}, err
		}
		if in.Validators > 0 {
			validators = in.Validators
		}
		if validators <= 0 {
			validators = suiteDefaultValidators
		}
	}
	launch := launchSets(spec.EnvLaunch)
	launch = append(launch, in.LaunchOpts...)
	if in.NetworkID != 0 {
		launch = append(launch, fmt.Sprintf("%s=%d", nodeconfig.KeyNetworkID, in.NetworkID))
	}
	up := &chainsetup.NetUpIn{
		DataDir: in.DataDir, Stage: chainsetup.UpStart,
		Chain: chain, Binary: binary, KeysDir: keysDir, KeysSource: keysSource,
		Validators: validators, Endpoints: endpoints, EndpointSyncMode: syncMode,
		Topology: inlineTopo, Binaries: resolvedBins,
		Server: in.Server, Docker: in.Docker,
		ChainID:     in.ChainID,
		GenesisSet:  hardforkSets(spec.Hardforks),
		OverlayPath: overlayPath,
		LaunchSet:   launch,
		ConfigSet:   spec.EnvConfig,
	}
	return composition{up: up}, nil
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
			case "index":
				n, ferr := countOf("nodes[].index", v)
				if ferr != nil {
					return nil, nil, "", ferr
				}
				entry.Index = n
			default:
				return nil, nil, "", fmt.Errorf("topology.nodes[%d].%s is not a key the composer knows (role, binary, sync, bootnode, index)", i, k)
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
	topoSyncMode     = "syncMode"
	topoSyncModeSnak = "sync_mode"
)

// topologyOf reads the node counts a declaration gives: validators (or bp),
// endpoints (or en), and the endpoints' sync mode. A key it does not know is
// an error rather than a silently ignored intention.
func topologyOf(t map[string]any) (validators, endpoints int, syncMode string, err error) {
	for k, v := range t {
		switch k {
		case topoValidators, topoBP:
			validators, err = countOf(k, v)
		case topoEndpoints, topoEN:
			endpoints, err = countOf(k, v)
		case topoSyncMode, topoSyncModeSnak:
			s, ok := v.(string)
			if !ok {
				err = fmt.Errorf("topology.%s must be a string", k)
			}
			syncMode = s
		default:
			err = fmt.Errorf("topology.%s is not a key the composer knows (validators|bp, endpoints|en, syncMode)", k)
		}
		if err != nil {
			return 0, 0, "", err
		}
	}
	return validators, endpoints, syncMode, nil
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

// launchSets renders declared launch knobs as the launchopts step's --set
// arguments: a boolean knob travels as a bare key.
func launchSets(kvs []dsl.LaunchKV) []string {
	out := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		if kv.Value == "" || kv.Value == "true" {
			out = append(out, kv.Key)
			continue
		}
		out = append(out, kv.Key+"="+kv.Value)
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
	path := filepath.Join(dataDir, overlayFile)
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
	if err := h.ComposePlan(base); err != nil {
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
