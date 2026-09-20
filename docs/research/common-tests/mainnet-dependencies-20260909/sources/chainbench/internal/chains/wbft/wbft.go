// Package wbft registers the go-wbft chain plugin: the wbft consensus family +
// the wbft accounts protocol + the wbft manifest and genesis template
// (croissant), embedded from this folder. Importing it for side effects
// registers the chain.
package wbft

import (
	_ "embed"

	"github.com/0xmhha/accounts/protocol"

	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/registry"
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
func (p plugin) Family() registry.ConsensusFamily { return wbftfam.New() }
func (p plugin) Protocol() protocol.Protocol      { return protocol.WBFT() }
func (p plugin) GenesisTemplate() []byte          { return genesisTmpl }
