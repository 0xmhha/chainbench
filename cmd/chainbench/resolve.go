package main

import (
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/netmap"
)

// resolveAccountProvider returns the accounts provider for a run: from an
// external manifest's borrowed protocol when manifestPath is set (the hybrid
// model, so faucet/tx work on a project-supplied chain), otherwise the embedded
// chain's SDK protocol by id. It shares resolveChain (setup.go) for the external
// path.
func resolveAccountProvider(chain, manifestPath, templatePath string) (accounts.AccountProvider, error) {
	if manifestPath != "" {
		p, err := app.ResolveChain(chain, manifestPath, templatePath)
		if err != nil {
			return nil, err
		}
		return accounts.New(p.Protocol()), nil
	}
	return accounts.ForChain(chain)
}

// remoteDriver builds an SSH RemoteDriver from the --remote-* flags, so setup can
// provision and launch on a remote host. The SSH password comes only from the
// CHAINBENCH_REMOTE_PASS env var — never a flag — so it is not exposed in the
// process list or shell history. The host-key policy is resolved from the
// server set's ssh block (known_hosts_file, or insecure_host_key on a closed
// network). Returns nil when host is empty (the local driver is used).
func remoteDriver(host, user string, port int) (driver.Driver, error) {
	if host == "" {
		return nil, nil
	}
	pass := os.Getenv(remote.EnvPass)
	if pass == "" {
		return nil, fmt.Errorf("remote setup needs the SSH password in %s (do not pass it on the command line)", remote.EnvPass)
	}
	if port == 0 {
		port = 22
	}
	hostKey, err := netmap.SetPolicy("").Callback()
	if err != nil {
		return nil, err
	}
	creds := remote.Credentials{User: user, Host: host, Port: port, Password: pass}
	return driver.NewRemoteDriver(driver.SSHRunner(creds, hostKey)), nil
}
