package netcompose

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/chains/external"
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
	Target TargetSpec
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

	target := opts.Target
	if target.Kind == "" {
		target.Kind = TargetLocal
	}
	if target.Kind == TargetLocal && target.DataRoot == "" {
		target.DataRoot = w.comp.Dir()
	}
	// Validate the target resolves (remote needs host + reachable auth later,
	// but structural validation happens here); env is nil so no live SSH dial.
	if target.IsRemote() && (target.Host == "" || target.DataRoot == "") {
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
	w.state.Target = target

	var loc string
	if target.IsRemote() {
		loc = fmt.Sprintf("remote %s@%s:%s", target.User, target.Host, target.DataRoot)
	} else {
		loc = fmt.Sprintf("local %s", target.DataRoot)
	}
	detail := fmt.Sprintf("%s: family %s, chain id %d, bootstrap %s; keys %s; target %s",
		m.ID, m.ConsensusFamily, m.ChainID, m.Bootstrap.Type, keysDir, loc)
	w.markStep("new", detail)
	return detail, nil
}
