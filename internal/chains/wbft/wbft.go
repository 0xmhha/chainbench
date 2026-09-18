// Package wbft composes the go-wbft chain: the wbft consensus family, the wbft
// accounts protocol, and the manifest and genesis template embedded from this
// folder. Importing it for side effects registers the chain.
//
// It composes the same consensus as stablenet and differs in its protocol, its
// genesis template and its constants — which is the point of composing rather
// than inheriting: the shared half is one implementation, named here.
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

func init() {
	registry.Register(registry.StaticPlugin{
		M:     registry.MustParseManifest(manifestJSON),
		Fam:   wbftfam.New(),
		Proto: protocol.WBFT(),
		Tmpl:  genesisTmpl,
	})
}
