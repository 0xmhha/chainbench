package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/genesis"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/preset"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The genesis step: the document every node initializes from.
//
// It builds from the chain's template, applies the declaration's overrides and
// overlay, and schedules a fork when one is declared — a fork travels either in
// the genesis or in the node config, and which one is the chain's answer rather
// than this step's. The network's advertised capabilities are computed here for
// the same reason: what a network can answer for is decided by the genesis it
// was given.

type GenesisOpts struct {
	// ChainID, when non-zero, overrides the manifest chain id.
	ChainID int64
	// Overrides sets bare keys in the genesis `config` object, e.g.
	// {"bohoBlock": "10"} to move a fork off genesis. The fork ordering of the
	// result is validated, so a bad delayed-fork request fails here rather than
	// at node boot.
	Overrides map[string]string
	// Overlay is a genesis JSON fragment deep-merged into the built genesis
	// (extra alloc accounts, config bits). Fork ordering is re-validated after
	// the merge.
	Overlay []byte
	// Capabilities are advertised alongside the network so capability-gated
	// cases run — an overlay declares what it enables.
	Capabilities []string
	// HaltsAt is the block this genesis makes the network stop one short of, 0
	// for one that keeps producing. It is recorded so the readiness gate reads
	// a chain that is meant to stop as ready instead of waiting it out.
	HaltsAt int64
	// Existing is a reference to a finished genesis file used verbatim (genesis
	// mode "existing"): the file is read on its machine and written to each
	// target, instead of building one from a template. Overrides/Overlay do not
	// apply to it. Empty builds as usual.
	Existing string
	// Fork, when set, schedules a hardfork whose consensus configuration comes
	// from another chain's own genesis. See GenesisFork.
	Fork *GenesisFork
	// Variants are extra genesis documents, one per binary name, each a JSON
	// fragment deep-merged onto the built genesis. The nodes running that
	// binary initialize from the result; every other node keeps the network's.
	//
	// It exists because a network can run two binaries that do not accept the
	// same genesis. A handoff across a fork is the case: the successor needs
	// fork settings the predecessor may refuse, and whether it refuses them is
	// a fact about those two builds rather than something a composer can
	// assume. Merged onto the BUILT genesis, not rebuilt from the template, so
	// the second document is demonstrably the first plus what that side needs —
	// and a family whose genesis its own binary generates is not generated
	// twice.
	Variants map[string][]byte
}

// GenesisFork schedules a hardfork this network crosses, taking the fork's
// consensus configuration from the chain that seals after it.
//
// The section is not a constant and cannot be written down. It is the to-chain's
// own genesis, built from its own template with the real post-fork validator
// set, with the fork's config section lifted out of it. A literal overlay could
// carry today's values and would be wrong the moment the key set changed — the
// validators, their BLS keys and the RLP extra-data that encodes them all come
// from the ring.
type GenesisFork struct {
	// Name is the fork ("croissant"); At is the block it activates on. The
	// activation key follows the "<name>Block" convention the chains use.
	Name string `json:"name"`
	At   int64  `json:"at"`
	// Binary names the binary that seals after the fork. Its chain supplies the
	// section, and the nodes running it are the post-fork validators.
	//
	// By binary rather than by role: before the fork these nodes are endpoints,
	// and after it they produce. Which side of the fork a node is on is which
	// build it runs, which is the same question binaryFor, genesisFor and
	// pluginFor each answer.
	Binary string `json:"binary"`
	// Carrier is which file carries the fork's configuration to the nodes that
	// need it. Empty means ForkInGenesis.
	Carrier ForkCarrier `json:"carrier,omitempty"`
	// Restart says every node moves to Binary at the fork, rather than some of
	// them already running it.
	//
	// The two shapes differ in who is standing where. In the ordinary one the
	// successors are up from the start, syncing, and take over when the fork
	// arrives; in a restart there is one build at a time and every node crosses
	// together, the producers still producing afterwards. So the post-fork
	// validators are read differently: by which build a node runs, or — when no
	// node runs it yet — by which nodes produce.
	Restart bool `json:"restart,omitempty"`
}

// ForkCarrier is which file a fork's configuration travels in.
//
// Both routes end with the same chain: the pre-fork build stops sealing at the
// fork block and the post-fork build takes over. They differ in which file the
// second build reads its extra configuration from, and therefore in which file
// the network has two of.
type ForkCarrier string

const (
	// ForkInGenesis writes the fork's section into the genesis. The network
	// then has two genesis documents — the post-fork build's is the other one
	// plus its section — and one config per node.
	//
	// The cheaper of the two: a genesis document is JSON, so it is written the
	// way it is read, and a build whose config has no field for the section
	// ignores it rather than refusing it.
	ForkInGenesis ForkCarrier = "genesis"
	// ForkInConfig leaves the fork's section out of the genesis and writes the
	// whole genesis, section included, into the config of the nodes running the
	// post-fork build. The network then has one genesis document and two shapes
	// of config.
	//
	// The costlier of the two. A config file is read by a TOML decoder that has
	// none of the genesis document's conversions and refuses a key it cannot
	// place, so the genesis has to be respelled (genesis.ConfigTOML) and the chain
	// has to declare the keys its binary has no field for
	// (registry.GenesisSpec.ConfigOmit). In exchange, every node initializes
	// from one genesis.
	ForkInConfig ForkCarrier = "config"
)

// carrier is the route this fork takes.
//
// The genesis is the default and the cheaper route. A restart takes the config,
// and not as a preference: its nodes were initialized by a build that did not
// know the fork, so the chain config stored in their databases does not mention
// it — and that stored config is what every later start reads. A genesis
// document is read only into an empty database.
//
// Measured: four nodes initialized by a pre-boho gstable, relaunched on a build
// that knows boho, ran past block 200 with bohoBlock absent from the stored
// config and govMinter still v1. The fork never happened. The config file is
// read on every launch, which is the route that reaches an initialized node.
func (f GenesisFork) carrier() ForkCarrier {
	if f.Carrier != "" {
		return f.Carrier
	}
	if f.Restart {
		return ForkInConfig
	}
	return ForkInGenesis
}

// Genesis builds the genesis from the key set's validator material and writes
// it to the target's data root (upload-if-absent semantics are the provision
// step's concern; genesis always reflects the current inputs).
func (w *Workspace) Genesis(ctx context.Context, opts GenesisOpts) (string, error) {
	// The rule belongs to the operation, not to the one builder that happened to
	// construct these options. genesisOpts checks it too, and every caller today
	// goes through genesisOpts — but this method is exported and takes the
	// options directly, so "a finished genesis is never quietly changed" held
	// only as long as no one assembled a GenesisOpts by hand. An invariant that
	// depends on which door you came in is not an invariant.
	if err := opts.checkExistingIsUnchanged(); err != nil {
		return "", err
	}
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if err := w.require("genesis"); err != nil {
		return "", err
	}
	// A family whose genesis its binary writes takes a different source: the
	// generic dispatch builds a genesis by substituting a template, and for
	// wemix that produces a file that initializes cleanly and runs the wrong
	// consensus.
	var (
		art genesis.Artifacts
		gen []byte
	)
	gen, art, err = w.genesisBytes(ctx, p, opts)
	if err != nil {
		return "", err
	}
	// Every machine gets the genesis (and its by-products): each node's init
	// reads it locally, and spread across a set "locally" is that node's server.
	// The genesis is a generated file, so it sits under the composition's
	// runtime directory when isolated (flat otherwise). The path is derived the
	// one way, per machine, so a set writes each server the same relative path.
	var forkConfigs map[string][]byte
	w.state.Fork = nil
	if opts.Fork != nil {
		if gen, forkConfigs, err = w.applyFork(gen, *opts.Fork); err != nil {
			return "", err
		}
		art.Genesis = gen
		// Recorded because crossing the fork is a later step's work: it has to
		// know which block the pre-fork build stops at and which nodes take
		// over, and the request that said so is gone by then.
		fork := *opts.Fork
		w.state.Fork = &fork
	}
	lay, err := w.layout()
	if err != nil {
		return "", err
	}
	path := lay.GenesisPath()
	err = w.eachMachine(func(t *resource.Access, _ []node.Record) error {
		ml := lay
		ml.Root = t.DataRoot
		p := ml.GenesisPath()
		if err := t.Files.Write(ctx, p, gen, 0o644); err != nil {
			return fmt.Errorf("chainsetup: genesis: write: %w", err)
		}
		w.recordInput(p, gen)
		// The step's by-products go beside the genesis: a wemix bring-up
		// reads its governance config back during deploy-governance.
		for name, content := range art.Extra {
			extra := filepath.Join(filepath.Dir(p), name)
			if err := t.Files.Write(ctx, extra, content, 0o644); err != nil {
				return fmt.Errorf("chainsetup: genesis: write %s: %w", name, err)
			}
			w.recordInput(extra, content)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	w.state.GenesisPath = path
	if err := w.writeGenesisVariants(ctx, lay, gen, opts.Variants); err != nil {
		return "", err
	}
	if err := w.writeGenesisConfigs(ctx, lay, forkConfigs); err != nil {
		return "", err
	}
	w.state.Capabilities = networkCapabilities(p.Manifest(), p.GenesisTemplate(), opts)
	w.state.HaltsAt = opts.HaltsAt

	detail := fmt.Sprintf("%d bytes at %s, %d validator(s)", len(gen), path, w.state.BPCount)
	if opts.ChainID != 0 {
		detail += fmt.Sprintf(", chain id %d (override)", opts.ChainID)
	}
	if len(opts.Overrides) > 0 {
		detail += fmt.Sprintf(", %d config override(s)", len(opts.Overrides))
	}
	if len(opts.Overlay) > 0 {
		detail += ", overlay merged"
	}
	if n := len(opts.Variants); n > 0 {
		detail += fmt.Sprintf(", %d binary-specific genesis", n)
	}
	if opts.Fork != nil {
		detail += fmt.Sprintf(", %s fork at %d carried by %s", opts.Fork.Name, opts.Fork.At, opts.Fork.carrier())
	}
	w.markStep("genesis", detail)
	return detail, nil
}

// applyFork schedules the fork on the built genesis, taking its consensus
// configuration from the chain that seals after it.
//
// Three steps, all of them existing pieces: build the to-chain's genesis from
// the key set with the post-fork validators, lift its fork section out, and set
// that section plus the activation block on this genesis. The fork order is
// re-validated, so a fork scheduled before one it must follow fails here rather
// than at the boot of every node.
//
// Where the section lands is the carrier's choice. Both routes put the
// activation block on the network's genesis — that is what makes the pre-fork
// build stop sealing, and every node needs it. The section itself either goes
// beside it (ForkInGenesis) or into the config of the nodes running the
// post-fork build (ForkInConfig), and the second return is that config, by
// binary, empty for the genesis route.
func (w *Workspace) applyFork(base []byte, f GenesisFork) ([]byte, map[string][]byte, error) {
	if f.Name == "" {
		return nil, nil, fmt.Errorf("chainsetup: genesis: a fork needs a name")
	}
	withBlock, err := genesis.SetConfigSection(base, f.Name+"Block", json.RawMessage(strconv.FormatInt(f.At, 10)))
	if err != nil {
		return nil, nil, fmt.Errorf("chainsetup: genesis: set %sBlock: %w", f.Name, err)
	}
	// A restart's fork is a block, and what happens at it is the post-fork
	// build's to know. Whatever configuration the fork needs is in the genesis
	// already — the declaration puts it there like any other genesis content —
	// so there is no second chain to lift a section out of. The block is
	// written here so the declaration, not an overlay, is the one place that
	// says when.
	//
	// It still has to be carried, because by the time it matters the nodes are
	// initialized and nothing reads a genesis document into a database that is
	// not empty.
	if f.Restart {
		if err := genesis.ValidateForks(withBlock); err != nil {
			return nil, nil, fmt.Errorf("chainsetup: genesis: with the %q fork at %d: %w", f.Name, f.At, err)
		}
		if f.carrier() == ForkInGenesis {
			return withBlock, nil, nil
		}
		p, err := w.plugin()
		if err != nil {
			return nil, nil, err
		}
		toml, err := genesis.ConfigTOML(withBlock, forkConfigTable, p.Manifest().Genesis.ConfigOmit)
		if err != nil {
			return nil, nil, fmt.Errorf("chainsetup: genesis: render the %q fork for binary %q's config: %w", f.Name, f.Binary, err)
		}
		return withBlock, map[string][]byte{f.Binary: toml}, nil
	}
	section, err := w.forkSection(f)
	if err != nil {
		return nil, nil, err
	}
	full, err := genesis.SetConfigSection(withBlock, f.Name, section)
	if err != nil {
		return nil, nil, fmt.Errorf("chainsetup: genesis: set the %q section: %w", f.Name, err)
	}
	// Validated on the document that holds everything, so an ordering the
	// post-fork build would refuse fails here on either route.
	if err := genesis.ValidateForks(full); err != nil {
		return nil, nil, fmt.Errorf("chainsetup: genesis: with the %q fork at %d: %w", f.Name, f.At, err)
	}
	if f.carrier() == ForkInGenesis {
		return full, nil, nil
	}
	p, err := w.plugin()
	if err != nil {
		return nil, nil, err
	}
	toml, err := genesis.ConfigTOML(full, forkConfigTable, p.Manifest().Genesis.ConfigOmit)
	if err != nil {
		return nil, nil, fmt.Errorf("chainsetup: genesis: render the %q fork for binary %q's config: %w", f.Name, f.Binary, err)
	}
	return withBlock, map[string][]byte{f.Binary: toml}, nil
}

// forkConfigTable is where a geth-family binary reads a genesis it is given in
// its config file.
const forkConfigTable = "Eth.Genesis"

// forkSection is the fork's consensus configuration, read out of the genesis of
// the chain that seals after it.
//
// The section is not a constant and cannot be written down: the validators,
// their BLS keys and the RLP extra-data that encodes them all come from the
// ring, so it is built from the ring every time.
func (w *Workspace) forkSection(f GenesisFork) (json.RawMessage, error) {
	id := w.state.BinaryChains[f.Binary]
	if id == "" {
		return nil, fmt.Errorf("chainsetup: genesis: the %q fork seals on binary %q, which names no chain of its own — the fork's configuration comes from that chain's genesis", f.Name, f.Binary)
	}
	p, err := external.ResolveChain(id, "", "")
	if err != nil {
		return nil, fmt.Errorf("chainsetup: genesis: fork chain %q: %w", id, err)
	}
	indices, err := w.forkValidators(f)
	if err != nil {
		return nil, err
	}
	preset, err := preset.LoadKeyPresetWithAccounts(w.state.KeysDir)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: genesis: %w", err)
	}
	net, err := preset.NetworkForNodes(indices)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: genesis: the %q fork's validators: %w", f.Name, err)
	}
	toGen, err := genesis.Build(p, genesis.Inputs{
		Validators: net.Validators, BLSKeys: net.BLSKeys, ExtraData: net.ExtraData,
		Members: net.Members, Alloc: net.Alloc,
	})
	if err != nil {
		return nil, fmt.Errorf("chainsetup: genesis: build the %q genesis the fork hands over to: %w", id, err)
	}
	section, err := genesis.ExtractConfigSection(toGen, f.Name)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: genesis: read the %q section from the %q genesis: %w", f.Name, id, err)
	}
	if len(section) == 0 {
		return nil, fmt.Errorf("chainsetup: genesis: the %q genesis has no %q section to hand over", id, f.Name)
	}
	return section, nil
}

// forkValidators is which of this network's nodes seal after the fork, by node
// index: the ones running the build that takes over.
//
// Before the fork they are endpoints and after it they produce, which is what
// binaryFor, genesisFor and pluginFor each ask in their turn. Only a handover
// asks this at all — a restart hands production to nobody, because the nodes
// that produced before the fork go on producing after it.
func (w *Workspace) forkValidators(f GenesisFork) ([]int, error) {
	var indices []int
	for _, ns := range w.state.Nodes {
		if ns.Binary == f.Binary {
			indices = append(indices, ns.Index)
		}
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("chainsetup: genesis: no node runs binary %q, so the %q fork would hand over to nobody", f.Binary, f.Name)
	}
	return indices, nil
}

// writeGenesisConfigs writes the per-binary genesis configs — the whole
// genesis in the spelling a config file takes — and records where they landed,
// so the config step can hand each node the one its build reads.
//
// Beside the genesis rather than inside each node's config file, for the same
// reason a genesis variant is its own document: one file per binary, written
// once, that a reader can open and compare against the network's genesis.
func (w *Workspace) writeGenesisConfigs(ctx context.Context, lay node.Layout, configs map[string][]byte) error {
	if len(configs) == 0 {
		w.state.GenesisConfigPaths = nil
		return nil
	}
	paths := make(map[string]string, len(configs))
	for _, name := range slices.Sorted(maps.Keys(configs)) {
		if w.state.Binaries[name] == "" {
			return fmt.Errorf("chainsetup: genesis: a genesis config is declared for binary %q, which no node runs — name one of the declared binaries", name)
		}
		body := configs[name]
		path := lay.GenesisConfigPath(name)
		err := w.eachMachine(func(t *resource.Access, _ []node.Record) error {
			ml := lay
			ml.Root = t.DataRoot
			p := ml.GenesisConfigPath(name)
			if werr := t.Files.Write(ctx, p, body, 0o644); werr != nil {
				return fmt.Errorf("chainsetup: genesis: write %s: %w", p, werr)
			}
			w.recordInput(p, body)
			return nil
		})
		if err != nil {
			return err
		}
		paths[name] = path
	}
	w.state.GenesisConfigPaths = paths
	return nil
}

// writeGenesisVariants builds and writes the per-binary genesis documents, each
// the built genesis plus what that binary needs, and records where they landed.
//
// Through genesis.Customize, the same merge-then-revalidate the network's own
// overlay goes through: a variant that reorders the forks has to fail here, at
// the step that wrote it, rather than at the boot of the nodes that read it.
//
// A variant naming a binary no node runs is refused. Writing it would leave a
// document nothing reads and say nothing, and the likeliest cause is a
// misspelled name — in which case the nodes that were meant to get it silently
// initialized from the network's genesis instead.
func (w *Workspace) writeGenesisVariants(ctx context.Context, lay node.Layout, base []byte, variants map[string][]byte) error {
	if len(variants) == 0 {
		w.state.GenesisPaths = nil
		return nil
	}
	paths := make(map[string]string, len(variants))
	for _, name := range slices.Sorted(maps.Keys(variants)) {
		if w.state.Binaries[name] == "" {
			return fmt.Errorf("chainsetup: genesis: a genesis is declared for binary %q, which no node runs — name one of the declared binaries", name)
		}
		gen, err := genesis.Customize(base, genesis.NetworkOptions{Overlay: variants[name]})
		if err != nil {
			return fmt.Errorf("chainsetup: genesis: binary %q: %w", name, err)
		}
		path := lay.GenesisVariantPath(name)
		err = w.eachMachine(func(t *resource.Access, _ []node.Record) error {
			ml := lay
			ml.Root = t.DataRoot
			p := ml.GenesisVariantPath(name)
			if werr := t.Files.Write(ctx, p, gen, 0o644); werr != nil {
				return fmt.Errorf("chainsetup: genesis: write %s: %w", p, werr)
			}
			w.recordInput(p, gen)
			return nil
		})
		if err != nil {
			return err
		}
		paths[name] = path
	}
	w.state.GenesisPaths = paths
	return nil
}

// delayedForkSuffix marks a config override that moves a fork off genesis. Such
// a network is advertised as delayed-<fork> so the fork-transition cases gate on
// it and skip on a normal network where the fork is active at genesis.
const delayedForkSuffix = "Block"

// networkCapabilities is what the composed network advertises: the chain's own
// capabilities, the ones its manifest data implies (its contracts, hardforks,
// engine, family and tx types — see registry.Manifest.DerivedCapabilities),
// "ws" (composed nodes always serve a WebSocket endpoint), a delayed-<fork>
// marker per fork moved off genesis, and whatever the caller declared for its
// overlay.
//
// The derived ones are what let a case ask for what it needs — a govMinter, the
// boho fork — instead of naming the chain it was written on. The chain's name
// is not one of them on purpose: a case that gates on a name has to be edited
// to meet a second chain, which is the thing being removed.
func networkCapabilities(m registry.Manifest, genesisTemplate []byte, opts GenesisOpts) []string {
	caps := append([]string(nil), m.Capabilities...)
	caps = append(caps, m.DerivedCapabilities(genesisTemplate)...)
	caps = append(caps, "ws")
	for _, key := range slices.Sorted(maps.Keys(opts.Overrides)) {
		fork, ok := strings.CutSuffix(key, delayedForkSuffix)
		if !ok || fork == "" {
			continue
		}
		n, err := strconv.Atoi(opts.Overrides[key])
		if err != nil {
			continue
		}
		// The override switches the fork on, whether at genesis or later, so
		// this network answers for it. Only a later one is also "delayed".
		caps = append(caps, registry.CapFork+strings.ToLower(fork))
		if n > 0 {
			caps = append(caps, "delayed-"+strings.ToLower(fork))
		}
	}
	// An overlay carries a fork the same way the template does — as a block
	// number or as an engine section — and a network built from one answers for
	// it however it arrived. Without this the capability depended on which of
	// the three routes a spec happened to take.
	for _, fork := range m.Genesis.Hardforks {
		if registry.ForkActiveIn(opts.Overlay, fork) && len(opts.Overlay) > 0 {
			caps = append(caps, registry.CapFork+fork)
		}
	}
	return append(caps, opts.Capabilities...)
}

// Config renders each node's TOML config and writes it to the target.
