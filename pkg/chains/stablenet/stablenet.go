// Package stablenet registers the go-stablenet chain plugin: the wbft
// consensus family + the stablenet accounts protocol + the stablenet manifest.
// It is a thin composition layer (decision D9) — importing it for side effects
// registers the chain.
package stablenet

import (
	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/manifests"
	"github.com/0xmhha/chainbench/pkg/consensus/wbft"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

type plugin struct{ m registry.Manifest }

func init() {
	raw, err := manifests.Raw("stablenet")
	if err != nil {
		panic(err)
	}
	m, err := registry.ParseManifest(raw)
	if err != nil {
		panic(err)
	}
	registry.Register(plugin{m: m})
}

func (p plugin) Manifest() registry.Manifest      { return p.m }
func (p plugin) Family() registry.ConsensusFamily { return wbft.New() }
func (p plugin) Protocol() protocol.Protocol      { return protocol.StableNet() }
