package netcompose

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// NewOpts initializes a workspace's chain identity and compose target.
type NewOpts struct {
	Chain  string
	Binary string
	// Target is where the network's data plane lives. A zero Target defaults to
	// a local target whose data root is the workspace directory.
	Target TargetSpec
}

// New records the target chain, optional binary, and compose target on the
// workspace, validating that the chain is a registered plugin. It is the first
// step; later steps read these from the workspace. A local target with no data
// root defaults to the workspace directory.
func (w *Workspace) New(opts NewOpts) (string, error) {
	if opts.Chain == "" {
		return "", fmt.Errorf("netcompose: --chain is required")
	}
	p, err := registry.Get(opts.Chain)
	if err != nil {
		return "", err
	}

	target := opts.Target
	if target.Kind == "" {
		target.Kind = TargetLocal
	}
	if target.Kind == TargetLocal && target.DataRoot == "" {
		target.DataRoot = w.dir
	}
	// Validate the target resolves (remote needs host + reachable auth later,
	// but structural validation happens here); env is nil so no live SSH dial.
	if target.IsRemote() && (target.Host == "" || target.DataRoot == "") {
		return "", fmt.Errorf("netcompose: remote target needs --remote-host and --target-dir")
	}

	m := p.Manifest()
	w.state.Chain = opts.Chain
	w.state.Binary = opts.Binary
	w.state.Target = target

	loc := string(target.Kind)
	if target.IsRemote() {
		loc = fmt.Sprintf("remote %s@%s:%s", target.User, target.Host, target.DataRoot)
	} else {
		loc = fmt.Sprintf("local %s", target.DataRoot)
	}
	detail := fmt.Sprintf("%s: family %s, chain id %d, bootstrap %s; target %s",
		m.ID, m.ConsensusFamily, m.ChainID, m.Bootstrap.Type, loc)
	w.markStep("new", detail)
	return detail, nil
}
