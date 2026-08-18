package main

import (
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/remote"
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

// remoteDriver builds an SSH RemoteDriver and a matching remote FileSink from the
// --remote-* flags, so setup can provision and launch on a remote host: the
// driver runs init/launch, the sink ships genesis and per-node config over the
// same transport. The SSH password comes only from the CHAINBENCH_REMOTE_PASS env
// var — never a flag — so it is not exposed in the process list or shell history.
// The host-key policy is resolved from the standard SSH env
// (CHAINBENCH_SSH_KNOWN_HOSTS, or CHAINBENCH_SSH_INSECURE_HOST_KEY=1 for a
// throwaway host). Returns nil, nil when host is empty (local driver + local
// filesystem are used).
func remoteDriver(host, user string, port int) (driver.Driver, provision.FileSink, error) {
	if host == "" {
		return nil, nil, nil
	}
	pass := os.Getenv("CHAINBENCH_REMOTE_PASS")
	if pass == "" {
		return nil, nil, fmt.Errorf("remote setup needs the SSH password in CHAINBENCH_REMOTE_PASS (do not pass it on the command line)")
	}
	if port == 0 {
		port = 22
	}
	hostKey, err := remote.ResolveHostKeyCallback(os.Getenv)
	if err != nil {
		return nil, nil, err
	}
	creds := remote.Credentials{User: user, Host: host, Port: port, Password: pass}
	run := driver.SSHRunner(creds, hostKey)
	return driver.NewRemoteDriver(run), driver.NewRemoteFileSink(run), nil
}
