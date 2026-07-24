// Package wbft registers the go-wbft chain plugin: the wbft consensus family +
// the wbft accounts protocol + the wbft manifest. Importing it for side effects
// registers the chain.
package wbft

import (
	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/manifests"
	wbftfam "github.com/0xmhha/chainbench/pkg/consensus/wbft"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

type plugin struct{ m registry.Manifest }

func init() {
	raw, err := manifests.Raw("wbft")
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
func (p plugin) Family() registry.ConsensusFamily { return wbftfam.New() }
func (p plugin) Protocol() protocol.Protocol      { return protocol.WBFT() }
