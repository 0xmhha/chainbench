package chainsetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/preset"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Composition steps: keys, place, genesis, config, build, deploy.
//
// Each reads the accumulated state, fails fast when a prerequisite step has not
// run, performs its one concern through the same core packages the engine uses,
// and records itself in the step table. Lifecycle steps (init, start, stop, ...)
// live in steps_lifecycle.go.
//
// This file holds what those steps share: provisioning the target, the genesis
// artifacts they read back, and the table saying which step needs which. The
// steps themselves are one file each — steps_keys.go, steps_place.go,
// steps_genesis.go, steps_config.go — so a reader after one step is not reading
// the other four. They stay in this package: the split is for the reader, and
// the boundary the compiler enforces has not moved.

// Placement bounds applied when the caller did not supply a server set.
// The port bands themselves live in serverset (built-in defaults, or the
// operator's gitignored server set) — never here.
const (
	portBandSize              = 100
	minValidatorsForPlacement = 1
)

// The kinds of failure the deploy stage has.
//
// Both are about a launch input that is not what this workspace expects, and
// the difference is the one that matters to whoever has to fix it: a file that
// is not there needs an earlier step re-run, and a file that is there but is
// somebody else's needs a decision about which is right.
var (
	// errDeployInputMissing: a launch input is not on the target.
	errDeployInputMissing = errors.New("a launch input is missing on the target")
	// errDeployInputForeign: a launch input is there and is not the one this
	// workspace built.
	errDeployInputForeign = errors.New("a launch input on the target is not this workspace's")
)

func (w *Workspace) Provision(ctx context.Context) (StepOut, error) {
	if err := w.require("deploy"); err != nil {
		return StepOut{}, err
	}
	present, shipped := 0, 0
	err := w.eachMachine(func(t *resource.Access, nodes []node.Record) error {
		check := func(path string) error {
			exists, err := t.Files.Exists(ctx, path)
			if err != nil {
				return err
			}
			if !exists {
				return lifecycle.Mark(errDeployInputMissing,
					fmt.Errorf("chainsetup: provision: %s missing — run the genesis/config steps first", path))
			}
			// Present is not the same as ours. A genesis someone edited, or a
			// config left by a previous composition, is present and would be
			// launched from.
			if want, known := w.state.LaunchInputs[path]; known {
				have, err := t.Files.Checksum(ctx, path)
				if err != nil {
					return err
				}
				if have != want {
					return lifecycle.Mark(errDeployInputForeign,
						fmt.Errorf("chainsetup: provision: %s is not the file this workspace built "+
							"(built %s, found %s) — something else wrote it; re-run the step that makes it "+
							"(`chain genesis` or `chain config`) to put yours back", path, short(want), short(have)))
				}
			}
			present++
			return nil
		}
		if err := check(w.state.GenesisPath); err != nil {
			return err
		}
		for _, ns := range nodes {
			if err := check(ns.ConfigPath); err != nil {
				return err
			}
		}
		n, err := w.shipIdentities(ctx, t, nodes)
		shipped += n
		return err
	})
	if err != nil {
		return StepOut{}, err
	}
	detail := fmt.Sprintf("%d launch input(s) present on the target (reused, not rewritten)", present)
	if shipped > 0 {
		detail += fmt.Sprintf(", %d identity file(s) shipped to %s", shipped, w.keysBase())
	}
	w.markStep("deploy", detail)
	// Which of the two the deploy did, said by the step that counted it.
	// Nothing outside can: whether anything was shipped is the difference
	// between a local target and a remote one, and it is counted here.
	at := lifecycle.ChainDeployNodesVerifiedLocal
	if shipped > 0 {
		at = lifecycle.ChainDeployNodesShippedRemote
	}
	return StepOut{Detail: detail, Passed: []lifecycle.Status{at}}, nil
}

// shipIdentities uploads each node's identity files — the devp2p nodekey, the
// validator keystore, and the shared password — from the local key set to
// keysBase on a remote target, upload-if-absent like the rest of filestore.
// The rendered config and the launch argv point at keysBase, so without this
// a remote node would look for its keys on the operator's resource. A local
// target ships nothing: keysBase is the key set itself.
func (w *Workspace) shipIdentities(ctx context.Context, t *resource.Access, nodes []node.Record) (int, error) {
	if !t.Spec.IsRemote() {
		return 0, nil
	}
	shipped := 0
	put := func(src, dst string, mode fs.FileMode) error {
		b, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // e.g. an endpoint node with no keystore
			}
			return err
		}
		exists, err := t.Files.Exists(ctx, dst)
		if err != nil {
			return err
		}
		if exists {
			have, err := t.Files.Checksum(ctx, dst)
			if err != nil {
				return err
			}
			if have == filestore.Hash(b) {
				return nil // identical content already on the target: not re-sent
			}
			// A stale key file under the same name is not the one we mean; ship
			// the current content over it rather than launch with the wrong key.
		}
		if err := t.Files.Write(ctx, dst, b, mode); err != nil {
			return err
		}
		shipped++
		return nil
	}
	base := w.keysBase()
	if err := put(filepath.Join(w.state.KeysDir, "password"), filepath.Join(base, "password"), 0o600); err != nil {
		return shipped, fmt.Errorf("chainsetup: provision: password: %w", err)
	}
	for _, ns := range nodes {
		src := filepath.Join(w.state.KeysDir, fmt.Sprintf("node%d", ns.Index))
		dst := filepath.Join(base, fmt.Sprintf("node%d", ns.Index))
		if err := put(filepath.Join(src, "nodekey"), filepath.Join(dst, "nodekey"), 0o600); err != nil {
			return shipped, fmt.Errorf("chainsetup: provision: node%d nodekey: %w", ns.Index, err)
		}
		entries, err := os.ReadDir(filepath.Join(src, "keystore"))
		if err != nil {
			continue // no keystore for this node
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if err := put(filepath.Join(src, "keystore", e.Name()),
				filepath.Join(dst, "keystore", e.Name()), 0o600); err != nil {
				return shipped, fmt.Errorf("chainsetup: provision: node%d keystore: %w", ns.Index, err)
			}
		}
	}
	return shipped, nil
}

// ParseOverrides maps "key=value" strings (bare key for booleans) onto typed
// launchopt overrides. Whether a key exists for the target binary is checked
// at assembly by the Builder.
func ParseOverrides(sets []string) ([]nodeconfig.Override, error) {
	out := make([]nodeconfig.Override, 0, len(sets))
	for _, s := range sets {
		k, v, _ := strings.Cut(s, "=")
		if k == "" {
			return nil, lifecycle.Mark(errBuildBadOption,
				fmt.Errorf("chainsetup: bad --set %q (want key=value or a bare boolean key)", s))
		}
		out = append(out, nodeconfig.Override{Key: nodeconfig.OptionKey(k), Value: v})
	}
	return out, nil
}

// nodeHost is the address a composed node is reachable at: the one the
// allocator recorded, falling back to this machine for a plan that predates
// per-node hosts.
func nodeHost(ns node.Record) string {
	if ns.Host != "" {
		return ns.Host
	}
	return localHost
}

// Netmap reads the workspace's node table as a placement map, so the peer
// policy and the address lookups run off one representation. The host is the
// node's own recorded address, which spread across a set is not this resource.
func (w *Workspace) Netmap() (*node.Map, error) {
	placements := make([]node.Placement, 0, len(w.state.Nodes))
	ordinals := map[node.Role]int{}
	for _, ns := range w.state.Nodes {
		role, err := node.NormalizeRole(ns.Role)
		if err != nil {
			return nil, fmt.Errorf("node%d: %w", ns.Index, err)
		}
		ordinals[role]++
		placements = append(placements, node.Placement{
			Index:   ns.Index,
			Label:   ns.NodeLabel(),
			Role:    role,
			Ord:     ordinals[role],
			Host:    nodeHost(ns),
			Ports:   ns.Endpoints,
			DataDir: ns.DataDir,
		})
	}
	return node.NewMap(placements)
}

// netmapRequests turns the composed node list into placement requests. Only the
// role travels: position comes from the order, which is also the node's
// identity.
func netmapRequests(reqs []node.LaunchReq) []resource.Request {
	out := make([]resource.Request, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, resource.Request{Role: r.Role})
	}
	return out
}

// genesisBytes produces the genesis this composition would write, without
// writing it. Rendering is separated from writing so a caller can ask what a
// composition WOULD produce — reuse-if-matching has to know whether the inputs
// changed before it touches the target, and asking afterwards means a refusal
// that has already replaced the running network's genesis.
//
// A finished genesis is read on its own machine and used verbatim — no template
// build, no overrides. It must be valid JSON, and (for a family that carries its
// validator set in the genesis) its validators must be the composed key set.
func (w *Workspace) genesisBytes(ctx context.Context, p registry.ChainPlugin, opts GenesisOpts) ([]byte, genesis.Artifacts, error) {
	if opts.Existing != "" {
		b, rerr := w.readInputRef(ctx, node.Record{}, opts.Existing, resource.PurposeGenesis)
		if rerr != nil {
			return nil, genesis.Artifacts{}, lifecycle.Mark(errGenesisExistingInvalid,
				fmt.Errorf("chainsetup: genesis: read existing %q: %w", opts.Existing, rerr))
		}
		if !json.Valid(b) {
			return nil, genesis.Artifacts{}, lifecycle.Mark(errGenesisExistingInvalid,
				fmt.Errorf("chainsetup: genesis: existing genesis %q is not valid JSON", opts.Existing))
		}
		if err := w.verifyExistingGenesisKeys(p, b, opts.Existing); err != nil {
			return nil, genesis.Artifacts{}, err
		}
		return b, genesis.Artifacts{}, nil
	}
	art, err := w.genesisArtifacts(ctx, p, opts)
	if err != nil {
		return nil, genesis.Artifacts{}, err
	}
	return art.Genesis, art, nil
}

// genesisArtifacts builds the genesis through the one composition every surface
// uses. The wemix source runs the chain binary, so the request also carries the
// placement: the governance config names the producer by host and p2p port.
func (w *Workspace) genesisArtifacts(ctx context.Context, p registry.ChainPlugin, opts GenesisOpts) (genesis.Artifacts, error) {
	// The placement always travels with the request: a family whose genesis
	// names the producer's address reads it, and one whose genesis carries the
	// validator set ignores it. Branching here on the family was the second
	// place the wemix path had to be special-cased.
	placed, err := w.Netmap()
	if err != nil {
		return genesis.Artifacts{}, fmt.Errorf("chainsetup: genesis: %w", err)
	}
	req := genesis.Request{Validators: w.state.BPCount, Nodes: placed}
	cfg := genesis.Config{
		KeysDir:         w.state.KeysDir,
		Binary:          w.state.Binary,
		ChainID:         opts.ChainID,
		ConfigOverrides: opts.Overrides,
		Overlay:         opts.Overlay,
	}
	// A family whose genesis its own binary writes (wemix) runs that binary. On
	// a remote target the binary lives there, not here, so stage the generator's
	// inputs on the target and run it over the same access init/start use. A
	// family whose genesis is in-process ignores these.
	if w.state.Target.IsRemote() {
		boot, ok := firstProducer(w.state.Nodes)
		if !ok {
			return genesis.Artifacts{}, lifecycle.Mark(errGenesisTargetUnable,
				fmt.Errorf("chainsetup: genesis: no producer to generate the genesis on"))
		}
		access, err := w.machineFor(boot)
		if err != nil {
			return genesis.Artifacts{}, fmt.Errorf("chainsetup: genesis: %w", err)
		}
		cmdr, ok := access.Driver.(process.Commander)
		if !ok {
			return genesis.Artifacts{}, lifecycle.Mark(errGenesisTargetUnable,
				fmt.Errorf("chainsetup: genesis: the target cannot run a command, so a binary-written genesis cannot be generated there"))
		}
		cfg.Files = access.Files
		cfg.WorkDir = path.Join(access.DataRoot, genesisWorkDir)
		cfg.Runner = commanderRunner(cmdr)
	}
	return genesis.Compose(ctx, p, req, cfg)
}

// genesisWorkDir is where a binary-written genesis stages its config, template,
// and output on the target, under the data root.
const genesisWorkDir = "genesis-work"

// firstProducer returns the first block-producing node in the table — the node
// a binary-written genesis is generated on and the boot phase launches alone.
func firstProducer(nodes []node.Record) (node.Record, bool) {
	for _, ns := range nodes {
		if node.Is(node.Role(ns.Role), node.RoleBP) {
			return ns, true
		}
	}
	return node.Record{}, false
}

// commanderRunner adapts a process.Commander (the local or remote driver's
// arbitrary-command capability) into the runner a binary-written genesis and the
// poa bootstrap take, so both run on the same transport as init and start. The
// binary and its args are shell-quoted into one command line.
// commanderRunner is process.ShellRunner in the genesis package's spelling. The
// adapter itself lives in process, beside Commander, because app needs the same
// one for the handoff's bootstrap on a target.
func commanderRunner(c process.Commander) genesis.CommandRunner {
	return process.ShellRunner(c)
}

// peerPlan is what every per-node rendering needs beyond the record: the key
// set (identity and public keys), the placement, the validated peering, and a
// public-key lookup by index. Config, launchopts and start all render from
// the same four, so they are gathered once.
func (w *Workspace) peerPlan(p registry.ChainPlugin) (preset.Key, *node.Map, node.Peering, func(int) (string, bool), error) {
	// With accounts, for the same reason start loads them: this renders each
	// node's config, and a producer's config names the account it unlocks.
	keys, err := preset.LoadKeyPresetWithAccounts(w.state.KeysDir)
	if err != nil {
		return preset.Key{}, nil, "", nil, err
	}
	placed, err := w.Netmap()
	if err != nil {
		return preset.Key{}, nil, "", nil, err
	}
	peering, err := node.ParsePeering(w.state.Peering)
	if err != nil {
		return preset.Key{}, nil, "", nil, err
	}
	if err := peering.Validate(placed, p.Family().SupportsRole); err != nil {
		return preset.Key{}, nil, "", nil, err
	}
	// The peer's own recorded address: spread across a set each node lives on
	// a different host, and a static-node list pointing at this machine would
	// leave every node unable to find its peers. Keys reach the composition
	// as inputs — the node module joins them to placements.
	pubkey := func(index int) (string, bool) {
		nk, ok := keys.Node(index)
		if !ok {
			return "", false
		}
		return nk.PublicKey, true
	}
	return keys, placed, peering, pubkey, nil
}

// recordInput remembers what a launch input hashed to when this workspace wrote
// it, so deploy can tell the file it built from one that merely occupies the
// same path.
func (w *Workspace) recordInput(path string, content []byte) {
	if w.state.LaunchInputs == nil {
		w.state.LaunchInputs = map[string]string{}
	}
	w.state.LaunchInputs[path] = filestore.Hash(content)
}

// short renders a hash the way a reader compares two of them: enough to tell
// them apart, not so much that the message wraps.
func short(hash string) string {
	if i := strings.IndexByte(hash, ':'); i >= 0 && len(hash) > i+13 {
		return hash[:i+13]
	}
	return hash
}

// composeNeeds is the composition's resolution order, declared once.
//
// It was a convention before: each step hand-rolled a check on whatever state
// field it happened to need and wrote its own "run X first" message. Three
// things went wrong with that. The checks disagreed about what a step needs —
// genesis looked at the validator count and never at the key set, though it
// cannot build extraData without one. The messages named different steps for
// the same missing prerequisite. And the order existed nowhere a reader could
// see it, so N9's question ("what has to resolve before what") could only be
// answered by reading six functions.
//
// The order is NOT keyring → netmap, which is how the worklist recorded it.
// `keys` takes its node count from the placement, so `place` runs first and
// `chain up` has always run them that way; the note was written from the
// intended design rather than from the code.
//
// genesis needs place and NOT keys, which is the second thing writing this down
// corrected. It hands the key directory to the family and the family decides:
// one that seals extraData from the validator keys reads it, one that
// substitutes a supplied template never opens it. Requiring a key set here
// broke composing an external chain from its own template, which is a thing
// that worked, so the dependency belongs to the family and not to the step.
//
// Each entry lists what a step reaches for DIRECTLY. deploy needs a key set as
// much as config does, and gets it by way of config rather than by claiming it
// here, so a change in what config needs does not have to be copied.
//
// enode is absent because it produces nothing and marks no step: it derives a
// view from place and keys, and verbs_enode.go asks for both directly.
var composeNeeds = map[string][]string{
	"place":   {"new"},
	"keys":    {"new"},
	"genesis": {"place"},
	"config":  {"place", "keys"},
	"build":   {"place", "keys"},
	"deploy":  {"place", "genesis", "config"},
	// init and start were in the order and not in this table, so the rule that
	// a datadir is initialized before a node launches lived only in the order
	// upSteps happens to iterate. Running the steps by hand, or resuming from
	// one, could launch a node over a datadir no genesis had ever reached.
	"init":  {"deploy"},
	"start": {"init"},
}

// require reports whether every step that has to resolve before step has run,
// naming the first one that has not.
//
// It reads the recorded steps rather than the state fields they leave behind.
// A field can be non-empty because something else filled it, and a step that
// half-ran leaves exactly that: state that looks composed and was not.
func (w *Workspace) require(step string) error {
	for _, need := range composeNeeds[step] {
		if _, done := w.state.Steps[need]; !done {
			return lifecycle.Mark(errOpPrecondition,
				fmt.Errorf("chainsetup: %s: %s has not run — run `chain %s` first", step, need, need))
		}
	}
	return nil
}

// DeployFailure is which of the deploy stage's two failures this error is.
//
// The default is the shipping itself: a file store that will not read the local
// key or will not write it to the machine. That is the store's refusal, and the
// two named here are about what is on the target rather than about getting
// there.
func DeployFailure(err error) lifecycle.Status {
	switch {
	case errors.Is(err, errDeployInputMissing):
		return lifecycle.ChainDeployNodesFailInputMissing
	case errors.Is(err, errDeployInputForeign):
		return lifecycle.ChainDeployNodesFailInputForeign
	}
	return lifecycle.FailStageUnclassified
}
