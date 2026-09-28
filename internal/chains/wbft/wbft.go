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

// genesisTmpl carries a block gas limit of 105,000,000, the figure the real
// network runs with (params.WemixTestnetChainConfig in go-wbft). It also
// settles a drift: go-wbft's miner strives for a ceiling of its own
// (miner.Config GasCeil, also 105,000,000) and walks the limit toward it by a
// 1024th each block, so the 20,000,000 that stood here was never the limit the
// chain ran at, only the one it started from. Genesis and ceiling now agree.
// JSON cannot hold this note, so it lives here.
//
// The round timeout stays at 1, NOT at the 1000 the real network declares. The
// field is multiplied by 1000 into a millisecond RequestTimeout, so 1 is a
// one-second round-change timer (the engine's own default) and 1000 is a
// 1000-second one, which this genesis cannot cap because it leaves
// maxRequestTimeoutSeconds null and nothing caps round 0.
//
// A 1000-second timer does not merely slow fault recovery down; it stops a
// network from starting. Measured on 2026-09-28: node1 took round 0 and
// broadcast PRE-PREPARE for block 1 at 20:53:12, its peers connected 35
// seconds later, and with no round change due for 1000 seconds the network
// sealed nothing at all until the harness gave up. Four nodes, no blocks.
//
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
