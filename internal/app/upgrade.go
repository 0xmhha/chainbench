package app

import (
	"context"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// defaultEtcdFormTimeout bounds the wait for the producer's etcd cluster to
// form when the caller does not set one.
const defaultEtcdFormTimeout = 60 * time.Second

// UpgradeRunIn shapes a profile-based consensus handoff (the CLI `upgrade run`).
type UpgradeRunIn struct {
	ProfilePath    string
	PresetDir      string
	FromBinary     string
	ToBinary       string
	Template       string
	GenesisOverlay string
	DataDir        string
	// EtcdTimeout bounds the etcd-cluster-form wait; zero uses the default.
	EtcdTimeout time.Duration
}

// UpgradeRunOut is the handoff result.
type UpgradeRunOut struct {
	Nodes      node.NodeSet
	Governance string
	Cluster    string
}

// UpgradeRun performs a profile-based consensus handoff (go-wemix -> go-wbft at
// a fork), the same sequence the CLI `upgrade run` drives, so the MCP surface
// reaches it too. It wraps upgrade.NewHandoff and the shared Handoff.Run.
func UpgradeRun(ctx context.Context, _ Deps, in UpgradeRunIn) (UpgradeRunOut, error) {
	h, err := upgrade.NewHandoff(upgrade.HandoffInputs{
		ProfilePath:    in.ProfilePath,
		PresetDir:      in.PresetDir,
		FromBinary:     in.FromBinary,
		ToBinary:       in.ToBinary,
		Template:       in.Template,
		GenesisOverlay: in.GenesisOverlay,
		DataDir:        in.DataDir,
	})
	if err != nil {
		return UpgradeRunOut{}, err
	}
	timeout := in.EtcdTimeout
	if timeout <= 0 {
		timeout = defaultEtcdFormTimeout
	}
	ns, info, err := h.Run(ctx, timeout)
	if err != nil {
		return UpgradeRunOut{Nodes: ns}, err
	}
	return UpgradeRunOut{Nodes: ns, Governance: info.Governance, Cluster: info.Cluster()}, nil
}
