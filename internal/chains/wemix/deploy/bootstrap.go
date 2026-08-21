package deploy

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// wemixConfigPerm is the mode of the governance config shipped to the boot
// producer. It carries members and stakes, which are on-chain anyway, so it is
// world-readable — unlike key material, which the file store writes at 0600.
const wemixConfigPerm fs.FileMode = 0o644

// poaCommand renders the shell command a poa.Runner step runs on the remote
// host: the binary followed by its (shell-quoted) args.
func poaCommand(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(name))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// SSHPoaRunner returns a poa.Runner that runs each binary+args over SSH on the
// given host — the remote equivalent of the local execRunner that backs
// poa.DeployGovernance / poa.EtcdInit.
func SSHPoaRunner(rc remote.Credentials, hostKey remote.HostKeyCallback) poa.Runner {
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		res, err := remote.Exec(ctx, rc, hostKey, poaCommand(name, args))
		if err != nil {
			return []byte(res.Stdout + res.Stderr), err
		}
		return []byte(res.Stdout + res.Stderr), nil
	}
}

// BootProducer returns the boot producer (the first wemix_bp server, which
// deploys governance and initializes etcd), or false when there is none.
func BootProducer(c *Cluster) (Server, bool) {
	prod := c.Producers()
	if len(prod) == 0 {
		return Server{}, false
	}
	return prod[0], true
}

// Bootstrap performs the governance + etcd bring-up on the cluster's boot
// producer over SSH: it ships the wemix governance config, deploys the
// governance contracts, and initializes the embedded etcd cluster. The boot
// producer must already be running (Deploy) so its IPC is live. gwemix embeds
// etcd — no external etcd is contacted.
func Bootstrap(ctx context.Context, c *Cluster, cr *Credentials, hostKey remote.HostKeyCallback, cfg poa.Config, env func(string) string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	boot, ok := BootProducer(c)
	if !ok {
		return fmt.Errorf("deploy: cluster has no wemix_bp producer to bootstrap")
	}
	rc, err := cr.For(c, boot, env)
	if err != nil {
		return err
	}

	dd := c.dataRoot()
	p := c.Paths()
	configPath := path.Join(dd, "wemix-config.json")
	ipc := path.Join(dd, "gwemix.ipc")
	passwordPath := path.Join(dd, "conf", "keystore", ".password")
	binary := c.BinaryFor(boot)

	// Ship the governance config to the boot producer. It is not secret — the
	// members and their stakes are on-chain — so it is world-readable.
	cfgJSON, err := cfg.JSON()
	if err != nil {
		return err
	}
	if err := serverFiles(rc, hostKey).Write(ctx, configPath, cfgJSON, wemixConfigPerm); err != nil {
		return fmt.Errorf("deploy: ship wemix config: %w", err)
	}

	runner := SSHPoaRunner(rc, hostKey)
	if err := poa.DeployGovernance(ctx, runner, binary, ipc, configPath, p.CoinbaseKeystore, passwordPath); err != nil {
		return err
	}
	return poa.EtcdInit(ctx, runner, binary, ipc)
}
