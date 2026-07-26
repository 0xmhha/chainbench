package main

import (
	"github.com/0xmhha/chainbench/pkg/accounts"
)

// resolveAccountProvider returns the accounts provider for a run: from an
// external manifest's borrowed protocol when manifestPath is set (the hybrid
// model, so faucet/tx work on a project-supplied chain), otherwise the embedded
// chain's SDK protocol by id. It shares resolveChain (setup.go) for the external
// path.
func resolveAccountProvider(chain, manifestPath, templatePath string) (accounts.AccountProvider, error) {
	if manifestPath != "" {
		p, err := resolveChain(chain, manifestPath, templatePath)
		if err != nil {
			return nil, err
		}
		return accounts.New(p.Protocol()), nil
	}
	return accounts.ForChain(chain)
}
