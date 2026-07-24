// Package wemix registers the go-wemix chain plugin: the poa consensus family +
// the wemix accounts protocol + the wemix manifest. wemix genesis is deploy-time
// (etcd/registry, no static template — G7), so GenesisTemplate is nil.
// Importing it for side effects registers the chain.
package wemix

import (
	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/manifests"
	"github.com/0xmhha/chainbench/pkg/consensus/poa"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

type plugin struct {
	m    registry.Manifest
	tmpl []byte
}

func init() {
	raw, err := manifests.Raw("wemix")
	if err != nil {
		panic(err)
	}
	m, err := registry.ParseManifest(raw)
	if err != nil {
		panic(err)
	}
	var tmpl []byte
	if m.Genesis.Template != "" {
		if tmpl, err = manifests.GenesisTemplate(m.Genesis.Template); err != nil {
			panic(err)
		}
	}
	registry.Register(plugin{m: m, tmpl: tmpl})
}

func (p plugin) Manifest() registry.Manifest      { return p.m }
func (p plugin) Family() registry.ConsensusFamily { return poa.New() }
func (p plugin) Protocol() protocol.Protocol      { return protocol.WeMix() }
func (p plugin) GenesisTemplate() []byte          { return p.tmpl }
