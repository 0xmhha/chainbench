// Package stablenet registers the go-stablenet chain plugin: the wbft consensus
// family + the stablenet accounts protocol + the stablenet manifest and genesis
// template (anzeon), embedded from this folder. It is a thin composition layer
// (decision D9) — importing it for side effects registers the chain (and, via
// caps.go, its capabilities).
package stablenet

import (
	_ "embed"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/pkg/consensus/wbft"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

//go:embed manifest.json
var manifestJSON []byte

//go:embed genesis.json
var genesisTmpl []byte

type plugin struct{ m registry.Manifest }

func init() {
	m, err := registry.ParseManifest(manifestJSON)
	if err != nil {
		panic(err)
	}
	registry.Register(plugin{m: m})
}

func (p plugin) Manifest() registry.Manifest      { return p.m }
func (p plugin) Family() registry.ConsensusFamily { return wbft.New() }
func (p plugin) Protocol() protocol.Protocol      { return protocol.StableNet() }
func (p plugin) GenesisTemplate() []byte          { return genesisTmpl }
