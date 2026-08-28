package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// liveHandoff drives the real handoff through upgrade.Handoff, which owns the
// sequence's body. What is left here is the step seam: each HandoffDriver
// method is one step the case runner reports.
type liveHandoff struct {
	h *upgrade.Handoff
}

// NewLiveHandoff returns a HandoffDriver backed by the real binaries.
func NewLiveHandoff() HandoffDriver { return &liveHandoff{} }

func (l *liveHandoff) Prepare(_ context.Context, o HandoffOptions) (string, error) {
	h, err := upgrade.NewHandoff(upgrade.HandoffInputs{
		ProfilePath: o.ProfilePath, PresetDir: o.PresetDir,
		FromBinary: o.FromBinary, ToBinary: o.ToBinary,
		Template: o.Template, GenesisOverlay: o.GenesisOverlay,
		DataDir: o.DataDir, Exec: o.Exec, Files: o.Files,
	})
	if err != nil {
		return "", err
	}
	l.h = h
	return h.Describe(), nil
}

func (l *liveHandoff) Config(ctx context.Context, _ HandoffOptions) (string, error) {
	return l.h.WriteConfig(ctx)
}

func (l *liveHandoff) BaseGenesis(ctx context.Context, _ HandoffOptions, _ string) (string, error) {
	return l.h.BaseGenesis(ctx)
}

func (l *liveHandoff) Plan(_ context.Context, _ HandoffOptions, basePath string) (string, error) {
	if err := l.h.ComposePlan(basePath); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d node(s); fork section %q merged; preflight passed", len(l.h.Plan.Nodes), l.h.Plan.AtFork), nil
}

func (l *liveHandoff) Overlay(_ context.Context, _ HandoffOptions) (string, error) {
	return l.h.ApplyOverlay()
}

func (l *liveHandoff) Launch(ctx context.Context, _ HandoffOptions) (node.NodeSet, error) {
	return l.h.Launch(ctx)
}

func (l *liveHandoff) WireMesh(ctx context.Context, ns node.NodeSet) (string, error) {
	if err := l.h.WireMesh(ctx, ns); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d endpoint(s) meshed", len(ns.Nodes)), nil
}

func (l *liveHandoff) DeployGovernance(ctx context.Context, _ HandoffOptions, producer node.Node) (string, error) {
	if err := l.h.DeployGovernance(ctx, producer); err != nil {
		return "", err
	}
	return "deploy-governance returned success (effect is checked by verify-etcd)", nil
}

func (l *liveHandoff) EtcdInit(ctx context.Context, _ HandoffOptions, producer node.Node) (string, error) {
	if err := l.h.EtcdInit(ctx, producer); err != nil {
		return "", err
	}
	return "admin.etcdInit() returned without error (effect is checked by verify-etcd)", nil
}

func (l *liveHandoff) ProducerIPC(_ HandoffOptions, producer node.Node) string {
	return l.h.ProducerIPC(producer)
}

func (l *liveHandoff) AwaitFork(ctx context.Context, ns node.NodeSet, o HandoffOptions) (string, error) {
	return l.h.AwaitFork(ctx, ns, o.ForkTimeout)
}
