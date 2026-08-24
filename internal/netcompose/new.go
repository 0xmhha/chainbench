package netcompose

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/target"
)

// NewOpts initializes a workspace's chain identity, key set, and compose target.
type NewOpts struct {
	Chain  string
	Binary string
	// ManifestPath selects an external, project-supplied chain manifest instead
	// of an embedded chain; TemplatePath is its genesis template. When set, the
	// chain id is taken from the manifest.
	ManifestPath string
	TemplatePath string
	// KeysDir is the key set the network composes from (default keys/preset).
	// Account management/inspection is the `account` subcommand's job; net only
	// records which key set to use.
	KeysDir string
	// Target is where the network's data plane lives. A zero Target defaults to
	// a local target whose data root is the workspace directory.
	Target target.TargetSpec
	// Docker treats the composition's servers as local docker containers: the
	// harness's dials are translated through the localmap next to the server
	// inventory. Recorded once here so every later step follows it.
	Docker bool
}

// New records the target chain, optional binary, key set, and compose target on
// the workspace, validating that the chain is a registered plugin. It is the
// first step; later steps read these from the workspace. A local target with no
// data root defaults to the workspace directory.
func (w *Workspace) New(opts NewOpts) (string, error) {
	if opts.Chain == "" && opts.ManifestPath == "" {
		return "", fmt.Errorf("netcompose: --chain or --manifest is required")
	}
	p, err := external.ResolveChain(opts.Chain, opts.ManifestPath, opts.TemplatePath)
	if err != nil {
		return "", err
	}
	keysDir := opts.KeysDir
	if keysDir == "" {
		keysDir = "keys/preset"
	}

	tgt := opts.Target
	if tgt.Kind == "" {
		tgt.Kind = target.TargetLocal
	}
	if tgt.Kind == target.TargetLocal && tgt.DataRoot == "" {
		tgt.DataRoot = w.comp.Dir()
	}
	// Validate the target resolves (remote needs host + reachable auth later,
	// but structural validation happens here); env is nil so no live SSH dial.
	if tgt.IsRemote() && (tgt.Host == "" || tgt.DataRoot == "") {
		return "", fmt.Errorf("netcompose: remote target needs --remote-host and --target-dir")
	}

	m := p.Manifest()
	// An external manifest names its own chain; recording it keeps status and
	// later steps reporting one id rather than an empty one.
	w.state.Chain = p.Manifest().ID
	w.state.ManifestPath = opts.ManifestPath
	w.state.TemplatePath = opts.TemplatePath
	w.state.Binary = opts.Binary
	w.state.KeysDir = keysDir
	w.state.Target = tgt
	w.state.Docker = opts.Docker

	var loc string
	if tgt.IsRemote() {
		loc = fmt.Sprintf("remote %s@%s:%s", tgt.User, tgt.Host, tgt.DataRoot)
	} else {
		loc = fmt.Sprintf("local %s", tgt.DataRoot)
	}
	if opts.Docker {
		loc += " (docker: dials translated via localmap)"
	}
	detail := fmt.Sprintf("%s: family %s, chain id %d, bootstrap %s; keys %s; target %s",
		m.ID, m.ConsensusFamily, m.ChainID, m.Bootstrap.Type, keysDir, loc)
	w.markStep("new", detail)
	return detail, nil
}

// Retarget replaces where the network's data plane lives, after `new` recorded
// a default. It exists for the server inventory: the entry an operator selects
// decides both the host and the data root, and that decision arrives with the
// placement rather than at `new`. A target with no data root keeps the current
// one, so naming only a host does not blank the path.
func (w *Workspace) Retarget(t target.TargetSpec) error {
	if t.Kind == "" {
		return nil
	}
	if t.DataRoot == "" {
		t.DataRoot = w.state.Target.DataRoot
	}
	if t.Kind == target.TargetRemote && (t.Host == "" || t.DataRoot == "") {
		return fmt.Errorf("netcompose: remote target needs a host and a data root")
	}
	w.state.Target = t
	return nil
}
