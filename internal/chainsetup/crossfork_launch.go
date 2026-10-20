package chainsetup

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
)

func (w *Workspace) crossForkNodeArgs(ns node.Record, spec nodeconfig.Spec) ([]string, error) {
	overrides, err := ParseOverrides(w.launchOverridesFor(ns.Role, ns.Index))
	if err != nil {
		return nil, err
	}
	return w.nodeLaunchArgs(ns, spec, overrides)
}
