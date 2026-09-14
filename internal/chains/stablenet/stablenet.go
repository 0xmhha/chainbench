// Package stablenet composes the go-stablenet chain: the wbft consensus family,
// the stablenet accounts protocol, and the manifest and genesis template
// embedded from this folder. Importing it for side effects registers the chain
// (and, via caps.go, its capabilities).
//
// Everything this chain IS, is in this folder: the four choices are in the
// literal below and the chain's constants are in manifest.json beside it.
package stablenet

import (
	_ "embed"

	"github.com/0xmhha/accounts/protocol"

	"github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

//go:embed manifest.json
var manifestJSON []byte

//go:embed genesis.json
var genesisTmpl []byte

func init() {
	registry.Register(registry.StaticPlugin{
		M:     registry.MustParseManifest(manifestJSON),
		Fam:   wbft.New(),
		Proto: protocol.StableNet(),
		Tmpl:  genesisTmpl,
	})
}
