package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// ipcWait is how long a bootstrap action waits for the node's IPC socket. The
// steps run over IPC rather than HTTP, and the socket appears some way into
// startup.
const ipcWait = 30 * time.Second

// WemixBootstrap executes the bring-up actions a poa network names between its
// phases: deploy the governance contracts, form the etcd cluster, and confirm
// it formed.
//
// It is the executor side of the seam the family declares. The family says what
// must happen and in what order; this says how, for one particular target. The
// supervisor owns when, how long, and how a failure is classified.
type WemixBootstrap struct {
	// Binary is the gwemix executable the actions drive.
	Binary string
	// KeysDir holds the key set: the boot node's keystore and the password.
	KeysDir string
	// ConfigName is the governance config's name on the target, written beside
	// the genesis by the genesis step. deploy-governance reads it back rather
	// than rebuilding it, so the deploy and the genesis cannot disagree. Empty
	// takes the name the genesis source writes.
	ConfigName string
	// Run executes the binary; nil uses os/exec.
	Run poa.Runner
}

// Action performs one named bring-up action against a node.
func (b WemixBootstrap) Action(ctx context.Context, name string, plan driver.Plan, on node.Node) error {
	spec, ok := specFor(plan, on.Index)
	if !ok {
		return fmt.Errorf("engine: wemix bootstrap: the plan has no node%d to run %q on", on.Index, name)
	}
	run := b.Run
	if run == nil {
		run = execRunner
	}
	binary := b.Binary
	if binary == "" {
		binary = spec.Binary
	}
	if binary == "" {
		return fmt.Errorf("engine: wemix bootstrap: no binary to run %q with", name)
	}
	ipc := ipcPath(spec, binary)
	if err := poa.WaitForIPC(ctx, ipc, ipcWait); err != nil {
		return fmt.Errorf("engine: wemix bootstrap: %q: %w", name, err)
	}

	switch name {
	case poa.ActionDeployGovernance:
		keystore, err := bootKeystore(b.KeysDir, on.Index)
		if err != nil {
			return fmt.Errorf("engine: wemix bootstrap: %q: %w", name, err)
		}
		password := filepath.Join(b.KeysDir, "password")
		cfgName := b.ConfigName
		if cfgName == "" {
			cfgName = wemixConfigName
		}
		cfgPath := filepath.Join(plan.DataRoot, cfgName)
		if _, err := os.Stat(cfgPath); err != nil {
			return fmt.Errorf("engine: wemix bootstrap: %q needs the governance config the genesis step writes to the target: %w", name, err)
		}
		return poa.DeployGovernance(ctx, run, binary, ipc, cfgPath, keystore, password)
	case poa.ActionEtcdInit:
		return poa.EtcdInit(ctx, run, binary, ipc)
	case poa.ActionVerifyEtcd:
		return poa.VerifyEtcd(ctx, run, binary, ipc)
	default:
		// An action nobody implements is a gap in the bring-up, not something
		// to skip: the phase that named it expects it to have happened.
		return fmt.Errorf("engine: wemix bootstrap: no executor for action %q", name)
	}
}

// specFor finds a node's launch spec, which is where its datadir lives.
func specFor(plan driver.Plan, index int) (driver.NodeSpec, bool) {
	for _, s := range plan.Nodes {
		if s.Index == index {
			return s, true
		}
	}
	return driver.NodeSpec{}, false
}

// ipcPath is where the node exposes its console socket: inside its datadir,
// named after the binary.
func ipcPath(spec driver.NodeSpec, binary string) string {
	return filepath.Join(spec.DataDir, filepath.Base(binary)+".ipc")
}

// bootKeystore is the boot node's keystore file. deploy-governance signs with
// it, so the exact file is needed rather than the directory.
func bootKeystore(keysDir string, index int) (string, error) {
	dir := filepath.Join(keysDir, fmt.Sprintf("node%d", index), "keystore")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("keystore for node%d: %w", index, err)
	}
	for _, e := range ents {
		if !e.IsDir() {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("keystore for node%d is empty (%s)", index, dir)
}
