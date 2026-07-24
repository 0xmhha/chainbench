// Package wemix registers the go-wemix chain plugin: the poa consensus family +
// the wemix accounts protocol + the wemix manifest. Importing it for side
// effects registers the chain.
package wemix

import (
	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/manifests"
	"github.com/0xmhha/chainbench/pkg/consensus/poa"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

type plugin struct{ m registry.Manifest }

func init() {
	raw, err := manifests.Raw("wemix")
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
func (p plugin) Family() registry.ConsensusFamily { return poa.New() }
func (p plugin) Protocol() protocol.Protocol      { return protocol.WeMix() }
