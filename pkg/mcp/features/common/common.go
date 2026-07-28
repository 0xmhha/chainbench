// Package common implements the chainbench capabilities shared by every chain
// (the "common" project). Importing it for side effects loads its catalog
// (common.jsonl) and registers its handlers into the capability registry.
package common

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/mcp/capability"
)

//go:embed common.jsonl
var catalog []byte

func init() {
	if err := capability.LoadCatalog(catalog); err != nil {
		panic(err)
	}
	capability.RegisterHandler("v1", capability.CommonChain, "chains.list", chainsList)
	capability.RegisterHandler("v1", capability.CommonChain, "chains.info", chainsInfo)
	capability.RegisterHandler("v1", capability.CommonChain, "chains.hardforks", chainsHardforks)
}

func chainsList(_ context.Context, _ map[string]any) (string, error) {
	var b strings.Builder
	for _, id := range registry.Names() {
		p, err := registry.Get(id)
		if err != nil {
			return "", err
		}
		m := p.Manifest()
		fmt.Fprintf(&b, "%s\tfamily=%s\tbinary=%s\tchain_id=%d\tnamespace=%s\n",
			m.ID, m.ConsensusFamily, m.Binary, m.ChainID, m.Consensus.RPCNamespace)
	}
	return b.String(), nil
}

func chainsInfo(_ context.Context, args map[string]any) (string, error) {
	id := capability.ArgString(args, "chain", "")
	if id == "" {
		return "", fmt.Errorf("chain is required")
	}
	p, err := registry.Get(id)
	if err != nil {
		return "", err
	}
	m := p.Manifest()
	var b strings.Builder
	fmt.Fprintf(&b, "chain=%s\nfamily=%s\nbinary=%s\nchain_id=%d\nnamespace=%s\ntx_types=%s\ncapabilities=%s\n",
		m.ID, m.ConsensusFamily, m.Binary, m.ChainID, m.Consensus.RPCNamespace,
		strings.Join(m.TxTypes, ","), strings.Join(m.Capabilities, ","))
	return b.String(), nil
}

func chainsHardforks(_ context.Context, args map[string]any) (string, error) {
	id := capability.ArgString(args, "chain", "")
	if id == "" {
		return "", fmt.Errorf("chain is required")
	}
	p, err := registry.Get(id)
	if err != nil {
		return "", err
	}
	forks := p.Manifest().Genesis.Hardforks
	if len(forks) == 0 {
		return fmt.Sprintf("%s: no hardforks declared", id), nil
	}
	return fmt.Sprintf("%s hardforks: %s", id, strings.Join(forks, " -> ")), nil
}
