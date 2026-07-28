// Package wemix registers the go-wemix chain plugin: the poa consensus family +
// the wemix accounts protocol + the wemix manifest and base genesis template,
// embedded from this folder. The validator set is not in the genesis (poa/etcd
// membership is set at bootstrap). Importing it for side effects registers the
// chain (and, via caps.go, its capabilities).
package wemix

import (
	_ "embed"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/pkg/consensus/poa"
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
func (p plugin) Family() registry.ConsensusFamily { return poa.New() }
func (p plugin) Protocol() protocol.Protocol      { return protocol.WeMix() }
func (p plugin) GenesisTemplate() []byte          { return genesisTmpl }
