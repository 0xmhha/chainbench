package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// planSuiteThroughCLI uses exactly the input projection executed by run. Only
// the private case files are temporary; content-addressed overlays survive
// beneath the owned planning directory for the accepted plan to reuse.
func planSuiteThroughCLI(ctx context.Context, cli ChainCLI, in RunSuiteIn, parent string) (layout ChainUpIn, plan ComposePlan, err error) {
	if cli == nil {
		return layout, plan, errNoChainCLI
	}
	dir, err := os.MkdirTemp(parent, "plan-")
	if err != nil {
		return layout, plan, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	args, err := suiteCLIArgs(ctx, in, dir, "")
	if err != nil {
		return layout, plan, err
	}
	raw, err := cli.Run(ctx, append(args, "--plan")...)
	if err != nil {
		return layout, plan, err
	}
	var reports []SuitePlan
	if err = json.Unmarshal(raw, &reports); err != nil {
		return layout, plan, fmt.Errorf("chainbench run plan: %w", err)
	}
	if len(reports) != 1 || reports[0].Plan.Chain == "" || reports[0].Layout.Chain == "" {
		return layout, plan, errors.New("chainbench run plan: incomplete suite layout")
	}
	return reports[0].Layout, reports[0].Plan, nil
}

func planPresetThroughCLI(ctx context.Context, cli ChainCLI, raw []byte, in RunSuiteIn) (out PresetPlan, err error) {
	if cli == nil {
		return out, errNoChainCLI
	}
	dir, err := os.MkdirTemp(filepath.Dir(in.DataDir), "preset-plan-")
	if err != nil {
		return out, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	path := filepath.Join(dir, "preset.json")
	if err = (filestore.Local{}).Write(ctx, path, raw, 0600); err != nil {
		return out, err
	}
	args := []string{"chain", "up", "--chain-preset", path, "--plan", "--json", "--workspace-dir", in.DataDir}
	for _, f := range []struct{ flag, value string }{{"--chain", in.Chain}, {"--binary", in.Binary}, {"--keys", in.KeysDir}, {"--workspace-config", in.WorkspaceConfigPath}, {"--server-set", in.Server.SetPath}, {"--server", in.Server.Name}} {
		if f.value != "" {
			args = append(args, f.flag, f.value)
		}
	}
	if in.Docker {
		args = append(args, "--docker")
	}
	stdout, err := cli.Run(ctx, args...)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(stdout, &out); err != nil {
		return out, fmt.Errorf("chainbench preset plan: %w", err)
	}
	if out.Plan.Chain == "" || out.Layout.Chain == "" {
		return out, errors.New("chainbench preset plan: incomplete layout")
	}
	return out, nil
}
