package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InitDatadir writes the genesis into the node's data dir and runs the datadir
// init locally, satisfying the Initializer capability. The genesis is placed at
// <dataDir>/genesis.json so the step is self-contained per node (the same shape
// a remote driver ships).
func (d *LocalDriver) InitDatadir(ctx context.Context, spec NodeSpec, genesis []byte) error {
	if err := os.MkdirAll(spec.DataDir, 0o755); err != nil {
		return fmt.Errorf("driver: mkdir datadir: %w", err)
	}
	genesisPath := filepath.Join(spec.DataDir, "genesis.json")
	if err := os.WriteFile(genesisPath, genesis, 0o644); err != nil {
		return fmt.Errorf("driver: write genesis: %w", err)
	}
	return InitDatadir(ctx, spec.Binary, spec.DataDir, genesisPath)
}

// InitDatadir initializes a node's data directory from a genesis file by running
// the geth-family `<binary> init --datadir <dataDir> <genesisPath>` command and
// waiting for it to complete. It is the "environment build" step that must run
// before Launch. Unlike Launch, this is a short-lived command, so it uses
// os/exec directly and returns the combined output on failure.
func InitDatadir(ctx context.Context, binary, dataDir, genesisPath string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("driver: mkdir datadir: %w", err)
	}
	cmd := exec.CommandContext(ctx, binary, "init", "--datadir", dataDir, genesisPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("driver: %q init %s failed: %w: %s", binary, dataDir, err, out)
	}
	return nil
}
