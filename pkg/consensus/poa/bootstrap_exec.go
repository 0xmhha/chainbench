package poa

import (
	"context"
	"fmt"
)

// Runner runs a command and returns its combined output. It is injected so the
// bootstrap steps (which shell out to the gwemix binary) are testable without a
// real binary.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// GenerateGenesis materializes a wemix genesis from a governance config and a
// template via `gwemix wemix genesis`. The wemix genesis (extraData bootnode
// encoding, alloc, wemix fork config) is produced by the binary, not in Go; the
// caller then injects the croissant section (genesis.InjectCroissant) for the
// upgrade.
func GenerateGenesis(ctx context.Context, r Runner, binary, configPath, templatePath, outPath string) error {
	out, err := r(ctx, binary, "wemix", "genesis", "--data", configPath, "--genesis", templatePath, "--out", outPath)
	if err != nil {
		return fmt.Errorf("poa: wemix genesis: %w: %s", err, out)
	}
	return nil
}

// DeployGovernance deploys the governance contracts through the running boot
// node's IPC. It uses the TWO-argument form (config + account, no lockAmount):
// the three-argument form is broken in the current gwemix (it shadows lockAmount
// with :=, passing nil and panicking), so the amount is left to default to
// STAKING_MIN.
func DeployGovernance(ctx context.Context, r Runner, binary, ipc, configPath, keystorePath, passwordPath string) error {
	out, err := r(ctx, binary, "wemix", "deploy-governance",
		"--url", ipc, "--password", passwordPath, configPath, keystorePath)
	if err != nil {
		return fmt.Errorf("poa: deploy-governance: %w: %s", err, out)
	}
	return nil
}

// EtcdInit initializes the etcd cluster on the boot node via its IPC console.
// After this the registered member nodes join the cluster and the wemix
// producer rotation begins.
func EtcdInit(ctx context.Context, r Runner, binary, ipc string) error {
	out, err := r(ctx, binary, "attach", ipc, "--exec", "admin.etcdInit()")
	if err != nil {
		return fmt.Errorf("poa: etcdInit: %w: %s", err, out)
	}
	return nil
}
